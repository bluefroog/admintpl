package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluefroog/admintpl/app/monitor/domain"
	"github.com/bluefroog/admintpl/common/cache"
)

// Cache 缓存监控服务
//
// 说明：本项目使用进程内内存缓存（common/cache），不依赖 Redis，
// 因此这里统计的是各内存缓存实例的键数量与读写情况
type Cache struct{}

// cacheValueMaxLength 缓存值展示的最大长度（超出截断，避免响应体过大）
const cacheValueMaxLength = 200

// Info 缓存监控信息
// @author: [bluefrog](https://github.com/freewu)
// @function: Info
// @description: 汇总各内存缓存的键数量与读写统计
// @return: *domain.CacheInfo 缓存监控信息
func (s Cache) Info() *domain.CacheInfo {
	caches := cache.Registered()
	stats := make([]domain.CacheStats, 0, len(caches))
	var dbSize int
	var hits, misses, sets, deletes uint64
	for _, item := range caches {
		itemStats := item.Stats()
		dbSize += itemStats.KeyCount
		hits += itemStats.Hits
		misses += itemStats.Misses
		sets += itemStats.Sets
		deletes += itemStats.Deletes
		stats = append(stats, domain.CacheStats{
			Name:            itemStats.Name,
			KeyCount:        itemStats.KeyCount,
			Hits:            itemStats.Hits,
			Misses:          itemStats.Misses,
			Sets:            itemStats.Sets,
			Deletes:         itemStats.Deletes,
			Expired:         itemStats.Expired,
			CleanupInterval: itemStats.CleanupInterval.String(),
			CreatedAt:       itemStats.CreatedAt.Format("2006-01-02 15:04:05"),
			Uptime:          formatDuration(itemStats.Uptime),
		})
	}
	return &domain.CacheInfo{
		Info:   stats,
		DbSize: dbSize,
		CommandStats: []domain.CacheCommandStat{
			{Name: "get", Value: hits},
			{Name: "miss", Value: misses},
			{Name: "set", Value: sets},
			{Name: "del", Value: deletes},
		},
		Names: cache.Names(),
	}
}

// List 缓存列表
// @author: [bluefrog](https://github.com/freewu)
// @function: List
// @description: 分页查询内存缓存中的键值（按缓存名称/缓存键过滤）
// @param: searchParams *domain.CacheListSearchRequest 查询参数
// @return: err error, list []domain.CacheItem, total int64
func (s Cache) List(searchParams *domain.CacheListSearchRequest) (err error, list []domain.CacheItem, total int64) {
	caches := cache.Registered()
	if searchParams.CacheName != "" {
		item, exists := cache.Find(searchParams.CacheName)
		if !exists {
			return errors.New("缓存名称不存在: " + searchParams.CacheName), []domain.CacheItem{}, 0
		}
		caches = []*cache.Cache{item}
	}

	items := make([]domain.CacheItem, 0, 16)
	for _, item := range caches {
		for _, entry := range item.Entries() {
			if searchParams.CacheKey != "" && !strings.Contains(entry.Key, searchParams.CacheKey) {
				continue
			}
			items = append(items, domain.CacheItem{
				CacheName:  item.Stats().Name,
				CacheKey:   entry.Key,
				CacheValue: formatCacheValue(entry.Value),
				Expiration: formatExpiration(entry.Expiration),
			})
		}
	}

	total = int64(len(items))
	page, pageSize := searchParams.Page, searchParams.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return nil, []domain.CacheItem{}, total
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return nil, items[start:end], total
}

// Clear 清空缓存
// @author: [bluefrog](https://github.com/freewu)
// @function: Clear
// @description: 清空指定缓存，cacheName 为空时清空全部缓存
// @param: cacheName string 缓存名称
// @return: err error
func (s Cache) Clear(cacheName string) error {
	if cacheName == "" {
		cache.ClearAll()
		return nil
	}
	item, exists := cache.Find(cacheName)
	if !exists {
		return errors.New("缓存名称不存在: " + cacheName)
	}
	item.Clear()
	return nil
}

// formatCacheValue 格式化缓存值（超长截断）
func formatCacheValue(value interface{}) string {
	text := fmt.Sprintf("%v", value)
	if len(text) > cacheValueMaxLength {
		return text[:cacheValueMaxLength] + "..."
	}
	return text
}

// formatExpiration 格式化过期时间
func formatExpiration(expiration int64) string {
	if expiration <= 0 {
		return ""
	}
	return time.UnixMilli(expiration).Format("2006-01-02 15:04:05")
}

// formatDuration 格式化运行时长
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分钟", int(d.Minutes()))
	}
	return fmt.Sprintf("%d小时%d分钟", int(d.Hours()), int(d.Minutes())%60)
}
