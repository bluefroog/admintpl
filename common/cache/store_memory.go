package cache

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// memoryStore 进程内内存后端：仅用于测试或临时排查（不持久化，多实例不共享）
type memoryStore struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

// memoryItem 内存缓存项
type memoryItem struct {
	payload    []byte
	expiration int64
}

// newMemoryStore 创建内存后端
// @return: *memoryStore
func newMemoryStore() *memoryStore {
	return &memoryStore{items: make(map[string]memoryItem, 16)}
}

// Kind 后端类型
func (s *memoryStore) Kind() string { return DriverMemory }

// Set 写入缓存
func (s *memoryStore) Set(key string, payload []byte, expiration int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = memoryItem{payload: payload, expiration: expiration}
	return nil
}

// Get 读取缓存
func (s *memoryStore) Get(key string) ([]byte, error) {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if expired(item.expiration) {
		s.mu.Lock()
		delete(s.items, key)
		s.mu.Unlock()
		return nil, ErrExpired
	}
	return item.payload, nil
}

// Delete 删除缓存
func (s *memoryStore) Delete(keys ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range keys {
		delete(s.items, key)
	}
	return nil
}

// Entries 枚举缓存条目
func (s *memoryStore) Entries(prefix string) ([]RawEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]RawEntry, 0, len(s.items))
	for key, item := range s.items {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		if expired(item.expiration) {
			continue
		}
		entries = append(entries, RawEntry{Key: key, Payload: item.payload, Expiration: item.expiration})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries, nil
}

// Clear 清空指定前缀下的缓存
func (s *memoryStore) Clear(prefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.items {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			delete(s.items, key)
		}
	}
	return nil
}

// Sweep 清理过期缓存
func (s *memoryStore) Sweep(prefix string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for key, item := range s.items {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		if expired(item.expiration) {
			delete(s.items, key)
			removed++
		}
	}
	return removed, nil
}

// Close 释放资源（内存后端无需处理）
func (s *memoryStore) Close() error { return nil }

// expired 判断毫秒时间戳是否已过期（0 表示永不过期）
func expired(expiration int64) bool {
	return expiration > 0 && time.Now().UnixMilli() > expiration
}
