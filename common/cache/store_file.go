package cache

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// fileName 本地缓存数据文件名（目录由 system.cache.file.dir 配置）
const fileName = "cache.gob"

// fileItem 落盘的缓存项（payload 已是序列化后的字节，不参与 gob 的接口类型推断）
type fileItem struct {
	Payload    []byte // 序列化后的缓存值
	Expiration int64  // 过期时间（毫秒时间戳），0 表示永不过期
}

// fileBackend 本地文件后端：进程内 map 读写 + 定时 gob 落盘
//
// 说明：
//   - 所有缓存实例共用一个 backend（一个文件、一个落盘协程），键通过前缀区分；
//   - 启动时从磁盘加载，加载时丢弃已过期条目；
//   - 写入只改内存并标记 dirty，按 flush-interval 落盘（原子替换：先写 .tmp 再 rename）；
//   - 关闭时（cache.Shutdown）立即落盘，避免丢失最后一个间隔内的写入。
type fileBackend struct {
	dir           string
	path          string
	flushInterval time.Duration

	mu    sync.RWMutex
	items map[string]fileItem
	dirty bool

	stop chan struct{}
	once sync.Once
}

// newFileBackend 创建（并按需加载）本地文件后端
// @param: opts FileOptions 本地文件配置
// @return: *fileBackend, error 错误信息
func newFileBackend(opts FileOptions) (*fileBackend, error) {
	dir := opts.Dir
	if dir == "" {
		dir = DefaultFileDir
	}
	interval := opts.FlushInterval
	if interval <= 0 {
		interval = DefaultFileFlushSeconds * time.Second
	}
	backend := &fileBackend{
		dir:           dir,
		path:          filepath.Join(dir, fileName),
		flushInterval: interval,
		items:         make(map[string]fileItem, 64),
		stop:          make(chan struct{}),
	}
	if err := backend.load(); err != nil {
		return nil, err
	}
	go backend.flushLoop()
	return backend, nil
}

// load 从磁盘加载缓存数据（文件不存在视为空缓存）
// @return: error 错误信息
func (b *fileBackend) load() error {
	data, err := os.ReadFile(b.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // 首次启动没有历史文件
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	items := make(map[string]fileItem, 64)
	if err = gob.NewDecoder(bytes.NewReader(data)).Decode(&items); err != nil {
		// 文件损坏时不让服务起不来：备份后以空缓存启动（缓存可回源数据库）
		logrus.Warnf("本地缓存文件解析失败，已备份为 %s.bak 并重新开始: %s", b.path, err)
		_ = os.Rename(b.path, b.path+".bak")
		return nil
	}
	now := time.Now().UnixMilli()
	for key, item := range items {
		if item.Expiration > 0 && now > item.Expiration {
			continue // 丢弃已过期条目
		}
		b.items[key] = item
	}
	logrus.Infof("本地缓存已加载: %s（%d 条）", b.path, len(b.items))
	return nil
}

// flushLoop 定时落盘
func (b *fileBackend) flushLoop() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := b.flush(); err != nil {
				logrus.Warnf("本地缓存落盘失败: %s", err)
			}
		case <-b.stop:
			return
		}
	}
}

// flush 将内存数据原子写入磁盘（无变更时跳过）
// @return: error 错误信息
func (b *fileBackend) flush() error {
	b.mu.Lock()
	if !b.dirty {
		b.mu.Unlock()
		return nil
	}
	snapshot := make(map[string]fileItem, len(b.items))
	for key, item := range b.items {
		snapshot[key] = item
	}
	b.dirty = false
	b.mu.Unlock()

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snapshot); err != nil {
		b.markDirty()
		return err
	}
	if err := os.MkdirAll(b.dir, 0o750); err != nil {
		b.markDirty()
		return err
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		b.markDirty()
		return err
	}
	if err := os.Rename(tmp, b.path); err != nil {
		b.markDirty()
		return err
	}
	return nil
}

// markDirty 标记需要落盘
func (b *fileBackend) markDirty() {
	b.mu.Lock()
	b.dirty = true
	b.mu.Unlock()
}

// Close 停止落盘协程并立即落盘
func (b *fileBackend) Close() error {
	var err error
	b.once.Do(func() {
		close(b.stop)
		err = b.flush()
	})
	return err
}

// fileStore 本地文件后端的 Store 实现（按前缀区分不同缓存实例）
type fileStore struct {
	backend *fileBackend
}

// newFileStore 创建本地文件缓存后端
// @param: opts FileOptions 本地文件配置
// @return: *fileStore, error 错误信息
func newFileStore(opts FileOptions) (*fileStore, error) {
	backend, err := newFileBackend(opts)
	if err != nil {
		return nil, err
	}
	return &fileStore{backend: backend}, nil
}

// Kind 后端类型
func (s *fileStore) Kind() string { return DriverFile }

// Set 写入缓存
func (s *fileStore) Set(key string, payload []byte, expiration int64) error {
	b := s.backend
	b.mu.Lock()
	b.items[key] = fileItem{Payload: payload, Expiration: expiration}
	b.dirty = true
	b.mu.Unlock()
	return nil
}

// Get 读取缓存
func (s *fileStore) Get(key string) ([]byte, error) {
	b := s.backend
	b.mu.RLock()
	item, ok := b.items[key]
	b.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	if expired(item.Expiration) {
		_ = s.Delete(key)
		return nil, ErrExpired
	}
	return item.Payload, nil
}

// Delete 删除缓存
func (s *fileStore) Delete(keys ...string) error {
	b := s.backend
	b.mu.Lock()
	for _, key := range keys {
		delete(b.items, key)
	}
	b.dirty = true
	b.mu.Unlock()
	return nil
}

// Entries 枚举缓存条目
func (s *fileStore) Entries(prefix string) ([]RawEntry, error) {
	b := s.backend
	b.mu.RLock()
	defer b.mu.RUnlock()
	now := time.Now().UnixMilli()
	entries := make([]RawEntry, 0, len(b.items))
	for key, item := range b.items {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		if item.Expiration > 0 && now > item.Expiration {
			continue
		}
		entries = append(entries, RawEntry{Key: key, Payload: item.Payload, Expiration: item.Expiration})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries, nil
}

// Clear 删除指定前缀下的缓存
func (s *fileStore) Clear(prefix string) error {
	b := s.backend
	b.mu.Lock()
	for key := range b.items {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			delete(b.items, key)
		}
	}
	b.dirty = true
	b.mu.Unlock()
	return nil
}

// Sweep 清理过期缓存
func (s *fileStore) Sweep(prefix string) (int, error) {
	b := s.backend
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now().UnixMilli()
	removed := 0
	for key, item := range b.items {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		if item.Expiration > 0 && now > item.Expiration {
			delete(b.items, key)
			removed++
		}
	}
	if removed > 0 {
		b.dirty = true
	}
	return removed, nil
}

// Close 落盘并释放后端
func (s *fileStore) Close() error { return s.backend.Close() }
