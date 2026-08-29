package domain

import (
	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// Menu 菜单 model
type Menu struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:menu_id;type:bigint(20) auto_increment;primary_key;not null;comment:菜单ID"` // 菜单ID

	MenuBase
}

// TableName 指定表名
func (m *Menu) TableName() string {
	return "sys_menu"
}

// MenuBase 菜单基础结构
type MenuBase struct {
	Name      string `json:"name" form:"name" validate:"required" gorm:"column:menu_name;type:varchar(50);not null;comment:菜单名称"`   // 菜单名称
	ParentID  uint   `json:"parent_id" form:"parent_id" gorm:"column:parent_id;type:bigint(20);default:0;comment:父菜单ID"`            // 父菜单ID
	Sort      uint   `json:"sort" form:"sort" gorm:"column:order_num;type:int(4);default:0;comment:显示顺序"`                          // 显示顺序
	Path      string `json:"path" form:"path" gorm:"column:path;type:varchar(200);default:'';comment:路由地址"`                        // 路由地址
	Component string `json:"component" form:"component" gorm:"column:component;type:varchar(255);comment:组件路径"`                    // 组件路径
	Query     string `json:"query" form:"query" gorm:"column:query;type:varchar(255);comment:路由参数"`                               // 路由参数
	RouteName string `json:"route_name" form:"route_name" gorm:"column:route_name;type:varchar(50);default:'';comment:路由名称"`       // 路由名称
	IsFrame   uint   `json:"is_frame" form:"is_frame" gorm:"column:is_frame;type:int(1);default:1;comment:是否为外链（0是 1否）"`            // 是否为外链（0是 1否）
	IsCache   uint   `json:"is_cache" form:"is_cache" gorm:"column:is_cache;type:int(1);default:0;comment:是否缓存（0缓存 1不缓存）"`        // 是否缓存（0缓存 1不缓存）
	MenuType  string `json:"menu_type" form:"menu_type" gorm:"column:menu_type;type:char(1);default:'';comment:菜单类型（M目录 C菜单 F按钮）"` // 菜单类型（M目录 C菜单 F按钮）
	Visible   string `json:"visible" form:"visible" gorm:"column:visible;type:char(1);default:'0';comment:菜单状态（0显示 1隐藏）"`          // 菜单状态（0显示 1隐藏）
	Status    uint   `json:"status" form:"status" gorm:"column:status;type:char(1);default:0;comment:菜单状态（0正常 1停用）"`               // 菜单状态（0正常 1停用）
	Perms     string `json:"perms" form:"perms" gorm:"column:perms;type:varchar(100);comment:权限标识"`                               // 权限标识
	Icon      string `json:"icon" form:"icon" gorm:"column:icon;type:varchar(100);default:'#';comment:菜单图标"`                      // 菜单图标

	core.RuoyiModel
}

// MenuSearchRequest 菜单搜索请求结构体
type MenuSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword  string `json:"keyword" form:"keyword"`   // 关键字
	Name     string `json:"name" form:"name"`         // 菜单名称
	Path     string `json:"path" form:"path"`         // 路由地址
	Perms    string `json:"perms" form:"perms"`       // 权限标识
	ParentID *uint  `json:"parent_id" form:"parent_id"` // 父菜单ID
	ID       *uint  `json:"id" form:"id"`             // 菜单ID
	Status   *uint  `json:"status" form:"status"`     // 菜单状态（0正常 1停用）
	MenuType string `json:"menu_type" form:"menu_type"` // 菜单类型（M目录 C菜单 F按钮）

	IDNotIn   []uint `json:"idNotIn"`   // 不包含的ID列表
	IDIn      []uint `json:"idIn"`      // 包含的ID列表
	ParentIDIn []uint `json:"parentIDIn"` // 父菜单ID 数组

	core.RouyiSearchRequest
}

// MenuCreateRequest 菜单添加请求结构体
type MenuCreateRequest struct {
	MenuBase
}

// MenuUpdateRequest 菜单编辑请求结构体
type MenuUpdateRequest struct {
	request.GetById
	MenuBase
}
