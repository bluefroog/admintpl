package service

import (
	"time"

	"github.com/bluefroog/admintpl/app/cron/domain"
	"github.com/bluefroog/admintpl/common/database"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Seed 定时任务基础数据初始化服务
type Seed struct{}

// jobSeed 定时任务种子数据
type jobSeed struct {
	name           string
	group          string
	executor       string
	invokeTarget   string
	jobArgs        string
	cronExpression string
	misfirePolicy  string
	concurrent     string
	status         uint
}

// jobSeeds 内置示例任务（默认「暂停」，不会自动执行；可在定时任务页面启用或立即执行一次）
var jobSeeds = []jobSeed{
	{
		name: "系统默认（无参）", group: "default", executor: "default",
		invokeTarget: "demoTask.noParams()", cronExpression: "0/10 * * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentRefuse, status: JobStatusPaused,
	},
	{
		name: "系统默认（有参）", group: "default", executor: "default",
		invokeTarget: "demoTask.params('demo')", cronExpression: "0/15 * * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentRefuse, status: JobStatusPaused,
	},
	{
		name: "系统默认（多参）", group: "default", executor: "default",
		invokeTarget: "demoTask.multipleParams('demo', true, 2000, 1.0)", cronExpression: "0/20 * * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentRefuse, status: JobStatusPaused,
	},
	{
		name: "示例任务（无参，job_args 传参）", group: "default", executor: "default",
		invokeTarget: "testTask.test1()", jobArgs: "hello|2000", cronExpression: "0 0/5 * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentAllow, status: JobStatusPaused,
	},
}

// Ensure 初始化定时任务示例数据（表为空才写入，保证幂等）
func (s Seed) Ensure(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	empty, err := database.TableEmpty(db, &domain.Job{})
	if err != nil || !empty {
		return err
	}
	log.Info("开始初始化定时任务示例数据……")
	if err = s.seedJobs(db); err != nil {
		log.Errorf("定时任务示例数据初始化失败：%v", err)
		return err
	}
	log.Info("定时任务示例数据初始化完成")
	return nil
}

// seedJobs 初始化定时任务示例数据
func (s Seed) seedJobs(tx *gorm.DB) error {
	now := time.Now()
	for _, item := range jobSeeds {
		// 使用 map 写入：status=0/concurrent="0" 等零值会被 GORM 的 default 标签忽略
		record := map[string]interface{}{
			"Name":           item.name,
			"Group":          item.group,
			"Executor":       item.executor,
			"InvokeTarget":   item.invokeTarget,
			"JobArgs":        item.jobArgs,
			"JobKwargs":      "",
			"CronExpression": item.cronExpression,
			"MisfirePolicy":  item.misfirePolicy,
			"Concurrent":     item.concurrent,
			"Status":         item.status,
			"Remark":         "系统内置任务",
			"CreateBy":       "system",
			"CreateTime":     now,
			"UpdateBy":       "system",
			"UpdateTime":     now,
		}
		if err := tx.Model(&domain.Job{}).Create(record).Error; err != nil {
			return err
		}
	}
	return nil
}
