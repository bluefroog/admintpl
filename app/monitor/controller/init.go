package controller

import (
	"github.com/bluefroog/admintpl/app/system/middleware"
	"github.com/gin-gonic/gin"
)

// InitRouter 注册「系统监控」模块路由（统一前缀 /monitor）
// 权限标识与若依保持一致：/monitor/online/list -> monitor:online:list
func InitRouter(r *gin.Engine) {
	authorized := r.Group("", middleware.Auth())

	group := authorized.Group("/monitor")
	{
		// 服务监控接口
		server := Server{}
		group.GET("/server", server.Info)

		// 缓存监控 / 缓存列表接口
		cache := Cache{}
		group.GET("/cache", cache.Info)
		group.GET("/cache/list", cache.List)
		group.DELETE("/cache/clear", cache.Clear)

		// 在线用户相关接口
		userOnline := UserOnline{}
		group.GET("/online/list", userOnline.List)
		group.DELETE("/online/delete", userOnline.Delete)
		group.GET("/online/detail", userOnline.Detail)
		group.DELETE("/online/force-logout", userOnline.ForceLogout)

		// 登录日志相关接口
		loginLog := LoginLog{}
		group.GET("/logininfor/list", loginLog.List)
		group.DELETE("/logininfor/delete", loginLog.Delete)
		group.GET("/logininfor/detail", loginLog.Detail)
		group.DELETE("/logininfor/clear", loginLog.Clear)

		// 操作日志相关接口
		operationLog := OperationLog{}
		group.GET("/operlog/list", operationLog.List)
		group.DELETE("/operlog/delete", operationLog.Delete)
		group.GET("/operlog/detail", operationLog.Detail)
		group.DELETE("/operlog/clear", operationLog.Clear)
	}
}
