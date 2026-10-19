package service

import (
	"testing"

	"github.com/bluefroog/admintpl/common/cache"
	"github.com/bluefroog/admintpl/common/database"
)

// TestConfigCacheRegistered 校验参数缓存已注册到缓存监控
func TestConfigCacheRegistered(t *testing.T) {
	found := false
	for _, item := range cache.Names() {
		if item == ConfigCacheName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("参数缓存 %s 未注册到缓存监控，当前已注册：%v", ConfigCacheName, cache.Names())
	}
}

// TestConfigCacheSetGetEvict 校验参数缓存的读写与失效
func TestConfigCacheSetGetEvict(t *testing.T) {
	s := Config{}
	defer s.EvictCache("test.config.key")

	s.EvictCache("test.config.key")
	if v, exists := s.GetByKeyFromCache("test.config.key"); exists {
		t.Fatalf("未写入缓存时不应命中，实际值：%q", v)
	}

	configCacheStore.Set("test.config.key", "中文参数值", configCacheTTL)
	value, exists := s.GetByKeyFromCache("test.config.key")
	if !exists || value != "中文参数值" {
		t.Fatalf("缓存命中失败：exists=%v value=%q", exists, value)
	}

	s.EvictCache("test.config.key")
	if _, exists := s.GetByKeyFromCache("test.config.key"); exists {
		t.Fatal("失效后不应再命中缓存")
	}
}

// TestConfigGetByKeyUsesCache 校验 GetByKey 命中缓存时不再访问数据库
// 数据库句柄置空，若走查库分支会 panic，以此证明命中的是缓存
func TestConfigGetByKeyUsesCache(t *testing.T) {
	s := Config{}
	defer s.EvictCache("test.config.cache.hit")
	s.EvictCache("test.config.cache.hit")
	configCacheStore.Set("test.config.cache.hit", "from-cache", configCacheTTL)

	origin := database.DB
	database.DB = nil
	defer func() { database.DB = origin }()

	err, value := s.GetByKey("test.config.cache.hit")
	if err != nil {
		t.Fatalf("命中缓存时不应返回错误：%v", err)
	}
	if value != "from-cache" {
		t.Fatalf("期望取到缓存值 from-cache，实际 %q", value)
	}
}

// TestConfigEvictCacheIgnoreEmpty 校验空键名不会误删缓存
func TestConfigEvictCacheIgnoreEmpty(t *testing.T) {
	s := Config{}
	configCacheStore.Set("test.config.keep", "keep", configCacheTTL)
	defer s.EvictCache("test.config.keep")

	s.EvictCache("")
	if _, exists := s.GetByKeyFromCache("test.config.keep"); !exists {
		t.Fatal("传入空键名不应删除其它缓存")
	}
}
