package domain

import (
	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// Role 角色 model
type Role struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:role_id;type:bigint(20) auto_increment;primary_key;not null;comment:角色ID"` // 角色ID

	RoleBase
}

// TableName 指定表名
func (m *Role) TableName() string {
	return "sys_role"
}

type RoleBase struct {
	Key    string `json:"key" form:"key" validate:"required" gorm:"column:role_key;type:varchar(100);not null;comment:角色权限字符串"` // 角色权限字符串
	Name   string `json:"name" form:"name" validate:"required" gorm:"column:role_name;type:varchar(30);not null;comment:角色名称"`  // 角色名称
	Sort   uint   `json:"sort" form:"sort" gorm:"column:role_sort;type:int(4);not null;comment:显示顺序"`                           // 显示顺序
	Status uint   `json:"status" form:"status" gorm:"column:status;type:char(1);not null;comment:角色状态（0正常 1停用）"`                // 角色状态（0正常 1停用）

	MenuCheckStrictly       uint `json:"menu_check_strictly" form:"menu_check_strictly" gorm:"column:menu_check_strictly;type:tinyint(1);default:1;comment:菜单树选择项是否关联显示（0否 1是）"` // 菜单树选择项是否关联显示（0否 1是）
	DepartmentCheckStrictly uint `json:"dept_check_strictly" form:"dept_check_strictly" gorm:"column:dept_check_strictly;type:tinyint(1);default:1;comment:部门树选择项是否关联显示（0否 1是）"` // 部门树选择项是否关联显示（0否 1是）
	DataScope               uint `json:"data_scope" form:"data_scope" gorm:"column:data_scope;type:char(1);default:1;comment:数据范围（1：全部数据权限 2：自定数据权限 3：本部门数据权限 4：本部门及以下数据权限）"`    //数据范围（1：全部数据权限 2：自定数据权限 3：本部门数据权限 4：本部门及以下数据权限）
	DeleteFlag              uint `json:"del_flag" form:"del_flag" gorm:"column:del_flag;type:char(1);default:0;comment:删除标志（0代表存在 2代表删除）"`                                       // 删除标志（0代表存在 2代表删除）

	core.AuditModel
}

// RoleSearchRequest 后台角色搜索请求结构体
type RoleSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword                 string `json:"keyword" form:"keyword"`                         // 关键字
	Key                     string `json:"key" form:"key"`                                 // 角色权限字符串
	Name                    string `json:"name" form:"name"`                               // 角色名称
	ID                      *uint  `json:"id" form:"id"`                                   // 角色编号
	Status                  *uint  `json:"status" form:"status"`                           // 角色状态（0正常 1停用）
	DataScope               *uint  `json:"data_scope" form:"data_scope"`                   // 数据范围（1：全部数据权限 2：自定数据权限 3：本部门数据权限 4：本部门及以下数据权限）
	MenuCheckStrictly       *uint  `json:"menu_check_strictly" form:"menu_check_strictly"` // 菜单树选择项是否关联显示（0否 1是）
	DepartmentCheckStrictly *uint  `json:"dept_check_strictly" form:"dept_check_strictly"` // 部门树选择项是否关联显示（0否 1是）
	DeleteFlag              *uint  `json:"del_flag" form:"del_flag"`                       // 删除标志（0代表存在 2代表删除）

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表

	core.AuditSearchRequest
}

// RoleCreateRequest 角色添加请求结构体
type RoleCreateRequest struct {
	RoleBase
}

// RoleUpdateRequest 角色编辑请求结构体
type RoleUpdateRequest struct {
	request.GetById
	RoleBase
}

// RoleCheckNameRequest 角色名称唯一性校验请求结构体
type RoleCheckNameRequest struct {
	ID   *uint  `json:"id" form:"id"`                         // 角色ID（编辑时传入，排除自身）
	Name string `json:"name" form:"name" validate:"required"` // 角色名称
}

// RoleCheckKeyRequest 角色权限字符串唯一性校验请求结构体
type RoleCheckKeyRequest struct {
	ID  *uint  `json:"id" form:"id"`                       // 角色ID（编辑时传入，排除自身）
	Key string `json:"key" form:"key" validate:"required"` // 角色权限字符串
}

// RoleDeptTreeRequest 角色部门树请求结构体
type RoleDeptTreeRequest struct {
	RoleID uint `json:"role_id" form:"role_id"` // 角色ID，为空时返回全部未勾选的部门树
}

// RoleMenuTreeRequest 角色菜单树请求结构体
type RoleMenuTreeRequest struct {
	RoleID uint `json:"role_id" form:"role_id"` // 角色ID
}

// RoleDataScopeRequest 角色数据权限修改请求结构体
type RoleDataScopeRequest struct {
	ID        uint   `json:"id" form:"id" validate:"required"` // 角色ID
	DataScope uint   `json:"data_scope" form:"data_scope"`     // 数据范围
	DeptIDs   []uint `json:"dept_ids" form:"dept_ids"`         // 数据权限关联的部门ID列表
}

// RoleMenuAssignRequest 角色菜单授权请求结构体
type RoleMenuAssignRequest struct {
	ID      uint   `json:"id" form:"id" validate:"required"` // 角色ID
	MenuIDs []uint `json:"menu_ids" form:"menu_ids"`         // 菜单ID列表
}

// RoleUserListRequest 角色用户查询请求结构体
type RoleUserListRequest struct {
	request.PageInfo
	request.SortInfo

	RoleID   uint   `json:"role_id" form:"role_id" validate:"required"` // 角色ID
	Username string `json:"username" form:"username"`                   // 用户名
	Mobile   string `json:"mobile" form:"mobile"`                       // 手机号
}

// RoleUserAssignRequest 角色批量选择用户授权请求结构体
type RoleUserAssignRequest struct {
	RoleID  uint   `json:"role_id" form:"role_id" validate:"required"` // 角色ID
	UserIDs []uint `json:"user_ids" form:"user_ids"`                   // 用户ID列表
}

// RoleUserCancelRequest 角色取消用户授权请求结构体
type RoleUserCancelRequest struct {
	RoleID uint `json:"role_id" form:"role_id" validate:"required"` // 角色ID
	UserID uint `json:"user_id" form:"user_id" validate:"required"` // 用户ID
}

// RoleUserCancelAllRequest 角色批量取消用户授权请求结构体
type RoleUserCancelAllRequest struct {
	RoleID  uint   `json:"role_id" form:"role_id" validate:"required"` // 角色ID
	UserIDs []uint `json:"user_ids" form:"user_ids"`                   // 用户ID列表
}
