package cache

import (
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// newTestRedisStore 启动内嵌 redis（miniredis）并创建 redis 后端
func newTestRedisStore(t *testing.T) (*redisStore, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	store, err := newRedisStore(RedisOptions{Mode: RedisModeStandalone, Addr: server.Addr(), Timeout: time.Second})
	if err != nil {
		t.Fatalf("newRedisStore 失败: %s", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, server
}

func TestRedisStoreSetGetDelete(t *testing.T) {
	store, _ := newTestRedisStore(t)
	if err := store.Ping(); err != nil {
		t.Fatalf("Ping 失败: %s", err)
	}
	if err := store.Set("admintpl:sys_config:a", []byte("1"), time.Now().Add(time.Minute).UnixMilli()); err != nil {
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
	if store.Kind() != DriverRedis || store.Mode() != RedisModeStandalone {
		t.Errorf("Kind/Mode = %s/%s", store.Kind(), store.Mode())
	}
}

func TestRedisStoreTTL(t *testing.T) {
	store, server := newTestRedisStore(t)
	expiration := time.Now().Add(time.Minute).UnixMilli()
	if err := store.Set("k", []byte("v"), expiration); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	ttl := server.TTL("k")
	if ttl <= 0 || ttl > time.Minute {
		t.Fatalf("redis TTL = %s; want (0, 1m]", ttl)
	}

	// 永久缓存不设 TTL
	if err := store.Set("keep", []byte("v"), 0); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	if ttl = server.TTL("keep"); ttl != 0 {
		t.Errorf("永不过期的键 TTL = %s; want 0", ttl)
	}
}

func TestRedisStoreEntriesAndClear(t *testing.T) {
	store, server := newTestRedisStore(t)
	_ = store.Set("admintpl:sys_dict:b", []byte("2"), 0)
	_ = store.Set("admintpl:sys_dict:a", []byte("1"), 0)
	_ = store.Set("admintpl:sys_config:a", []byte("3"), 0)
	_ = server.Set("other:key", "x") // 无关键不应被清理

	entries, err := store.Entries("admintpl:sys_dict:")
	if err != nil {
		t.Fatalf("Entries 失败: %s", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Entries = %v; want 2 条", entries)
	}
	for _, entry := range entries {
		if entry.Key != "admintpl:sys_dict:a" && entry.Key != "admintpl:sys_dict:b" {
			t.Errorf("Entries 包含意外键: %s", entry.Key)
		}
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
	if !server.Exists("other:key") {
		t.Errorf("无关键不应被清空")
	}
}

// TestRedisStoreMasterSlaveFailover 主从模式：主节点不可用时自动切到从节点
func TestRedisStoreMasterSlaveFailover(t *testing.T) {
	master := miniredis.RunT(t)
	replica := miniredis.RunT(t)

	store, err := newRedisStore(RedisOptions{
		Mode:    RedisModeMasterSlave,
		Addrs:   []string{master.Addr(), replica.Addr()},
		Timeout: 200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("newRedisStore 失败: %s", err)
	}
	defer func() { _ = store.Close() }()

	// 正常读写走主节点
	if err = store.Set("k", []byte("v"), 0); err != nil {
		t.Fatalf("Set 失败: %s", err)
	}
	if !master.Exists("k") {
		t.Fatalf("写入应落在主节点")
	}

	// 主节点下线后应自动切到从节点（模拟主从切换完成前/从节点被提升）
	master.Close()
	if err = store.Set("k2", []byte("v2"), 0); err != nil {
		t.Fatalf("主节点不可用时应切到从节点，Set 失败: %s", err)
	}
	if !replica.Exists("k2") {
		t.Fatalf("主节点不可用后写入应落在从节点")
	}
	if payload, err := store.Get("k2"); err != nil || string(payload) != "v2" {
		t.Fatalf("Get(k2) = %q, %v; want v2, nil", payload, err)
	}
	if keys, err := store.Entries(""); err != nil || len(keys) != 1 {
		t.Fatalf("主节点不可用时枚举应回退从节点，Entries = %v, %v", keys, err)
	}
}

// TestRedisStoreSentinelRequiresAddrs 哨兵/集群模式缺少节点配置时应给出明确错误
func TestRedisStoreRequiresAddrs(t *testing.T) {
	if _, err := newRedisStore(RedisOptions{Mode: RedisModeSentinel}); err == nil {
		t.Fatalf("哨兵模式缺少节点应返回错误")
	}
	if _, err := newRedisStore(RedisOptions{Mode: RedisModeCluster}); err == nil {
		t.Fatalf("集群模式缺少节点应返回错误")
	}
}

func TestRedisStoreWithCache(t *testing.T) {
	store, _ := newTestRedisStore(t)
	c := newCache(0, "unit:", store)
	Register("unit-redis", c)

	c.Set("plain", "text", time.Minute)
	c.Set("list", []string{"a", "b"}, 0)
	if value, ok := c.Get("plain"); !ok || value.(string) != "text" {
		t.Fatalf("Get(plain) = %v, %v", value, ok)
	}
	if value, ok := c.Get("list"); !ok || len(value.([]string)) != 2 {
		t.Fatalf("Get(list) = %v, %v", value, ok)
	}
	if keys := c.Keys(); len(keys) != 2 {
		t.Fatalf("Keys() = %v; want 2 条（去掉内部前缀）", keys)
	}
	if stats := c.Stats(); stats.Driver != DriverRedis || !stats.KeysSupported {
		t.Fatalf("Stats() = %+v", stats)
	}
	c.Clear()
	if count := c.Count(); count != 0 {
		t.Errorf("Clear 后 Count = %d; want 0", count)
	}
}

// TestNewStoreByDriver 驱动选择：未知驱动回退本地文件，memory 驱动可用
func TestNewStoreByDriver(t *testing.T) {
	store, err := newStore(Options{Driver: DriverMemory})
	if err != nil || store.Kind() != DriverMemory {
		t.Fatalf("newStore(memory) = %v, %v", store, err)
	}
	fileStore, err := newStore(Options{Driver: DriverFile, File: FileOptions{Dir: t.TempDir(), FlushInterval: time.Hour}})
	if err != nil || fileStore.Kind() != DriverFile {
		t.Fatalf("newStore(file) = %v, %v", fileStore, err)
	}
	_ = fileStore.Close()

	if _, err = newStore(Options{Driver: "unknown"}); err == nil {
		t.Fatalf("未知驱动应返回错误")
	}
}

func TestFileStoreSweepErrors(t *testing.T) {
	// Sweep 在无过期键时返回 0，且不修改数据
	store, _ := newTestFileStore(t)
	_ = store.Set("k", []byte("v"), 0)
	removed, err := store.Sweep("")
	if err != nil || removed != 0 {
		t.Fatalf("Sweep = %d, %v; want 0, nil", removed, err)
	}
	if _, err = store.Get("k"); err != nil {
		t.Fatalf("Sweep 不应删除有效键: %s", err)
	}
}

func TestMemoryStorePingNotSupported(t *testing.T) {
	store := newMemoryStore()
	if _, ok := any(store).(Pinger); ok {
		t.Errorf("内存后端不应实现 Pinger")
	}
	if _, ok := any(store).(Sweeper); !ok {
		t.Errorf("内存后端应实现 Sweeper")
	}
	if err := store.Close(); err != nil {
		t.Errorf("内存后端 Close 应返回 nil: %s", err)
	}
}

func TestRedisNilIsNotConnectError(t *testing.T) {
	// redis.Nil 表示键不存在，不能当作连接错误触发主从切换
	if isConnectError(errors.New("redis: nil")) {
		t.Errorf("redis.Nil 不应判定为连接错误")
	}
}
