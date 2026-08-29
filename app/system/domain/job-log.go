package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
	"time"
)

// JobLog 定时任务日志 model
type JobLog struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:job_log_id;type:bigint(20) auto_increment;primary_key;not null;comment:任务日志ID"` // 任务日志ID

	JobLogBase
}

// TableName 指定表名
func (m *JobLog) TableName() string {
	return "sys_job_log"
}

// JobLogBase 定时任务日志基础结构
type JobLogBase struct {
	JobName       string    `json:"job_name" form:"job_name" gorm:"column:job_name;type:varchar(64);not null;comment:任务名称"`             // 任务名称
	JobGroup      string    `json:"job_group" form:"job_group" gorm:"column:job_group;type:varchar(64);not null;comment:任务组名"`         // 任务组名
	JobExecutor   string    `json:"job_executor" form:"job_executor" gorm:"column:job_executor;type:varchar(64);not null;comment:任务执行器"` // 任务执行器
	InvokeTarget  string    `json:"invoke_target" form:"invoke_target" gorm:"column:invoke_target;type:varchar(500);not null;comment:调用目标字符串"` // 调用目标字符串
	JobArgs       string    `json:"job_args" form:"job_args" gorm:"column:job_args;type:varchar(255);default:'';comment:位置参数"`          // 位置参数
	JobKwargs     string    `json:"job_kwargs" form:"job_kwargs" gorm:"column:job_kwargs;type:varchar(255);default:'';comment:关键字参数"`    // 关键字参数
	JobTrigger    string    `json:"job_trigger" form:"job_trigger" gorm:"column:job_trigger;type:varchar(255);default:'';comment:任务触发器"`   // 任务触发器
	JobMessage    string    `json:"job_message" form:"job_message" gorm:"column:job_message;type:varchar(500);comment:日志信息"`            // 日志信息
	ExceptionInfo string    `json:"exception_info" form:"exception_info" gorm:"column:exception_info;type:varchar(2000);default:'';comment:异常信息"` // 异常信息
	Status        uint      `json:"status" form:"status" gorm:"column:status;type:char(1);default:0;comment:执行状态（0正常 1失败）"`            // 执行状态（0正常 1失败）
	CreateTime    time.Time `json:"create_time" form:"create_time" gorm:"column:create_time;type:datetime;comment:创建时间"`              // 创建时间
}

// JobLogSearchRequest 定时任务日志搜索请求结构体
type JobLogSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword      string `json:"keyword" form:"keyword"`             // 关键字
	JobName      string `json:"job_name" form:"job_name"`           // 任务名称
	JobGroup     string `json:"job_group" form:"job_group"`         // 任务组名
	JobExecutor  string `json:"job_executor" form:"job_executor"`   // 任务执行器
	InvokeTarget string `json:"invoke_target" form:"invoke_target"` // 调用目标字符串
	JobTrigger   string `json:"job_trigger" form:"job_trigger"`     // 任务触发器
	ID           *uint  `json:"id" form:"id"`                       // 任务日志ID
	Status       *uint  `json:"status" form:"status"`               // 执行状态（0正常 1失败）

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表
}

// JobLogCreateRequest 定时任务日志添加请求结构体
type JobLogCreateRequest struct {
	JobLogBase
}

// JobLogUpdateRequest 定时任务日志编辑请求结构体
type JobLogUpdateRequest struct {
	request.GetById
	JobLogBase
}
