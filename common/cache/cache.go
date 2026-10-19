package cache

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// item 缓存项
type item struct {
	value      interface{}
	expiration int64 // 过期时间（毫秒时间戳），0 表示永不过期
}

// expired 判断缓存项是否过期
func (i item) expired() bool {
	if i.expiration == 0 {
		return false
	}
	return time.Now().UnixMilli() > i.expiration
}

// Entry 缓存条目（缓存列表展示用）
type Entry struct {
	Key        string      // 缓存键
	Value      interface{} // 缓存值
	Expiration int64       // 过期时间（毫秒时间戳），0 表示永不过期
}

// Stats 缓存统计信息（缓存监控展示用）
type Stats struct {
	Name            string        // 缓存名称
	KeyCount        int           // 有效缓存键数量
	Hits            uint64        // 命中次数
	Misses          uint64        // 未命中次数
	Sets            uint64        // 写入次数
	Deletes         uint64        // 删除次数
	Expired         uint64        // 过期失效次数
	CleanupInterval time.Duration // 过期清理间隔
	CreatedAt       time.Time     // 创建时间
	Uptime          time.Duration // 运行时长
}

// Cache 带过期时间的内存缓存
// 用于验证码、登录失败次数等短生命周期数据；进程内有效，多实例部署时不共享
type Cache struct {
	name            string
	cleanupInterval time.Duration
	createdAt       time.Time
	items           sync.Map
	stop            chan struct{}
	once            sync.Once
	hits            uint64
	misses          uint64
	sets            uint64
	deletes         uint64
	expired         uint64
}

// New 创建内存缓存并启动过期清理协程
// @param: cleanupInterval time.Duration 清理间隔
// @return: *Cache
func New(cleanupInterval time.Duration) *Cache {
	c := &Cache{cleanupInterval: cleanupInterval, createdAt: time.Now(), stop: make(chan struct{})}
	if cleanupInterval > 0 {
		go c.cleanupLoop(cleanupInterval)
	}
	return c
}

// Set 写入缓存
// @param: key string 缓存键
// @param: value interface{} 缓存值
// @param: ttl time.Duration 有效期，<=0 表示永不过期
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixMilli()
	}
	c.items.Store(key, item{value: value, expiration: expiration})
	atomic.AddUint64(&c.sets, 1)
}

// Get 读取缓存（过期自动失效）
// @param: key string 缓存键
// @return: interface{} 缓存值, bool 是否存在
func (c *Cache) Get(key string) (interface{}, bool) {
	value, ok := c.items.Load(key)
	if !ok {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}
	it := value.(item)
	if it.expired() {
		c.items.Delete(key)
		atomic.AddUint64(&c.expired, 1)
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}
	atomic.AddUint64(&c.hits, 1)
	return it.value, true
}

// Delete 删除缓存
// @param: key string 缓存键
func (c *Cache) Delete(key string) {
	if _, exists := c.items.Load(key); exists {
		atomic.AddUint64(&c.deletes, 1)
	}
	c.items.Delete(key)
}

// Count 当前有效缓存项数量
// @return: int
func (c *Cache) Count() int {
	count := 0
	c.items.Range(func(_, value interface{}) bool {
		if !value.(item).expired() {
			count++
		}
		return true
	})
	return count
}

// Keys 当前有效缓存键（按字典序排序）
// @return: []string
func (c *Cache) Keys() []string {
	keys := make([]string, 0, 8)
	c.items.Range(func(key, value interface{}) bool {
		if !value.(item).expired() {
			keys = append(keys, key.(string))
		}
		return true
	})
	sort.Strings(keys)
	return keys
}

// Entries 当前有效缓存条目（按缓存键排序，缓存列表展示用）
// @return: []Entry
func (c *Cache) Entries() []Entry {
	entries := make([]Entry, 0, 8)
	c.items.Range(func(key, value interface{}) bool {
		it := value.(item)
		if it.expired() {
			return true
		}
		entries = append(entries, Entry{Key: key.(string), Value: it.value, Expiration: it.expiration})
		return true
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries
}

// Clear 清空缓存
func (c *Cache) Clear() {
	removed := uint64(0)
	c.items.Range(func(key, _ interface{}) bool {
		c.items.Delete(key)
		removed++
		return true
	})
	if removed > 0 {
		atomic.AddUint64(&c.deletes, removed)
	}
}

// Stats 缓存统计信息
// @return: Stats
func (c *Cache) Stats() Stats {
	return Stats{
		Name:            c.name,
		KeyCount:        c.Count(),
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

// Close 停止清理协程
func (c *Cache) Close() {
	c.once.Do(func() { close(c.stop) })
}

// cleanupLoop 定时清理过期缓存
func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.items.Range(func(key, value interface{}) bool {
				if value.(item).expired() {
					c.items.Delete(key)
					atomic.AddUint64(&c.expired, 1)
				}
				return true
			})
		case <-c.stop:
			return
		}
	}
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

func init() {
	// 默认缓存与验证码缓存纳入监控
	Register("default", Default)
}
