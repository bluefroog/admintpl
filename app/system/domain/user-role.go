package domain

import (
	"github.com/bluefroog/admintpl/common/model/request"
)

// UserRole 用户和角色关联 model
type UserRole struct {
	UserID uint `json:"user_id" form:"user_id" gorm:"column:user_id;type:bigint(20);primary_key;not null;comment:用户ID"` // 用户ID
	RoleID uint `json:"role_id" form:"role_id" gorm:"column:role_id;type:bigint(20);primary_key;not null;comment:角色ID"` // 角色ID
}

// TableName 指定表名
func (m *UserRole) TableName() string {
	return "sys_user_role"
}

// UserRoleSearchRequest 用户和角色关联搜索请求结构体
type UserRoleSearchRequest struct {
	request.PageInfo
	request.SortInfo

	UserID *uint `json:"user_id" form:"user_id"` // 用户ID
	RoleID *uint `json:"role_id" form:"role_id"` // 角色ID

	UserIDIn []uint `json:"userIDIn"` // 用户ID 数组
	RoleIDIn []uint `json:"roleIDIn"` // 角色ID 数组
}

// UserRoleCreateRequest 用户和角色关联添加请求结构体
type UserRoleCreateRequest struct {
	UserID uint `json:"user_id" form:"user_id" validate:"required"` // 用户ID
	RoleID uint `json:"role_id" form:"role_id" validate:"required"` // 角色ID
}

// UserRoleDeleteRequest 用户和角色关联删除请求结构体
type UserRoleDeleteRequest struct {
	UserID *uint `json:"user_id" form:"user_id"` // 用户ID
	RoleID *uint `json:"role_id" form:"role_id"` // 角色ID
}
