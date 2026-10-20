package cache

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Entry 缓存条目（缓存列表展示用）
type Entry struct {
	Key        string      // 缓存键（不含内部前缀）
	Value      interface{} // 缓存值
	Expiration int64       // 过期时间（毫秒时间戳），0 表示永不过期
}

// Stats 缓存统计信息（缓存监控展示用）
type Stats struct {
	Name            string        // 缓存名称
	Driver          string        // 后端类型：file / memory / memcache / redis
	KeyCount        int           // 有效缓存键数量
	KeysSupported   bool          // 后端是否支持枚举缓存键（memcache 不支持）
	Hits            uint64        // 命中次数
	Misses          uint64        // 未命中次数
	Sets            uint64        // 写入次数
	Deletes         uint64        // 删除次数
	Expired         uint64        // 过期失效次数（后端不支持主动清理时不统计）
	CleanupInterval time.Duration // 过期清理间隔
	CreatedAt       time.Time     // 创建时间
	Uptime          time.Duration // 运行时长
}

// Cache 缓存实例：统一封装多种后端（本地文件 / 内存 / memcache / redis）
//
// 键的实际存储形式为「配置前缀 + 缓存名 + : + 业务键」（如 admintpl:sys_config:sys.index.skinName），
// 这样多个缓存实例可以共用同一个 memcache/redis，且清空时只影响自己。
type Cache struct {
	name            string
	keyPrefix       string // 配置的全局键前缀（cache.prefix）
	cleanupInterval time.Duration
	createdAt       time.Time
	store           Store
	stop            chan struct{}
	once            sync.Once
	hits            uint64
	misses          uint64
	sets            uint64
	deletes         uint64
	expired         uint64
}

// New 按配置（cache.*）创建缓存实例
// @param: cleanupInterval time.Duration 过期清理间隔（<=0 表示不主动清理）
// @return: *Cache
func New(cleanupInterval time.Duration) *Cache {
	return newCache(cleanupInterval, LoadOptions().Prefix, sharedStore())
}

// newCache 使用指定后端创建缓存实例（测试与自定义后端用）
// @param: cleanupInterval time.Duration 过期清理间隔
// @param: prefix string 全局键前缀
// @param: store Store 缓存后端
// @return: *Cache
func newCache(cleanupInterval time.Duration, prefix string, store Store) *Cache {
	// 统一前缀格式为「prefix:」，避免出现 admintplsys_config:xxx 这样的粘连键
	if prefix = strings.TrimSuffix(strings.TrimSpace(prefix), ":"); prefix != "" {
		prefix += ":"
	}
	c := &Cache{
		keyPrefix:       prefix,
		cleanupInterval: cleanupInterval,
		createdAt:       time.Now(),
		store:           store,
		stop:            make(chan struct{}),
	}
	if cleanupInterval > 0 {
		go c.cleanupLoop(cleanupInterval)
	}
	return c
}

// cacheName 缓存名称（未注册时使用 default）
func (c *Cache) cacheName() string {
	if c.name == "" {
		return "default"
	}
	return c.name
}

// prefix 当前缓存实例的键前缀（含缓存名）
func (c *Cache) prefix() string { return c.keyPrefix + c.cacheName() + ":" }

// fullKey 拼接实际存储键
func (c *Cache) fullKey(key string) string { return c.prefix() + key }

// Set 写入缓存
// @param: key string 缓存键
// @param: value interface{} 缓存值
// @param: ttl time.Duration 有效期，<=0 表示永不过期
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixMilli()
	}
	payload, err := encodeValue(value)
	if err != nil {
		logrus.Warnf("缓存 %s 写入失败（%s）: %s", c.cacheName(), key, err)
		return
	}
	if err = c.store.Set(c.fullKey(key), payload, expiration); err != nil {
		logrus.Warnf("缓存 %s 写入失败（%s，后端 %s）: %s", c.cacheName(), key, c.store.Kind(), err)
		return
	}
	atomic.AddUint64(&c.sets, 1)
}

// Get 读取缓存（过期自动失效）
// @param: key string 缓存键
// @return: interface{} 缓存值, bool 是否存在
func (c *Cache) Get(key string) (interface{}, bool) {
	payload, err := c.store.Get(c.fullKey(key))
	if err != nil {
		if errors.Is(err, ErrExpired) {
			atomic.AddUint64(&c.expired, 1)
		} else if !errors.Is(err, ErrNotFound) {
			logrus.Debugf("缓存 %s 读取失败（%s，后端 %s）: %s", c.cacheName(), key, c.store.Kind(), err)
		}
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}
	value, err := decodeValue(payload)
	if err != nil {
		logrus.Warnf("缓存 %s 读取失败（%s）: %s", c.cacheName(), key, err)
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}
	atomic.AddUint64(&c.hits, 1)
	return value, true
}

// Delete 删除缓存
// @param: key string 缓存键
func (c *Cache) Delete(key string) {
	if _, err := c.store.Get(c.fullKey(key)); err == nil {
		atomic.AddUint64(&c.deletes, 1)
	}
	if err := c.store.Delete(c.fullKey(key)); err != nil {
		logrus.Debugf("缓存 %s 删除失败（%s，后端 %s）: %s", c.cacheName(), key, c.store.Kind(), err)
	}
}

// Count 当前有效缓存项数量（后端不支持枚举时返回 0）
// @return: int
func (c *Cache) Count() int {
	entries, err := c.store.Entries(c.prefix())
	if err != nil {
		return 0
	}
	return len(entries)
}

// Keys 当前有效缓存键（按字典序排序）
// @return: []string
func (c *Cache) Keys() []string {
	entries := c.rawEntries()
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, trimPrefix(entry.Key, c.prefix()))
	}
	sort.Strings(keys)
	return keys
}

// Entries 当前有效缓存条目（按缓存键排序，缓存列表展示用）
// @return: []Entry
func (c *Cache) Entries() []Entry {
	entries := c.rawEntries()
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		value, err := decodeValue(entry.Payload)
		if err != nil {
			logrus.Debugf("缓存 %s 值解析失败（%s）: %s", c.cacheName(), entry.Key, err)
			continue
		}
		result = append(result, Entry{
			Key:        trimPrefix(entry.Key, c.prefix()),
			Value:      value,
			Expiration: entry.Expiration,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

// rawEntries 后端原始条目（后端不支持枚举时返回空集合）
func (c *Cache) rawEntries() []RawEntry {
	entries, err := c.store.Entries(c.prefix())
	if err != nil {
		if !errors.Is(err, ErrUnsupported) {
			logrus.Debugf("缓存 %s 枚举失败（后端 %s）: %s", c.cacheName(), c.store.Kind(), err)
		}
		return nil
	}
	return entries
}

// Clear 清空缓存
func (c *Cache) Clear() {
	if err := c.store.Clear(c.prefix()); err != nil {
		logrus.Warnf("缓存 %s 清空失败（后端 %s）: %s", c.cacheName(), c.store.Kind(), err)
	}
}

// Stats 缓存统计信息
// @return: Stats
func (c *Cache) Stats() Stats {
	entries, err := c.store.Entries(c.prefix())
	keysSupported := err == nil
	return Stats{
		Name:            c.cacheName(),
		Driver:          c.store.Kind(),
		KeyCount:        len(entries),
		KeysSupported:   keysSupported,
		Hits:            atomic.LoadUint64(&c.hits),
		Misses:          atomic.LoadUint64(&c.misses),
		Sets:            atomic.LoadUint64(&c.sets),
		Deletes:         atomic.LoadUint64(&c.deletes),
		Expired:         atomic.LoadUint64(&c.expired),
		CleanupInterval: c.cleanupInterval,
		CreatedAt:       c.createdAt,
		Uptime:          time.Since(c.createdAt),
	}
}

// Close 停止清理协程（后端连接由 cache.Shutdown 统一释放）
func (c *Cache) Close() {
	c.once.Do(func() { close(c.stop) })
}

// cleanupLoop 定时清理过期缓存（redis / memcache 由服务端 TTL 负责）
func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sweeper, ok := c.store.(Sweeper)
			if !ok {
				continue
			}
			removed, err := sweeper.Sweep(c.prefix())
			if err != nil {
				logrus.Debugf("缓存 %s 过期清理失败（后端 %s）: %s", c.cacheName(), c.store.Kind(), err)
				continue
			}
			if removed > 0 {
				atomic.AddUint64(&c.expired, uint64(removed))
			}
		case <-c.stop:
			return
		}
	}
}

// trimPrefix 去掉缓存键的内部前缀
func trimPrefix(key, prefix string) string {
	if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
		return key[len(prefix):]
	}
	return key
}

// Default 包级默认缓存实例
var Default = New(time.Minute)

// Set 写入默认缓存
func Set(key string, value interface{}, ttl time.Duration) { Default.Set(key, value, ttl) }

// Get 读取默认缓存
func Get(key string) (interface{}, bool) { return Default.Get(key) }

// Delete 删除默认缓存
func Delete(key string) { Default.Delete(key) }

// 缓存注册表：缓存监控页面按名称展示各业务缓存
var (
	registryMu sync.RWMutex
	registry   = map[string]*Cache{}
)

// Register 注册被监控的缓存实例
// @param: name string 缓存名称（唯一）
// @param: c *Cache 缓存实例
func Register(name string, c *Cache) {
	if name == "" || c == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	c.name = name
	registry[name] = c
}

// Registered 已注册的缓存实例（按名称排序）
// @return: []*Cache
func Registered() []*Cache {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	caches := make([]*Cache, 0, len(names))
	for _, name := range names {
		caches = append(caches, registry[name])
	}
	return caches
}

// ClearAll 清空全部已注册缓存
func ClearAll() {
	for _, item := range Registered() {
		item.Clear()
	}
}

// Names 已注册的缓存名称（按名称排序）
// @return: []string
func Names() []string {
	caches := Registered()
	names := make([]string, 0, len(caches))
	for _, item := range caches {
		names = append(names, item.name)
	}
	return names
}

// Find 按名称查找已注册的缓存
// @param: name string 缓存名称
// @return: *Cache, bool
func Find(name string) (*Cache, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	c, ok := registry[name]
	return c, ok
}

// Driver 当前进程使用的缓存后端类型
// @return: string
func Driver() string { return sharedStore().Kind() }

func init() {
	// 默认缓存纳入监控
	Register("default", Default)
}
