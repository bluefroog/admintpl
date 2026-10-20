package controller

import (
	"github.com/bluefroog/admintpl/app/system/middleware"
	"github.com/gin-gonic/gin"
)

// InitRouter 注册「定时任务」模块路由（统一前缀 /monitor）
// 权限标识按路径推导：/monitor/job/list -> monitor:job:list
func InitRouter(r *gin.Engine) {
	authorized := r.Group("", middleware.Auth())

	group := authorized.Group("/monitor")
	{
		// 定时任务相关接口
		job := Job{}
		group.GET("/job/list", job.List)
		group.POST("/job/create", job.Create)
		group.PUT("/job/update", job.Update)
		group.DELETE("/job/delete", job.Delete)
		group.GET("/job/detail", job.Detail)
		// 定时任务扩展接口（状态修改 / 立即执行一次 / 唯一性校验）
		group.PUT("/job/change-status", job.ChangeStatus)
		group.PUT("/job/run", job.Run)
		group.GET("/job/check-name", job.CheckNameUnique)

		// 定时任务日志相关接口
		jobLog := JobLog{}
		group.GET("/jobLog/list", jobLog.List)
		group.DELETE("/jobLog/delete", jobLog.Delete)
		group.GET("/jobLog/detail", jobLog.Detail)
		group.DELETE("/jobLog/clear", jobLog.Clear)
	}
}
