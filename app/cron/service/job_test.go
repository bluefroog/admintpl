package service

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluefroog/admintpl/app/cron/domain"
	"github.com/bluefroog/admintpl/app/cron/task"
	"github.com/bluefroog/admintpl/common/cron"
	"github.com/bluefroog/admintpl/common/model/core"
)

// TestValidateJob 任务配置校验
func TestValidateJob(t *testing.T) {
	job := Job{}

	// 合法任务：默认值自动补全
	data := &domain.JobBase{
		Name:           "测试任务",
		InvokeTarget:   "demoTask.noParams()",
		CronExpression: "0/10 * * * * ?",
	}
	if err := job.ValidateJob(data); err != nil {
		t.Fatalf("合法任务校验失败：%v", err)
	}
	if data.Group != "default" || data.Executor != "default" {
		t.Errorf("任务组/执行器应补全为 default，实际 %q/%q", data.Group, data.Executor)
	}
	if data.Concurrent != JobConcurrentRefuse || data.MisfirePolicy != JobMisfireDoNothing {
		t.Errorf("并发策略/错过策略应补全为默认值，实际 %q/%q", data.Concurrent, data.MisfirePolicy)
	}

	// 非法场景
	cases := []struct {
		name string
		data domain.JobBase
		want string
	}{
		{"cron 为空", domain.JobBase{InvokeTarget: "demoTask.noParams()"}, "Cron 表达式不能为空"},
		{"cron 非法", domain.JobBase{CronExpression: "0 0 0 * *", InvokeTarget: "demoTask.noParams()"}, "Cron 表达式不正确"},
		{"调用目标未注册", domain.JobBase{CronExpression: "0/10 * * * * ?", InvokeTarget: "hack.exec('rm -rf /')"}, "调用目标字符串错误"},
		{"调用目标格式错误", domain.JobBase{CronExpression: "0/10 * * * * ?", InvokeTarget: "demoTask"}, "调用目标字符串错误"},
		{"并发策略非法", domain.JobBase{CronExpression: "0/10 * * * * ?", InvokeTarget: "demoTask.noParams()", Concurrent: "9"}, "并发策略"},
		{"错过策略非法", domain.JobBase{CronExpression: "0/10 * * * * ?", InvokeTarget: "demoTask.noParams()", MisfirePolicy: "9"}, "错过执行策略"},
		{"状态非法", domain.JobBase{CronExpression: "0/10 * * * * ?", InvokeTarget: "demoTask.noParams()", Status: 3}, "任务状态"},
	}
	for _, item := range cases {
		err := job.ValidateJob(&item.data)
		if err == nil {
			t.Errorf("%s：应当校验失败", item.name)
			continue
		}
		if !strings.Contains(err.Error(), item.want) {
			t.Errorf("%s：错误信息应包含 %q，实际 %q", item.name, item.want, err.Error())
		}
	}
}

// TestSchedulerStateWithoutDB 无数据库时调度器的健壮性
func TestSchedulerStateWithoutDB(t *testing.T) {
	scheduler := JobScheduler{}
	if scheduler.Running() {
		t.Error("未启动时 Running 应为 false")
	}
	if scheduler.Scheduled(1) || scheduler.ScheduledCount() != 0 {
		t.Error("未启动时不应有调度中的任务")
	}
	// 调度器未启动时注册应被忽略
	scheduler.register(&scheduledJob{id: 99, job: domain.Job{ID: 99}})
	if scheduler.Scheduled(99) {
		t.Error("调度器未启动时不应注册任务")
	}
	scheduler.Remove(99)
	// 无数据库时手动执行给出可读错误
	if err := scheduler.RunOnce(1); err == nil {
		t.Error("无数据库时 RunOnce 应当报错")
	}
	// 无数据库时清理日志/载入任务不应 panic
	scheduler.CleanExpiredJobLog()
	scheduler.Load()
}

// TestSchedulerConcurrency 并发策略：禁止并发时同一任务不重叠执行
func TestSchedulerConcurrency(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var count int32
	task.Register("utBlock", "run", "并发测试任务", func(args []interface{}) error {
		atomic.AddInt32(&count, 1)
		started <- struct{}{}
		<-release
		return nil
	})

	item := &scheduledJob{
		id: 1,
		job: domain.Job{JobBase: domain.JobBase{
			Name: "并发测试", Concurrent: JobConcurrentRefuse, InvokeTarget: "utBlock.run()",
		}},
	}
	scheduler := JobScheduler{}
	scheduler.execute(item, "测试")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("任务未被触发")
	}
	// 上一次执行尚未结束，禁止并发时应跳过
	scheduler.execute(item, "测试")
	close(release)
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("禁止并发时任务只应执行 1 次，实际 %d 次", got)
	}

	// 允许并发时两次都会执行
	concurrent := &scheduledJob{
		id: 2,
		job: domain.Job{JobBase: domain.JobBase{
			Name: "并发允许", Concurrent: JobConcurrentAllow, InvokeTarget: "utBlock.run()",
		}},
	}
	release2 := make(chan struct{})
	task.Register("utBlock", "run2", "并发测试任务2", func(args []interface{}) error {
		atomic.AddInt32(&count, 1)
		<-release2
		return nil
	})
	concurrent.job.InvokeTarget = "utBlock.run2()"
	scheduler.execute(concurrent, "测试")
	scheduler.execute(concurrent, "测试")
	time.Sleep(100 * time.Millisecond)
	close(release2)
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&count); got != 3 {
		t.Errorf("允许并发时任务应再执行 2 次（共 3 次），实际 %d 次", got)
	}
}

// TestSchedulerMissed 错过执行策略判断
func TestSchedulerMissed(t *testing.T) {
	scheduler := JobScheduler{}
	schedule, err := cron.Parse("*/1 * * * * *") // 每秒执行
	if err != nil {
		t.Fatalf("解析 cron 失败：%v", err)
	}
	now := time.Now()

	// 放弃执行策略：不补偿
	giveUp := &domain.Job{JobBase: domain.JobBase{MisfirePolicy: JobMisfireDoNothing}}
	if scheduler.missed(giveUp, schedule, now) {
		t.Error("「放弃执行」策略不应补偿")
	}

	// 立即执行策略 + 上次执行在 10 秒前 -> 存在错过的执行
	fireNow := &domain.Job{JobBase: domain.JobBase{
		MisfirePolicy: JobMisfireFireNow,
		AuditModel:    core.AuditModel{CreateTime: now.Add(-10 * time.Second)},
	}}
	if !scheduler.missed(fireNow, schedule, now) {
		t.Error("「立即执行」策略下存在错过执行时应补偿")
	}

	// 刚刚创建的任务不应补偿
	justCreated := &domain.Job{JobBase: domain.JobBase{
		MisfirePolicy: JobMisfireFireNow,
		AuditModel:    core.AuditModel{CreateTime: now},
	}}
	if scheduler.missed(justCreated, schedule, now) {
		t.Error("刚创建的任务不应触发补偿")
	}
}
