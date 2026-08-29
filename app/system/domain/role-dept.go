package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
)

// RoleDept 角色和部门关联 model
type RoleDept struct {
	RoleID uint `json:"role_id" form:"role_id" gorm:"column:role_id;type:bigint(20);primary_key;not null;comment:角色ID"` // 角色ID
	DeptID uint `json:"dept_id" form:"dept_id" gorm:"column:dept_id;type:bigint(20);primary_key;not null;comment:部门ID"` // 部门ID
}

// TableName 指定表名
func (m *RoleDept) TableName() string {
	return "sys_role_dept"
}

// RoleDeptSearchRequest 角色和部门关联搜索请求结构体
type RoleDeptSearchRequest struct {
	request.PageInfo
	request.SortInfo

	RoleID *uint `json:"role_id" form:"role_id"` // 角色ID
	DeptID *uint `json:"dept_id" form:"dept_id"` // 部门ID

	RoleIDIn []uint `json:"roleIDIn"` // 角色ID 数组
	DeptIDIn []uint `json:"deptIDIn"` // 部门ID 数组
}

// RoleDeptCreateRequest 角色和部门关联添加请求结构体
type RoleDeptCreateRequest struct {
	RoleID uint `json:"role_id" form:"role_id" validate:"required"` // 角色ID
	DeptID uint `json:"dept_id" form:"dept_id" validate:"required"` // 部门ID
}

// RoleDeptDeleteRequest 角色和部门关联删除请求结构体
type RoleDeptDeleteRequest struct {
	RoleID *uint `json:"role_id" form:"role_id"` // 角色ID
	DeptID *uint `json:"dept_id" form:"dept_id"` // 部门ID
}
