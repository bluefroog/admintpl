package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
	"time"
)

// OperationLog 操作日志 model
type OperationLog struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:oper_id;type:bigint(20) auto_increment;primary_key;not null;comment:日志主键"` // 日志主键

	OperationLogBase
}

// TableName 指定表名
func (m *OperationLog) TableName() string {
	return "sys_oper_log"
}

// OperationLogBase 操作日志基础结构
type OperationLogBase struct {
	Title         string    `json:"title" form:"title" gorm:"column:title;type:varchar(50);default:'';comment:模块标题"`                                  // 模块标题
	BusinessType  uint      `json:"business_type" form:"business_type" gorm:"column:business_type;type:int(2);default:0;comment:业务类型（0其它 1新增 2修改 3删除）"` // 业务类型（0其它 1新增 2修改 3删除）
	Method        string    `json:"method" form:"method" gorm:"column:method;type:varchar(100);default:'';comment:方法名称"`                             // 方法名称
	RequestMethod string    `json:"request_method" form:"request_method" gorm:"column:request_method;type:varchar(10);default:'';comment:请求方式"`       // 请求方式
	OperatorType  uint      `json:"operator_type" form:"operator_type" gorm:"column:operator_type;type:int(1);default:0;comment:操作类别（0其它 1后台用户 2手机端用户）"` // 操作类别（0其它 1后台用户 2手机端用户）
	OperName      string    `json:"oper_name" form:"oper_name" gorm:"column:oper_name;type:varchar(50);default:'';comment:操作人员"`                     // 操作人员
	DeptName      string    `json:"dept_name" form:"dept_name" gorm:"column:dept_name;type:varchar(50);default:'';comment:部门名称"`                     // 部门名称
	OperURL       string    `json:"oper_url" form:"oper_url" gorm:"column:oper_url;type:varchar(255);default:'';comment:请求URL"`                       // 请求URL
	OperIP        string    `json:"oper_ip" form:"oper_ip" gorm:"column:oper_ip;type:varchar(128);default:'';comment:主机地址"`                          // 主机地址
	OperLocation  string    `json:"oper_location" form:"oper_location" gorm:"column:oper_location;type:varchar(255);default:'';comment:操作地点"`        // 操作地点
	OperParam     string    `json:"oper_param" form:"oper_param" gorm:"column:oper_param;type:varchar(2000);default:'';comment:请求参数"`                // 请求参数
	JSONResult    string    `json:"json_result" form:"json_result" gorm:"column:json_result;type:varchar(2000);default:'';comment:返回参数"`             // 返回参数
	Status        uint      `json:"status" form:"status" gorm:"column:status;type:int(1);default:0;comment:操作状态（0正常 1异常）"`                          // 操作状态（0正常 1异常）
	ErrorMsg      string    `json:"error_msg" form:"error_msg" gorm:"column:error_msg;type:varchar(2000);default:'';comment:错误消息"`                   // 错误消息
	OperTime      time.Time `json:"oper_time" form:"oper_time" gorm:"column:oper_time;type:datetime;comment:操作时间"`                                  // 操作时间
	CostTime      uint      `json:"cost_time" form:"cost_time" gorm:"column:cost_time;type:bigint(20);default:0;comment:消耗时间"`                      // 消耗时间
}

// OperationLogSearchRequest 操作日志搜索请求结构体
type OperationLogSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword       string `json:"keyword" form:"keyword"`             // 关键字
	Title         string `json:"title" form:"title"`                 // 模块标题
	OperName      string `json:"oper_name" form:"oper_name"`         // 操作人员
	DeptName      string `json:"dept_name" form:"dept_name"`         // 部门名称
	OperURL       string `json:"oper_url" form:"oper_url"`           // 请求URL
	OperIP        string `json:"oper_ip" form:"oper_ip"`             // 主机地址
	RequestMethod string `json:"request_method" form:"request_method"` // 请求方式
	ID            *uint  `json:"id" form:"id"`                       // 日志主键
	Status        *uint  `json:"status" form:"status"`               // 操作状态（0正常 1异常）
	BusinessType  *uint  `json:"business_type" form:"business_type"` // 业务类型（0其它 1新增 2修改 3删除）

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表
}

// OperationLogCreateRequest 操作日志添加请求结构体
type OperationLogCreateRequest struct {
	OperationLogBase
}

// OperationLogUpdateRequest 操作日志编辑请求结构体
type OperationLogUpdateRequest struct {
	request.GetById
	OperationLogBase
}
