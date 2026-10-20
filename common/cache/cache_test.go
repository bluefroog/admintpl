package cache

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// newTestCache 使用内存后端创建缓存实例（避免测试写本地文件）
func newTestCache(t *testing.T, cleanupInterval time.Duration) *Cache {
	t.Helper()
	return newCache(cleanupInterval, "", newMemoryStore())
}

func TestCacheSetGetStats(t *testing.T) {
	c := newTestCache(t, 0)
	c.Set("a", "1", time.Minute)
	if value, ok := c.Get("a"); !ok || value.(string) != "1" {
		t.Fatalf("Get(a) = %v, %v; want 1, true", value, ok)
	}
	if _, ok := c.Get("not-exist"); ok {
		t.Fatalf("Get(not-exist) should miss")
	}
	stats := c.Stats()
	if stats.KeyCount != 1 {
		t.Errorf("KeyCount = %d; want 1", stats.KeyCount)
	}
	if !stats.KeysSupported {
		t.Errorf("内存后端应支持枚举键")
	}
	if stats.Driver != DriverMemory {
		t.Errorf("Driver = %s; want %s", stats.Driver, DriverMemory)
	}
	if stats.Sets != 1 {
		t.Errorf("Sets = %d; want 1", stats.Sets)
	}
	if stats.Hits != 1 {
		t.Errorf("Hits = %d; want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d; want 1", stats.Misses)
	}
	c.Delete("a")
	c.Delete("not-exist") // 不存在的键不计数
	if count := c.Count(); count != 0 {
		t.Errorf("Count = %d; want 0", count)
	}
	if stats = c.Stats(); stats.Deletes != 1 {
		t.Errorf("Deletes = %d; want 1", stats.Deletes)
	}
}

func TestCacheExpiration(t *testing.T) {
	c := newTestCache(t, 0)
	c.Set("expired", "1", time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	value, ok := c.Get("expired")
	if ok || value != nil {
		t.Fatalf("过期的缓存项应不可读取")
	}
	if count := c.Count(); count != 0 {
		t.Errorf("Count = %d; want 0（过期项不计入）", count)
	}
	if stats := c.Stats(); stats.Expired != 1 {
		t.Errorf("Expired = %d; want 1", stats.Expired)
	}
}

func TestCacheKeysEntriesAndClear(t *testing.T) {
	c := newTestCache(t, 0)
	c.Set("b", 2, 0) // 永不过期
	c.Set("a", 1, time.Minute)
	keys := c.Keys()
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("Keys() = %v; want [a b]", keys)
	}
	entries := c.Entries()
	if len(entries) != 2 || entries[0].Key != "a" || entries[0].Value.(int) != 1 {
		t.Fatalf("Entries() = %v", entries)
	}
	if entries[0].Expiration == 0 || entries[1].Expiration != 0 {
		t.Errorf("过期时间不符合预期：%v", entries)
	}
	c.Clear()
	if count := c.Count(); count != 0 {
		t.Errorf("Clear 后 Count = %d; want 0", count)
	}
}

// TestCacheIsolationBetweenCaches 不同缓存实例共用同一后端时键不会互相影响
// TestCacheKeyFormat 实际存储键必须是「前缀:缓存名:业务键」（分隔符不能丢）
func TestCacheKeyFormat(t *testing.T) {
	cases := map[string]string{
		"admintpl":  "admintpl:unit-key-format:business",
		"admintpl:": "admintpl:unit-key-format:business",
		"":          "unit-key-format:business",
		"  a-b  ":   "a-b:unit-key-format:business",
	}
	for prefix, want := range cases {
		store := newMemoryStore()
		c := newCache(0, prefix, store)
		Register("unit-key-format", c)
		c.Set("business", "v", 0)

		entries, err := store.Entries("")
		if err != nil || len(entries) != 1 {
			t.Fatalf("前缀 %q: Entries = %v, %v", prefix, entries, err)
		}
		if entries[0].Key != want {
			t.Errorf("前缀 %q: 实际键 = %q; want %q", prefix, entries[0].Key, want)
		}
		if keys := c.Keys(); len(keys) != 1 || keys[0] != "business" {
			t.Errorf("前缀 %q: Keys() = %v; want [business]", prefix, keys)
		}
		if count := c.Count(); count != 1 {
			t.Errorf("前缀 %q: Count() = %d; want 1", prefix, count)
		}
	}
}

func TestCacheIsolationBetweenCaches(t *testing.T) {
	store := newMemoryStore()
	first := newCache(0, "unit:", store)
	Register("unit-first", first)
	second := newCache(0, "unit:", store)
	Register("unit-second", second)

	first.Set("key", "first", 0)
	second.Set("key", "second", 0)

	if value, ok := first.Get("key"); !ok || value.(string) != "first" {
		t.Fatalf("first.Get(key) = %v, %v; want first", value, ok)
	}
	if value, ok := second.Get("key"); !ok || value.(string) != "second" {
		t.Fatalf("second.Get(key) = %v, %v; want second", value, ok)
	}
	if keys := first.Keys(); len(keys) != 1 || keys[0] != "key" {
		t.Fatalf("first.Keys() = %v; want [key]（应去掉内部前缀）", keys)
	}

	second.Clear()
	if count := second.Count(); count != 0 {
		t.Errorf("second.Clear 后 Count = %d; want 0", count)
	}
	if count := first.Count(); count != 1 {
		t.Errorf("first 的缓存不应被清空，Count = %d; want 1", count)
	}
}

func TestCacheRegistry(t *testing.T) {
	c := newTestCache(t, 0)
	Register("test-registry", c)
	if found, ok := Find("test-registry"); !ok || found != c {
		t.Fatalf("Find(test-registry) = %v, %v", found, ok)
	}
	if _, ok := Find("not-exist"); ok {
		t.Errorf("Find(not-exist) 应返回 false")
	}
	names := Names()
	found := false
	for _, name := range names {
		if name == "test-registry" {
			found = true
		}
	}
	if !found {
		t.Errorf("Names() = %v; want 包含 test-registry", names)
	}
	if len(Registered()) != len(names) {
		t.Errorf("Registered() 与 Names() 数量不一致")
	}

	c.Set("key", "value", 0)
	ClearAll()
	if count := c.Count(); count != 0 {
		t.Errorf("ClearAll 后 Count = %d; want 0", count)
	}
}

func TestCacheCloseIdempotent(t *testing.T) {
	c := newTestCache(t, time.Millisecond)
	c.Set("a", 1, 0)
	c.Close()
	c.Close() // 重复关闭不应 panic
}

// TestCacheUnsupportedEnumeration 后端不支持枚举时（memcache）不应报错，只是列表为空
func TestCacheUnsupportedEnumeration(t *testing.T) {
	c := newCache(0, "", &unsupportedStore{newMemoryStore()})
	Register("unit-unsupported", c)
	c.Set("a", "1", 0)

	if _, ok := c.Get("a"); !ok {
		t.Fatalf("不支持枚举的后端读写应正常")
	}
	if count := c.Count(); count != 0 {
		t.Errorf("Count = %d; want 0（后端不支持枚举）", count)
	}
	if keys := c.Keys(); len(keys) != 0 {
		t.Errorf("Keys() = %v; want 空", keys)
	}
	if entries := c.Entries(); len(entries) != 0 {
		t.Errorf("Entries() = %v; want 空", entries)
	}
	if stats := c.Stats(); stats.KeysSupported {
		t.Errorf("KeysSupported 应为 false")
	}
	c.Clear() // 后端不支持清空时只记录日志，不 panic
}

// unsupportedStore 模拟 memcache：不支持枚举键/前缀清空
type unsupportedStore struct {
	*memoryStore
}

func (s *unsupportedStore) Entries(string) ([]RawEntry, error) { return nil, ErrUnsupported }
func (s *unsupportedStore) Clear(string) error                 { return ErrUnsupported }

func TestCodecRoundTrip(t *testing.T) {
	type payload struct {
		Name  string
		Count int
		Tags  []string
	}
	RegisterType(payload{})

	cases := []interface{}{
		"text",
		[]byte("bytes"),
		42,
		int64(7),
		true,
		3.14,
		[]string{"a", "b"},
		map[string]string{"k": "v"},
		payload{Name: "demo", Count: 2, Tags: []string{"x"}},
		[]payload{{Name: "a"}, {Name: "b"}},
	}
	for _, value := range cases {
		encoded, err := encodeValue(value)
		if err != nil {
			t.Fatalf("encodeValue(%v) 失败: %s", value, err)
		}
		decoded, err := decodeValue(encoded)
		if err != nil {
			t.Fatalf("decodeValue(%v) 失败: %s", value, err)
		}
		if got, want := toText(decoded), toText(value); got != want {
			t.Errorf("编解码不一致: got %q, want %q", got, want)
		}
	}
}

func TestDecodeValueUnknownType(t *testing.T) {
	type unregistered struct{ Value string } // 未调用 RegisterType

	encoded, err := encodeValue(unregistered{Value: "x"})
	if err != nil {
		t.Fatalf("encodeValue 失败: %s", err)
	}
	// 模拟「其他进程写入、本进程未注册该类型」：清掉注册表
	typeMu.Lock()
	delete(types, "cache.unregistered")
	typeMu.Unlock()

	if _, err = decodeValue(encoded); err == nil {
		t.Fatalf("未注册类型应解码失败")
	}
}

func TestEncodeValueNil(t *testing.T) {
	if _, err := encodeValue(nil); err == nil {
		t.Fatalf("nil 值应返回错误")
	}
}

func TestDecodeValueCorrupted(t *testing.T) {
	if _, err := decodeValue([]byte("not-a-gob-payload")); err == nil {
		t.Fatalf("损坏的载荷应解码失败")
	}
}

func TestNormalizeDriverAndMode(t *testing.T) {
	drivers := map[string]string{
		"":            DriverFile,
		"file":        DriverFile,
		"FILE":        DriverFile,
		"local-file":  DriverFile,
		"memory":      DriverMemory,
		"in-memory":   DriverMemory,
		"memcached":   DriverMemcache,
		"memcache":    DriverMemcache,
		"redis":       DriverRedis,
		"unknown-xxx": DriverFile, // 未知驱动回退默认
	}
	for input, want := range drivers {
		if got := normalizeDriver(input); got != want {
			t.Errorf("normalizeDriver(%q) = %q; want %q", input, got, want)
		}
	}

	modes := map[string]string{
		"":             RedisModeStandalone,
		"standalone":   RedisModeStandalone,
		"single":       RedisModeStandalone,
		"master-slave": RedisModeMasterSlave,
		"replication":  RedisModeMasterSlave,
		"sentinel":     RedisModeSentinel,
		"failover":     RedisModeSentinel,
		"cluster":      RedisModeCluster,
		"unknown":      RedisModeStandalone,
	}
	for input, want := range modes {
		if got := normalizeRedisMode(input); got != want {
			t.Errorf("normalizeRedisMode(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestEscapeGlob(t *testing.T) {
	cases := map[string]string{
		"admintpl:":   "admintpl:",
		"a*b":         `a\*b`,
		"a?b":         `a\?b`,
		`a[b]`:        `a\[b\]`,
		`a\b`:         `a\\b`,
		"":            "",
		"普通前缀:":       "普通前缀:",
		"prefix:名称:x": "prefix:名称:x",
	}
	for input, want := range cases {
		if got := escapeGlob(input); got != want {
			t.Errorf("escapeGlob(%q) = %q; want %q", input, got, want)
		}
	}
}

func TestIsConnectError(t *testing.T) {
	if isConnectError(nil) {
		t.Error("nil 不应判定为连接错误")
	}
	if isConnectError(errors.New("WRONGPASS invalid username-password pair")) {
		t.Error("认证错误不应判定为连接错误")
	}
	if !isConnectError(errors.New("dial tcp 127.0.0.1:6379: connect: connection refused")) {
		t.Error("connection refused 应判定为连接错误")
	}
	if !isConnectError(errors.New("read tcp: i/o timeout")) {
		t.Error("i/o timeout 应判定为连接错误")
	}
}

// toText 把任意值转成可比较文本（同时比较类型与内容）
func toText(value interface{}) string {
	return fmt.Sprintf("%T=%#v", value, value)
}
