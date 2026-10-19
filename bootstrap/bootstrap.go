package bootstrap

import (
	"github.com/bluefroog/admintpl/app/cron"
	crondomain "github.com/bluefroog/admintpl/app/cron/domain"
	"github.com/bluefroog/admintpl/app/monitor"
	monitordomain "github.com/bluefroog/admintpl/app/monitor/domain"
	"github.com/bluefroog/admintpl/app/swagger"
	"github.com/bluefroog/admintpl/app/system"
	systemdomain "github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/database"
	"github.com/gin-gonic/gin"
)

// Bootstrap 应用启动：数据库 -> 表结构迁移 -> 各业务模块初始化
func Bootstrap(r *gin.Engine) error {
	// 数据库连接（database.admin.*）
	database.InitDB()
	// 表结构迁移（各模块 domain.Models() 声明自己的表）
	database.AutoMigrateAll(
		systemdomain.Models(),  // 系统管理
		monitordomain.Models(), // 系统监控
		crondomain.Models(),    // 定时任务
	)

	// todo 后期可以做成配置反射调用
	system.Initialize(r)  // 系统管理服务（含登录认证）
	monitor.Initialize(r) // 系统监控服务
	cron.Initialize(r)    // 定时任务服务
	swagger.Initialize(r) // swagger 服务
	return nil
}
