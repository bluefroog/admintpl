package controller

import (
	"github.com/bluefroog/admintpl/app/system/middleware"
	"github.com/gin-gonic/gin"
)

// InitRouter 注册代码生成模块路由（/tool/gen/**）
// 权限标识由路径推导（middleware.PathPermission），如 /tool/gen/list -> tool:gen:list
func InitRouter(r *gin.Engine) {
	authorized := r.Group("", middleware.Auth())

	group := authorized.Group("/tool/gen")
	{
		gen := Gen{}
		// 代码生成业务表
		group.GET("/list", gen.List)
		group.GET("/detail", gen.Detail)
		group.POST("/import", gen.Import)
		group.PUT("/update", gen.Update)
		group.DELETE("/delete", gen.Delete)
		// 数据库表（未导入）与字段
		group.GET("/db-list", gen.DbList)
		group.GET("/column-list", gen.ColumnList)
		// 同步数据库表结构
		group.POST("/synch-db", gen.SynchDb)
	}
}
