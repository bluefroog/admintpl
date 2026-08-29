package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
)

// RoleMenu 角色和菜单关联 model
type RoleMenu struct {
	RoleID uint `json:"role_id" form:"role_id" gorm:"column:role_id;type:bigint(20);primary_key;not null;comment:角色ID"` // 角色ID
	MenuID uint `json:"menu_id" form:"menu_id" gorm:"column:menu_id;type:bigint(20);primary_key;not null;comment:菜单ID"` // 菜单ID
}

// TableName 指定表名
func (m *RoleMenu) TableName() string {
	return "sys_role_menu"
}

// RoleMenuSearchRequest 角色和菜单关联搜索请求结构体
type RoleMenuSearchRequest struct {
	request.PageInfo
	request.SortInfo

	RoleID *uint `json:"role_id" form:"role_id"` // 角色ID
	MenuID *uint `json:"menu_id" form:"menu_id"` // 菜单ID

	RoleIDIn []uint `json:"roleIDIn"` // 角色ID 数组
	MenuIDIn []uint `json:"menuIDIn"` // 菜单ID 数组
}

// RoleMenuCreateRequest 角色和菜单关联添加请求结构体
type RoleMenuCreateRequest struct {
	RoleID uint `json:"role_id" form:"role_id" validate:"required"` // 角色ID
	MenuID uint `json:"menu_id" form:"menu_id" validate:"required"` // 菜单ID
}

// RoleMenuDeleteRequest 角色和菜单关联删除请求结构体
type RoleMenuDeleteRequest struct {
	RoleID *uint `json:"role_id" form:"role_id"` // 角色ID
	MenuID *uint `json:"menu_id" form:"menu_id"` // 菜单ID
}
