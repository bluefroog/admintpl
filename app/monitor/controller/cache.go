package controller

import (
	"github.com/bluefroog/admintpl/app/monitor/domain"
	"github.com/bluefroog/admintpl/app/monitor/service"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/gin-gonic/gin"
)

type Cache struct{}

var (
	cacheService = service.Cache{}
)

// Info 获取缓存监控信息
// @Summary 获取缓存监控信息
// @Description 缓存监控：各内存缓存的键数量与读写统计
// @Tags 缓存监控
// @Success 0 {object} response.Response{data=domain.CacheInfo} "{"code": 0, "msg": "success","data": { "info": [], "dbSize": 0, "commandStats": [], "names": [] } }"
// @Router /monitor/cache [get]
// @Security
func (u *Cache) Info(c *gin.Context) {
	response.OK(cacheService.Info(), c)
}

// List 获取缓存列表
// @Summary 获取缓存列表
// @Description 缓存列表：分页查询内存缓存中的键值
// @Tags 缓存列表
// @Param data query domain.CacheListSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.CacheItem}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /monitor/cache/list [get]
// @Security
func (u *Cache) List(c *gin.Context) {
	var searchParams domain.CacheListSearchRequest
	// 获取查询参数
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	// 分页参数归一（响应中回显生效值）
	if searchParams.Page <= 0 {
		searchParams.Page = 1
	}
	if searchParams.PageSize <= 0 {
		searchParams.PageSize = 20
	}
	err, list, total := cacheService.List(&searchParams)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(resp.PageResult{
		List:     list,
		Total:    total,
		Page:     searchParams.Page,
		PageSize: searchParams.PageSize,
	}, c)
}

// Clear 清空缓存
// @Summary 清空缓存
// @Description 清空缓存，cacheName 为空时清空全部缓存
// @Tags 缓存监控
// @Param data query domain.CacheClearRequest false "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/cache/clear [delete]
// @Security
func (u *Cache) Clear(c *gin.Context) {
	var param domain.CacheClearRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := cacheService.Clear(param.CacheName); err != nil {
		response.Fail(-1, "清空失败: "+err.Error(), c)
		return
	}
	response.Success("清空成功", c)
}
