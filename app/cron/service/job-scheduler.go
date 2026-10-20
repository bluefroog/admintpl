package service

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bluefroog/admintpl/app/cron/domain"
	"github.com/bluefroog/admintpl/app/cron/task"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/cron"
	"github.com/bluefroog/admintpl/common/database"
	log "github.com/sirupsen/logrus"
)

// 任务状态：与 sys_job.status 一致
const (
	JobStatusRunning = 0 // 正常（参与调度）
	JobStatusPaused  = 1 // 暂停
)

// 并发策略：与 sys_job.concurrent 一致
const (
	JobConcurrentAllow  = "0" // 允许并发
	JobConcurrentRefuse = "1" // 禁止并发
)

// 错过执行策略：与 sys_job.misfire_policy 一致
const (
	JobMisfireFireNow   = "1" // 立即执行
	JobMisfireFireOnce  = "2" // 执行一次
	JobMisfireDoNothing = "3" // 放弃执行
)

// jobTriggerManual 手动执行的日志触发来源
const jobTriggerManual = "手动执行"

// scheduledJob 调度中的任务（保存任务快照，执行期间不受任务表变更影响）
type scheduledJob struct {
	id       uint
	schedule *cron.Schedule
	job      domain.Job
	running  int32 // 正在执行的实例数，用于「禁止并发」判断
}

var (
	jobSchedulerMu   sync.RWMutex
	jobSchedulerJobs = make(map[uint]*scheduledJob)
	jobSchedulerStop chan struct{}
	jobSchedulerDone sync.WaitGroup
	jobSchedulerTick time.Time
)

// JobScheduler 定时任务调度器
//
// 采用单个常驻协程按秒扫描的轻量实现（标准库实现，不引入 Quartz/gcron 依赖）：
//   - 命中 cron 表达式的秒级时刻触发任务，任务在独立协程中执行；
//   - concurrent=1 时通过计数保证同一任务不并发执行；
//   - 启动时按 misfire_policy 补偿调度器停止期间错过的执行；
//   - 每次执行结果写入 sys_job_log。
type JobScheduler struct{}

// Start 启动调度器（重复调用只生效一次）
func (s JobScheduler) Start() {
	jobSchedulerMu.Lock()
	if jobSchedulerStop != nil {
		jobSchedulerMu.Unlock()
		log.Info("定时任务调度器已在运行，忽略重复启动")
		return
	}
	stop := make(chan struct{})
	jobSchedulerStop = stop
	// 从当前秒开始，避免启动瞬间对同一秒重复触发
	jobSchedulerTick = time.Now().Truncate(time.Second)
	jobSchedulerMu.Unlock()

	log.Info("定时任务调度器启动")
	s.Load()

	jobSchedulerDone.Add(1)
	go s.loop(stop)
}

// Stop 停止调度器（等待调度协程退出）
func (s JobScheduler) Stop() {
	jobSchedulerMu.Lock()
	stop := jobSchedulerStop
	jobSchedulerStop = nil
	jobSchedulerTick = time.Time{}
	jobSchedulerJobs = make(map[uint]*scheduledJob)
	jobSchedulerMu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	jobSchedulerDone.Wait()
	log.Info("定时任务调度器已停止")
}

// Running 调度器是否处于运行状态
func (s JobScheduler) Running() bool {
	jobSchedulerMu.RLock()
	defer jobSchedulerMu.RUnlock()
	return jobSchedulerStop != nil
}

// Scheduled 任务是否处于调度中
func (s JobScheduler) Scheduled(id uint) bool {
	jobSchedulerMu.RLock()
	defer jobSchedulerMu.RUnlock()
	_, ok := jobSchedulerJobs[id]
	return ok
}

// ScheduledCount 调度中的任务数量
func (s JobScheduler) ScheduledCount() int {
	jobSchedulerMu.RLock()
	defer jobSchedulerMu.RUnlock()
	return len(jobSchedulerJobs)
}

// Load 载入状态为「正常」的任务，并按错过执行策略补偿
func (s JobScheduler) Load() {
	if database.DB == nil {
		log.Warn("定时任务调度器：数据库未初始化，跳过任务载入")
		return
	}
	var jobs []domain.Job
	if err := database.DB.Where("status = ?", JobStatusRunning).Find(&jobs).Error; err != nil {
		log.Errorf("定时任务调度器：载入任务失败：%v", err)
		return
	}
	now := time.Now()
	loaded := 0
	for index := range jobs {
		job := jobs[index]
		item, err := newScheduledJob(&job)
		if err != nil {
			log.Warnf("定时任务调度器：任务「%s」配置不合法，已跳过：%v", job.Name, err)
			continue
		}
		s.register(item)
		loaded++
		if s.missed(&job, item.schedule, now) {
			log.Warnf("定时任务调度器：任务「%s」存在错过的执行，按「%s」策略补偿一次", job.Name, job.MisfirePolicy)
			s.execute(item, item.schedule.String()+"（错过执行补偿）")
		}
	}
	log.Infof("定时任务调度器：已载入 %d 个任务", loaded)
}

// Refresh 按任务表最新数据刷新调度（新增/修改/启停任务后调用）
func (s JobScheduler) Refresh(id uint) {
	if database.DB == nil {
		return
	}
	var job domain.Job
	if err := database.DB.Where("job_id = ?", id).First(&job).Error; err != nil {
		s.Remove(id)
		return
	}
	if job.Status != JobStatusRunning {
		s.Remove(id)
		return
	}
	item, err := newScheduledJob(&job)
	if err != nil {
		log.Warnf("定时任务调度器：任务「%s」配置不合法，已停止调度：%v", job.Name, err)
		s.Remove(id)
		return
	}
	s.register(item)
}

// Remove 摘除任务调度
func (s JobScheduler) Remove(id uint) {
	jobSchedulerMu.Lock()
	defer jobSchedulerMu.Unlock()
	delete(jobSchedulerJobs, id)
}

// RunOnce 立即执行一次任务（与任务状态无关）
func (s JobScheduler) RunOnce(id uint) error {
	if database.DB == nil {
		return errors.New("数据库未初始化，无法执行任务")
	}
	var job domain.Job
	if err := database.DB.Where("job_id = ?", id).First(&job).Error; err != nil {
		return errors.New("任务不存在")
	}
	item, err := newScheduledJob(&job)
	if err != nil {
		return err
	}
	s.execute(item, jobTriggerManual)
	return nil
}

// CleanExpiredJobLog 清理超过保留天数的任务日志
//
// 保留天数由 system.job.log-retain-days 配置，小于等于 0 表示不清理。
func (s JobScheduler) CleanExpiredJobLog() {
	days := config.Int("system.job.log-retain-days")
	if days <= 0 || database.DB == nil {
		return
	}
	result := database.DB.Where("create_time < ?", time.Now().AddDate(0, 0, -days)).Delete(&domain.JobLog{})
	if result.Error != nil {
		log.Errorf("清理过期定时任务日志失败：%v", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Infof("清理 %d 天前的定时任务日志：%d 条", days, result.RowsAffected)
	}
}

// register 注册调度（调度器未启动时忽略）
func (s JobScheduler) register(item *scheduledJob) {
	jobSchedulerMu.Lock()
	defer jobSchedulerMu.Unlock()
	if jobSchedulerStop == nil {
		return
	}
	jobSchedulerJobs[item.id] = item
}

// loop 调度主循环：按秒扫描
func (s JobScheduler) loop(stop chan struct{}) {
	defer jobSchedulerDone.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

// tick 扫描一个时间区间（循环被阻塞时按秒补扫，最多补 60 秒）
func (s JobScheduler) tick(now time.Time) {
	current := now.Truncate(time.Second)
	jobSchedulerMu.Lock()
	from := jobSchedulerTick.Add(time.Second)
	jobSchedulerTick = current
	jobSchedulerMu.Unlock()
	if from.IsZero() || from.After(current) {
		from = current
	}
	if current.Sub(from) > 60*time.Second {
		from = current.Add(-60 * time.Second)
	}
	for at := from; !at.After(current); at = at.Add(time.Second) {
		s.fire(at)
	}
}

// fire 触发该时刻命中的全部任务
func (s JobScheduler) fire(at time.Time) {
	jobSchedulerMu.RLock()
	items := make([]*scheduledJob, 0, len(jobSchedulerJobs))
	for _, item := range jobSchedulerJobs {
		items = append(items, item)
	}
	jobSchedulerMu.RUnlock()
	for _, item := range items {
		if item.schedule.Match(at) {
			s.execute(item, item.schedule.String())
		}
	}
}

// execute 按并发策略提交任务执行
func (s JobScheduler) execute(item *scheduledJob, trigger string) {
	if item.job.Concurrent == JobConcurrentRefuse {
		if !atomic.CompareAndSwapInt32(&item.running, 0, 1) {
			log.Warnf("定时任务「%s」上次执行尚未结束，按「禁止并发」策略跳过本次触发", item.job.Name)
			return
		}
	} else {
		atomic.AddInt32(&item.running, 1)
	}
	go func() {
		defer func() {
			if item.job.Concurrent == JobConcurrentRefuse {
				atomic.StoreInt32(&item.running, 0)
			} else {
				atomic.AddInt32(&item.running, -1)
			}
		}()
		s.run(&item.job, trigger)
	}()
}

// run 执行任务并写入任务日志
func (s JobScheduler) run(job *domain.Job, trigger string) {
	start := time.Now()
	log.Infof("定时任务「%s」开始执行（%s），调用目标：%s", job.Name, trigger, job.InvokeTarget)
	err := invokeTask(job)
	cost := time.Since(start)

	status, message, exception := uint(0), "执行成功", ""
	if err != nil {
		status, message, exception = 1, "执行失败", err.Error()
		log.Errorf("定时任务「%s」执行失败（耗时 %s）：%v", job.Name, cost, err)
	} else {
		log.Infof("定时任务「%s」执行成功（耗时 %s）", job.Name, cost)
	}
	s.writeLog(job, trigger, message, exception, status, start)
}

// invokeTask 执行调用目标（兜住 panic，避免影响调度循环）
func invokeTask(job *domain.Job) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("任务执行异常：%v", recovered)
		}
	}()
	return task.Invoke(job.InvokeTarget, task.SplitJobArgs(job.JobArgs)...)
}

// writeLog 写入任务执行日志
//
// status=0（成功）与 exception_info=” 都是「零值」，使用 map 写入避免被 GORM 忽略。
func (s JobScheduler) writeLog(job *domain.Job, trigger, message, exception string, status uint, start time.Time) {
	if database.DB == nil {
		return
	}
	record := map[string]interface{}{
		"JobName":       job.Name,
		"JobGroup":      job.Group,
		"JobExecutor":   job.Executor,
		"InvokeTarget":  job.InvokeTarget,
		"JobArgs":       job.JobArgs,
		"JobKwargs":     job.JobKwargs,
		"JobTrigger":    trigger,
		"JobMessage":    message,
		"ExceptionInfo": exception,
		"Status":        status,
		"CreateTime":    start,
	}
	if err := database.DB.Model(&domain.JobLog{}).Create(record).Error; err != nil {
		log.Errorf("写入定时任务日志失败：%v", err)
	}
}

// missed 判断任务是否有错过的执行（用于启动时的补偿）
func (s JobScheduler) missed(job *domain.Job, schedule *cron.Schedule, now time.Time) bool {
	if job.MisfirePolicy == JobMisfireDoNothing {
		return false
	}
	baseline := s.lastRunTime(job)
	if baseline.IsZero() {
		baseline = job.CreateTime
	}
	if baseline.IsZero() {
		baseline = now
	}
	next := schedule.Next(baseline)
	return !next.IsZero() && next.Before(now)
}

// lastRunTime 查询任务最近一次执行时间（无记录时返回零值）
func (s JobScheduler) lastRunTime(job *domain.Job) time.Time {
	if database.DB == nil {
		return time.Time{}
	}
	var row struct {
		LastTime *time.Time
	}
	err := database.DB.Model(&domain.JobLog{}).
		Select("MAX(create_time) AS last_time").
		Where("job_name = ? AND job_group = ?", job.Name, job.Group).
		Scan(&row).Error
	if err != nil || row.LastTime == nil {
		return time.Time{}
	}
	return *row.LastTime
}

// newScheduledJob 构建调度任务（校验 cron 表达式与调用目标）
func newScheduledJob(job *domain.Job) (*scheduledJob, error) {
	schedule, err := cron.Parse(job.CronExpression)
	if err != nil {
		return nil, err
	}
	if _, _, err := task.LookupTarget(job.InvokeTarget); err != nil {
		return nil, err
	}
	return &scheduledJob{id: job.ID, schedule: schedule, job: *job}, nil
}
