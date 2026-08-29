package domain

import (
	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// Job 定时任务 model
type Job struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:job_id;type:bigint(20) auto_increment;primary_key;not null;comment:任务ID"` // 任务ID

	JobBase
}

// TableName 指定表名
func (m *Job) TableName() string {
	return "sys_job"
}

// JobBase 定时任务基础结构
type JobBase struct {
	Name           string `json:"name" form:"name" validate:"required" gorm:"column:job_name;type:varchar(64);not null;comment:任务名称"`                             // 任务名称
	Group          string `json:"group" form:"group" gorm:"column:job_group;type:varchar(64);default:'default';comment:任务组名"`                                    // 任务组名
	Executor       string `json:"executor" form:"executor" gorm:"column:job_executor;type:varchar(64);default:'default';comment:任务执行器"`                          // 任务执行器
	InvokeTarget   string `json:"invoke_target" form:"invoke_target" validate:"required" gorm:"column:invoke_target;type:varchar(500);not null;comment:调用目标字符串"` // 调用目标字符串
	JobArgs        string `json:"job_args" form:"job_args" gorm:"column:job_args;type:varchar(255);default:'';comment:位置参数"`                                    // 位置参数
	JobKwargs      string `json:"job_kwargs" form:"job_kwargs" gorm:"column:job_kwargs;type:varchar(255);default:'';comment:关键字参数"`                            // 关键字参数
	CronExpression string `json:"cron_expression" form:"cron_expression" gorm:"column:cron_expression;type:varchar(255);default:'';comment:cron执行表达式"`           // cron执行表达式
	MisfirePolicy  string `json:"misfire_policy" form:"misfire_policy" gorm:"column:misfire_policy;type:varchar(20);default:'3';comment:计划执行错误策略（1立即执行 2执行一次 3放弃执行）"` // 计划执行错误策略（1立即执行 2执行一次 3放弃执行）
	Concurrent     string `json:"concurrent" form:"concurrent" gorm:"column:concurrent;type:char(1);default:'1';comment:是否并发执行（0允许 1禁止）"`                      // 是否并发执行（0允许 1禁止）
	Status         uint   `json:"status" form:"status" gorm:"column:status;type:char(1);default:0;comment:状态（0正常 1暂停）"`                                         // 状态（0正常 1暂停）

	core.RuoyiModel
}

// JobSearchRequest 定时任务搜索请求结构体
type JobSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword      string `json:"keyword" form:"keyword"`             // 关键字
	Name         string `json:"name" form:"name"`                   // 任务名称
	Group        string `json:"group" form:"group"`                 // 任务组名
	Executor     string `json:"executor" form:"executor"`           // 任务执行器
	InvokeTarget string `json:"invoke_target" form:"invoke_target"` // 调用目标字符串
	ID           *uint  `json:"id" form:"id"`                       // 任务ID
	Status       *uint  `json:"status" form:"status"`               // 状态（0正常 1暂停）

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表

	core.RouyiSearchRequest
}

// JobCreateRequest 定时任务添加请求结构体
type JobCreateRequest struct {
	JobBase
}

// JobUpdateRequest 定时任务编辑请求结构体
type JobUpdateRequest struct {
	request.GetById
	JobBase
}
