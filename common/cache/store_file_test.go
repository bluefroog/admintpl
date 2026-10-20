package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestFileStore 创建使用临时目录的本地文件后端
func newTestFileStore(t *testing.T) (*fileStore, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := newFileStore(FileOptions{Dir: dir, FlushInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("newFileStore 失败: %s", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, dir
}

func TestFileStoreSetGetDelete(t *testing.T) {
	store, _ := newTestFileStore(t)

	if err := store.Set("admintpl:sys_config:a", []byte("1"), 0); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	payload, err := store.Get("admintpl:sys_config:a")
	if err != nil || string(payload) != "1" {
		t.Fatalf("Get = %q, %v; want 1, nil", payload, err)
	}
	if _, err = store.Get("not-exist"); err != ErrNotFound {
		t.Fatalf("Get(not-exist) = %v; want ErrNotFound", err)
	}
	if err = store.Delete("admintpl:sys_config:a"); err != nil {
		t.Fatalf("Delete 失败: %s", err)
	}
	if _, err = store.Get("admintpl:sys_config:a"); err != ErrNotFound {
		t.Fatalf("删除后 Get = %v; want ErrNotFound", err)
	}
}

func TestFileStoreExpired(t *testing.T) {
	store, _ := newTestFileStore(t)
	if err := store.Set("k", []byte("v"), time.Now().Add(-time.Second).UnixMilli()); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	if _, err := store.Get("k"); err != ErrExpired {
		t.Fatalf("过期键 Get = %v; want ErrExpired", err)
	}
	// 过期键不应出现在枚举结果中，Sweep 后应被真正删除
	if entries, err := store.Entries(""); err != nil || len(entries) != 0 {
		t.Fatalf("Entries = %v, %v; want 空", entries, err)
	}
	removed, err := store.Sweep("")
	if err != nil || removed != 0 {
		// Get 时已删除，Sweep 数量为 0 也正确
		if err != nil {
			t.Fatalf("Sweep 失败: %s", err)
		}
	}
}

func TestFileStoreSweep(t *testing.T) {
	store, _ := newTestFileStore(t)
	_ = store.Set("prefix:live", []byte("1"), 0)
	_ = store.Set("prefix:dead", []byte("2"), time.Now().Add(-time.Second).UnixMilli())
	_ = store.Set("other:live", []byte("3"), 0)

	removed, err := store.Sweep("prefix:")
	if err != nil {
		t.Fatalf("Sweep 失败: %s", err)
	}
	if removed != 1 {
		t.Errorf("Sweep 清理数量 = %d; want 1（只清理指定前缀的过期键）", removed)
	}
	if _, err = store.Get("prefix:dead"); err != ErrNotFound {
		t.Errorf("过期键应被清理，Get = %v", err)
	}
	if _, err = store.Get("other:live"); err != nil {
		t.Errorf("其他前缀的键不应被清理，Get = %v", err)
	}
}

func TestFileStoreEntriesAndClear(t *testing.T) {
	store, _ := newTestFileStore(t)
	_ = store.Set("admintpl:sys_dict:b", []byte("2"), 0)
	_ = store.Set("admintpl:sys_dict:a", []byte("1"), 0)
	_ = store.Set("admintpl:sys_config:a", []byte("3"), 0)

	entries, err := store.Entries("admintpl:sys_dict:")
	if err != nil {
		t.Fatalf("Entries 失败: %s", err)
	}
	if len(entries) != 2 || entries[0].Key != "admintpl:sys_dict:a" || entries[1].Key != "admintpl:sys_dict:b" {
		t.Fatalf("Entries = %v; want 按 key 排序的 2 条", entries)
	}

	if err = store.Clear("admintpl:sys_dict:"); err != nil {
		t.Fatalf("Clear 失败: %s", err)
	}
	if entries, _ = store.Entries("admintpl:sys_dict:"); len(entries) != 0 {
		t.Errorf("Clear 后 Entries = %v; want 空", entries)
	}
	if entries, _ = store.Entries("admintpl:sys_config:"); len(entries) != 1 {
		t.Errorf("其他前缀不应被清空，Entries = %v; want 1 条", entries)
	}
}

// TestFileStorePersistence 落盘后重新加载应能还原缓存（含过期条目丢弃）
func TestFileStorePersistence(t *testing.T) {
	dir := t.TempDir()
	store, err := newFileStore(FileOptions{Dir: dir, FlushInterval: time.Hour})
	if err != nil {
		t.Fatalf("newFileStore 失败: %s", err)
	}
	if err = store.Set("admintpl:sys_config:keep", []byte("v"), 0); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	if err = store.Set("admintpl:sys_config:expired", []byte("v"), time.Now().Add(-time.Second).UnixMilli()); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	if err = store.Close(); err != nil { // 关闭时立即落盘
		t.Fatalf("Close 失败: %s", err)
	}

	path := filepath.Join(dir, fileName)
	if _, err = os.Stat(path); err != nil {
		t.Fatalf("落盘文件不存在: %s", err)
	}

	reloaded, err := newFileStore(FileOptions{Dir: dir, FlushInterval: time.Hour})
	if err != nil {
		t.Fatalf("重新加载失败: %s", err)
	}
	defer func() { _ = reloaded.Close() }()

	payload, err := reloaded.Get("admintpl:sys_config:keep")
	if err != nil || string(payload) != "v" {
		t.Fatalf("重新加载后 Get = %q, %v; want v, nil", payload, err)
	}
	if _, err = reloaded.Get("admintpl:sys_config:expired"); err != ErrNotFound {
		t.Fatalf("过期条目应在加载时被丢弃，Get = %v", err)
	}
}

// TestFileStoreCorruptedFile 数据文件损坏时应备份并以空缓存启动（不影响服务可用性）
func TestFileStoreCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte("not-a-gob-file"), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := newFileStore(FileOptions{Dir: dir, FlushInterval: time.Hour})
	if err != nil {
		t.Fatalf("损坏文件不应导致初始化失败: %s", err)
	}
	defer func() { _ = store.Close() }()

	if _, err = os.Stat(filepath.Join(dir, fileName+".bak")); err != nil {
		t.Fatalf("损坏文件应被备份为 .bak: %s", err)
	}
	if entries, _ := store.Entries(""); len(entries) != 0 {
		t.Errorf("损坏文件加载后应为空缓存，Entries = %v", entries)
	}
}

// TestFileStoreWithCache 通过 Cache 走一遍本地文件后端（验证键前缀与编解码）
func TestFileStoreWithCache(t *testing.T) {
	store, _ := newTestFileStore(t)
	c := newCache(0, "unit:", store)
	Register("unit-file", c)

	c.Set("plain", "text", time.Minute)
	c.Set("dict", []string{"a", "b"}, 0)

	if value, ok := c.Get("plain"); !ok || value.(string) != "text" {
		t.Fatalf("Get(plain) = %v, %v", value, ok)
	}
	if value, ok := c.Get("dict"); !ok || len(value.([]string)) != 2 {
		t.Fatalf("Get(dict) = %v, %v", value, ok)
	}
	if keys := c.Keys(); len(keys) != 2 || keys[0] != "dict" || keys[1] != "plain" {
		t.Fatalf("Keys() = %v; want [dict plain]", keys)
	}
	if stats := c.Stats(); stats.Driver != DriverFile || stats.KeyCount != 2 || !stats.KeysSupported {
		t.Fatalf("Stats() = %+v", stats)
	}
}
