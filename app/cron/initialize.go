package cron

import (
	"github.com/bluefroog/admintpl/app/cron/controller"
	"github.com/bluefroog/admintpl/app/cron/service"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/database"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Initialize 定时任务模块初始化（路由注册 + 调度器启动）
func Initialize(r *gin.Engine) {
	controller.InitRouter(r)

	log.Infof("cron 初始化：定时任务调度=%v", config.Bool("system.job.enabled"))

	// 清理超过保留期限的定时任务日志
	service.JobScheduler{}.CleanExpiredJobLog()

	// 启动定时任务调度器（system.job.enabled=true 时载入「正常」状态的任务）
	if config.Bool("system.job.enabled") {
		service.JobScheduler{}.Start()
	}

	// 定时任务示例数据初始化（system.seed.enabled=true 时按表补齐）
	if config.Bool("system.seed.enabled") {
		if err := (service.Seed{}).Ensure(database.DB); err != nil {
			log.Errorf("cron 基础数据初始化失败：%v", err)
		}
	}
}
