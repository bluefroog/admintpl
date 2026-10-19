package cache

import (
	"testing"
	"time"
)

func TestCacheSetGetStats(t *testing.T) {
	c := New(0)
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
	c := New(0)
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
	c := New(0)
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

func TestCacheRegistry(t *testing.T) {
	c := New(0)
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
	c := New(time.Millisecond)
	c.Set("a", 1, 0)
	c.Close()
	c.Close() // 重复关闭不应 panic
}
