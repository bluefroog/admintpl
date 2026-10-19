package monitor

import (
	"github.com/bluefroog/admintpl/app/monitor/controller"
	"github.com/gin-gonic/gin"
)

// Initialize 系统监控模块初始化（路由注册）
func Initialize(r *gin.Engine) {
	controller.InitRouter(r)
}
