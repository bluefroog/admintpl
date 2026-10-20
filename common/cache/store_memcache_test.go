package cache

import (
	"os"
	"testing"
	"time"
)

// newTestMemcacheStore 连接真实 memcached（默认跳过）
//
// 启用方式（需要 docker 或本机 memcached）：
//
//	CACHE_TEST_MEMCACHE=127.0.0.1:11211 go test ./common/cache/ -run Memcache
func newTestMemcacheStore(t *testing.T) *memcacheStore {
	t.Helper()
	servers := os.Getenv("CACHE_TEST_MEMCACHE")
	if servers == "" {
		t.Skip("未设置 CACHE_TEST_MEMCACHE（如 127.0.0.1:11211），跳过 memcache 集成测试")
	}
	store, err := newMemcacheStore(MemcacheOptions{
		Servers: splitList(servers),
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("newMemcacheStore 失败: %s", err)
	}
	if err = store.Ping(); err != nil {
		t.Fatalf("memcache 连接失败（%s）: %s", servers, err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestMemcacheStoreSetGetDelete(t *testing.T) {
	store := newTestMemcacheStore(t)
	RegisterType(string(""))

	key := "unit:memcache:set-get"
	if err := store.Set(key, []byte("1"), time.Now().Add(time.Minute).UnixMilli()); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	payload, err := store.Get(key)
	if err != nil || string(payload) != "1" {
		t.Fatalf("Get = %q, %v; want 1, nil", payload, err)
	}
	if _, err = store.Get("unit:memcache:not-exist"); err != ErrNotFound {
		t.Fatalf("Get(not-exist) = %v; want ErrNotFound", err)
	}
	if err = store.Delete(key); err != nil {
		t.Fatalf("Delete 失败: %s", err)
	}
	if err = store.Delete(key); err != nil { // 重复删除不应报错
		t.Fatalf("重复 Delete 失败: %s", err)
	}
}

func TestMemcacheStoreTTL(t *testing.T) {
	store := newTestMemcacheStore(t)
	key := "unit:memcache:ttl"
	if err := store.Set(key, []byte("v"), time.Now().Add(time.Second).UnixMilli()); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	time.Sleep(1500 * time.Millisecond)
	if _, err := store.Get(key); err != ErrNotFound {
		t.Fatalf("过期后 Get = %v; want ErrNotFound", err)
	}
}

func TestMemcacheStoreUnsupported(t *testing.T) {
	store := newTestMemcacheStore(t)
	if _, err := store.Entries(""); err != ErrUnsupported {
		t.Errorf("memcache 不支持枚举键，Entries = %v; want ErrUnsupported", err)
	}
	if err := store.Clear("unit:"); err != ErrUnsupported {
		t.Errorf("memcache 不支持按前缀清空，Clear = %v; want ErrUnsupported", err)
	}
	if err := store.Clear(""); err != nil {
		t.Errorf("Clear(空前缀) 应执行 FlushAll: %s", err)
	}
}
