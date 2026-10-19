package service

import (
	"testing"

	"github.com/bluefroog/admintpl/app/monitor/domain"
	"github.com/bluefroog/admintpl/common/cache"
	"github.com/bluefroog/admintpl/common/model/request"
)

// pageInfo 构造分页参数
func pageInfo(page, pageSize int) request.PageInfo {
	return request.PageInfo{Page: page, PageSize: pageSize}
}

func TestServerInfo(t *testing.T) {
	info := (Server{}).Info()
	if info == nil {
		t.Fatal("Server.Info() 返回 nil")
	}
	if info.CPU.Num <= 0 {
		t.Errorf("CPU.Num = %d; want > 0", info.CPU.Num)
	}
	if info.Go.Name != "GoLang" || info.Go.Version == "" {
		t.Errorf("Go 运行时信息异常：%+v", info.Go)
	}
	if info.Sys.OSArch == "" {
		t.Errorf("系统架构不应为空：%+v", info.Sys)
	}
}

func TestCacheInfoAndList(t *testing.T) {
	store := cache.New(0)
	cache.Register("monitor-service-test", store)
	store.Set("k1", "v1", 0)
	store.Set("k2", "v2", 0)

	info := (Cache{}).Info()
	if info.DbSize < 2 {
		t.Errorf("DbSize = %d; want >= 2", info.DbSize)
	}
	names := make(map[string]bool)
	for _, name := range info.Names {
		names[name] = true
	}
	if !names["monitor-service-test"] {
		t.Errorf("Names = %v; want 包含 monitor-service-test", info.Names)
	}
	var found bool
	for _, item := range info.Info {
		if item.Name == "monitor-service-test" && item.KeyCount == 2 {
			found = true
			if item.CleanupInterval == "" || item.CreatedAt == "" || item.Uptime == "" {
				t.Errorf("缓存统计字段不完整：%+v", item)
			}
		}
	}
	if !found {
		t.Errorf("Info() 未包含 monitor-service-test 统计：%+v", info.Info)
	}
	if len(info.CommandStats) != 4 {
		t.Errorf("CommandStats 数量 = %d; want 4", len(info.CommandStats))
	}

	// 查询全部
	err, list, total := (Cache{}).List(&domain.CacheListSearchRequest{})
	if err != nil {
		t.Fatalf("List() 出错: %v", err)
	}
	if total < 2 {
		t.Errorf("total = %d; want >= 2", total)
	}
	if len(list) != int(total) {
		t.Errorf("默认分页应返回全部：len(list)=%d total=%d", len(list), total)
	}

	// 按缓存名称 + 缓存键过滤
	err, list, total = (Cache{}).List(&domain.CacheListSearchRequest{CacheName: "monitor-service-test", CacheKey: "k1"})
	if err != nil {
		t.Fatalf("List() 出错: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].CacheKey != "k1" || list[0].CacheValue != "v1" {
		t.Fatalf("过滤结果不符合预期：total=%d list=%+v", total, list)
	}
	if list[0].CacheName != "monitor-service-test" {
		t.Errorf("CacheName = %s", list[0].CacheName)
	}

	// 分页
	err, list, total = (Cache{}).List(&domain.CacheListSearchRequest{CacheName: "monitor-service-test", PageInfo: pageInfo(1, 1)})
	if err != nil {
		t.Fatalf("List() 出错: %v", err)
	}
	if total != 2 || len(list) != 1 {
		t.Errorf("分页结果不符合预期：total=%d len(list)=%d", total, len(list))
	}
	err, list, _ = (Cache{}).List(&domain.CacheListSearchRequest{CacheName: "monitor-service-test", PageInfo: pageInfo(5, 10)})
	if err != nil || len(list) != 0 {
		t.Errorf("超出范围的页码应返回空列表：err=%v list=%v", err, list)
	}

	// 不存在的缓存名称
	if err, _, _ = (Cache{}).List(&domain.CacheListSearchRequest{CacheName: "not-exist"}); err == nil {
		t.Error("查询不存在的缓存名称应报错")
	}

	// 清空指定缓存
	if err := (Cache{}).Clear("monitor-service-test"); err != nil {
		t.Fatalf("Clear() 出错: %v", err)
	}
	if count := store.Count(); count != 0 {
		t.Errorf("Clear() 后 Count = %d; want 0", count)
	}
	if err := (Cache{}).Clear("not-exist"); err == nil {
		t.Error("清空不存在的缓存应报错")
	}
	if err := (Cache{}).Clear(""); err != nil {
		t.Errorf("Clear(\"\") 应清空全部缓存且不报错，得到 %v", err)
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := formatCacheValue("abc"); got != "abc" {
		t.Errorf("formatCacheValue = %s", got)
	}
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	if got := formatCacheValue(string(long)); len(got) != cacheValueMaxLength+3 {
		t.Errorf("超长缓存值应截断，长度 = %d", len(got))
	}
	if got := formatExpiration(0); got != "" {
		t.Errorf("永不过期的缓存应返回空串，得到 %s", got)
	}
	if got := formatExpiration(1700000000000); got == "" {
		t.Error("过期时间不应为空")
	}
	if got := formatDuration(0); got != "0秒" {
		t.Errorf("formatDuration(0) = %s", got)
	}
}
