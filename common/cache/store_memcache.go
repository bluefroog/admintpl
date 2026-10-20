package cache

import (
	"errors"
	"math"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// memcacheStore memcached 后端
//
// 说明：
//   - memcached 不支持枚举键，缓存监控页的「键列表」会提示不支持（读缓存本身不受影响）；
//   - 单条 value 上限 1MB（memcached 协议限制），超出会被服务端拒绝；
//   - 「清空缓存」只能 FlushAll（无法按前缀清空），因此多环境共用时请改清空方式或改用 redis。
type memcacheStore struct {
	client *memcache.Client
}

// newMemcacheStore 创建 memcache 后端
// @param: opts MemcacheOptions memcache 配置
// @return: *memcacheStore, error 错误信息
func newMemcacheStore(opts MemcacheOptions) (*memcacheStore, error) {
	if len(opts.Servers) == 0 {
		opts.Servers = []string{DefaultMemcacheAddr}
	}
	client := memcache.New(opts.Servers...)
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultMemcacheTimeout
	}
	client.Timeout = timeout
	client.MaxIdleConns = DefaultRedisPoolSize // 与 redis 保持一致的连接数上限
	return &memcacheStore{client: client}, nil
}

// Kind 后端类型
func (s *memcacheStore) Kind() string { return DriverMemcache }

// Set 写入缓存（memcached 的过期时间以秒为单位）
func (s *memcacheStore) Set(key string, payload []byte, expiration int64) error {
	item := &memcache.Item{Key: key, Value: payload}
	if expiration > 0 {
		seconds := int32(math.Ceil(time.Until(time.UnixMilli(expiration)).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		item.Expiration = seconds
	}
	return s.client.Set(item)
}

// Get 读取缓存
func (s *memcacheStore) Get(key string) ([]byte, error) {
	item, err := s.client.Get(key)
	if err != nil {
		if errors.Is(err, memcache.ErrCacheMiss) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item.Value, nil
}

// Delete 删除缓存
func (s *memcacheStore) Delete(keys ...string) error {
	for _, key := range keys {
		if err := s.client.Delete(key); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
			return err
		}
	}
	return nil
}

// Entries memcached 不支持枚举键
func (s *memcacheStore) Entries(string) ([]RawEntry, error) {
	return nil, ErrUnsupported
}

// Clear 清空缓存（memcached 只能整体清空，前缀无法隔离）
func (s *memcacheStore) Clear(prefix string) error {
	if prefix != "" {
		return ErrUnsupported
	}
	return s.client.FlushAll()
}

// Ping 探活
func (s *memcacheStore) Ping() error {
	return s.client.Ping()
}

// Close 释放连接
func (s *memcacheStore) Close() error { return nil }
