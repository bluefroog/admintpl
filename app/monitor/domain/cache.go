package domain

import "github.com/bluefroog/admintpl/common/model/request"

// CacheStats 单个缓存的统计信息
type CacheStats struct {
	Name            string `json:"name"`            // 缓存名称
	Driver          string `json:"driver"`          // 缓存后端：file / memory / memcache / redis
	KeyCount        int    `json:"keyCount"`        // 缓存键数量
	KeysSupported   bool   `json:"keysSupported"`   // 后端是否支持枚举缓存键（memcache 不支持）
	Hits            uint64 `json:"hits"`            // 命中次数
	Misses          uint64 `json:"misses"`          // 未命中次数
	Sets            uint64 `json:"sets"`            // 写入次数
	Deletes         uint64 `json:"deletes"`         // 删除次数
	Expired         uint64 `json:"expired"`         // 过期失效次数
	CleanupInterval string `json:"cleanupInterval"` // 过期清理间隔
	CreatedAt       string `json:"createdAt"`       // 创建时间
	Uptime          string `json:"uptime"`          // 运行时长
}

// CacheCommandStat 缓存读写命令统计
type CacheCommandStat struct {
	Name  string `json:"name"`  // 命令名称
	Value uint64 `json:"value"` // 执行次数
}

// CacheInfo 缓存监控信息
type CacheInfo struct {
	Driver       string             `json:"driver"`       // 缓存后端：file / memory / memcache / redis
	Info         []CacheStats       `json:"info"`         // 各缓存统计
	DbSize       int                `json:"dbSize"`       // 缓存键总数
	CommandStats []CacheCommandStat `json:"commandStats"` // 读写命令统计
	Names        []string           `json:"names"`        // 缓存名称列表
}

// CacheItem 缓存列表条目
type CacheItem struct {
	CacheName  string `json:"cacheName"`  // 缓存名称
	CacheKey   string `json:"cacheKey"`   // 缓存键
	CacheValue string `json:"cacheValue"` // 缓存值
	Expiration string `json:"expiration"` // 过期时间，永不过期为空
}

// CacheListSearchRequest 缓存列表查询参数
type CacheListSearchRequest struct {
	CacheName string `form:"cacheName"` // 缓存名称，为空查询全部
	CacheKey  string `form:"cacheKey"`  // 缓存键（模糊匹配）
	request.PageInfo
}

// CacheClearRequest 清空缓存请求参数
type CacheClearRequest struct {
	CacheName string `form:"cacheName"` // 缓存名称，为空清空全部缓存
}
