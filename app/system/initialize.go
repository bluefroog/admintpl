package system

import (
	"github.com/bluefroog/admintpl/app/system/controller"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Initialize(r *gin.Engine) {
	// 注册路由
	controller.InitRouter(r)
	// 注册DB
	domain.InitDB(r)
	// 输出关键开关，便于排查“配置未随部署更新”这类问题
	log.Infof("system 初始化：登录认证=%v 权限校验=%v 图形验证码=%v 数据权限=%v 基础数据初始化=%v 定时任务调度=%v",
		config.Bool("system.auth.enabled"),
		config.Bool("system.auth.permission-enabled"),
		config.Bool("system.captcha.enabled"),
		config.Bool("system.data-scope.enabled"),
		config.Bool("system.seed.enabled"),
		config.Bool("system.job.enabled"),
	)
	// 清理过期的在线用户记录
	service.Auth{}.CleanExpiredOnline()
	// 基础数据初始化（system.seed.enabled=true 时按表补齐，已存在的数据不会重复写入）
	if config.Bool("system.seed.enabled") {
		service.Seed{}.Ensure()
	}
	// 清理超过保留期限的定时任务日志
	service.JobScheduler{}.CleanExpiredJobLog()
	// 启动定时任务调度器（system.job.enabled=true 时载入「正常」状态的任务）
	if config.Bool("system.job.enabled") {
		service.JobScheduler{}.Start()
	}
}
