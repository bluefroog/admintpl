package cache

import (
	"sync"
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

// Cache 带过期时间的内存缓存
// 用于验证码、登录失败次数等短生命周期数据；进程内有效，多实例部署时不共享
type Cache struct {
	items sync.Map
	stop  chan struct{}
	once  sync.Once
}

// New 创建内存缓存并启动过期清理协程
// @param: cleanupInterval time.Duration 清理间隔
// @return: *Cache
func New(cleanupInterval time.Duration) *Cache {
	c := &Cache{stop: make(chan struct{})}
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
}

// Get 读取缓存（过期自动失效）
// @param: key string 缓存键
// @return: interface{} 缓存值, bool 是否存在
func (c *Cache) Get(key string) (interface{}, bool) {
	value, ok := c.items.Load(key)
	if !ok {
		return nil, false
	}
	it := value.(item)
	if it.expired() {
		c.items.Delete(key)
		return nil, false
	}
	return it.value, true
}

// Delete 删除缓存
// @param: key string 缓存键
func (c *Cache) Delete(key string) {
	c.items.Delete(key)
}

// Count 当前缓存项数量
// @return: int
func (c *Cache) Count() int {
	count := 0
	c.items.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// Clear 清空缓存
func (c *Cache) Clear() {
	c.items.Range(func(key, _ interface{}) bool {
		c.items.Delete(key)
		return true
	})
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
