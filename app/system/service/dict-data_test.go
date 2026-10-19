package service

import (
	"testing"

	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/cache"
)

// TestDictCacheRegistered 校验字典缓存已注册到缓存监控
func TestDictCacheRegistered(t *testing.T) {
	found := false
	for _, item := range cache.Names() {
		if item == DictCacheName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("字典缓存 %s 未注册到缓存监控，当前已注册：%v", DictCacheName, cache.Names())
	}
}

// TestDictCacheSetGetEvict 校验字典缓存的读写与按类型失效
func TestDictCacheSetGetEvict(t *testing.T) {
	s := DictData{}
	defer s.EvictCache("test_dict_type")

	s.EvictCache("test_dict_type")
	if _, exists := s.GetByTypeFromCache("test_dict_type"); exists {
		t.Fatal("未写入缓存时不应命中")
	}

	var male, female domain.DictData
	male.DictDataBase = domain.DictDataBase{Label: "男", Value: "0", Type: "test_dict_type"}
	female.DictDataBase = domain.DictDataBase{Label: "女", Value: "1", Type: "test_dict_type"}
	list := []domain.DictData{male, female}
	dictCacheStore.Set("test_dict_type", list, dictCacheTTL)

	cached, exists := s.GetByTypeFromCache("test_dict_type")
	if !exists || len(cached) != 2 || cached[1].Label != "女" {
		t.Fatalf("字典缓存命中失败：exists=%v list=%v", exists, cached)
	}

	s.ClearCache()
	if _, exists := s.GetByTypeFromCache("test_dict_type"); exists {
		t.Fatal("清空缓存后不应再命中")
	}
}

// TestDictCacheMultiEvict 校验一次失效多个字典类型
func TestDictCacheMultiEvict(t *testing.T) {
	s := DictData{}
	var a, b, c domain.DictData
	a.DictDataBase = domain.DictDataBase{Type: "test_dict_a"}
	b.DictDataBase = domain.DictDataBase{Type: "test_dict_b"}
	c.DictDataBase = domain.DictDataBase{Type: "test_dict_c"}
	dictCacheStore.Set("test_dict_a", []domain.DictData{a}, dictCacheTTL)
	dictCacheStore.Set("test_dict_b", []domain.DictData{b}, dictCacheTTL)
	dictCacheStore.Set("test_dict_c", []domain.DictData{c}, dictCacheTTL)
	defer s.ClearCache()

	s.EvictCache("test_dict_a", "test_dict_b")
	if _, exists := s.GetByTypeFromCache("test_dict_a"); exists {
		t.Fatal("test_dict_a 应已被失效")
	}
	if _, exists := s.GetByTypeFromCache("test_dict_b"); exists {
		t.Fatal("test_dict_b 应已被失效")
	}
	if _, exists := s.GetByTypeFromCache("test_dict_c"); !exists {
		t.Fatal("test_dict_c 不应被误删")
	}
}
