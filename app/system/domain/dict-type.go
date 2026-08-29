package domain

import (
	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// DictType 字典类型 model
type DictType struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:dict_id;type:bigint(20) auto_increment;primary_key;not null;comment:字典主键"` // 字典主键

	DictTypeBase
}

// TableName 指定表名
func (m *DictType) TableName() string {
	return "sys_dict_type"
}

// DictTypeBase 字典类型基础结构
type DictTypeBase struct {
	Name   string `json:"name" form:"name" validate:"required" gorm:"column:dict_name;type:varchar(100);not null;comment:字典名称"` // 字典名称
	Type   string `json:"type" form:"type" validate:"required" gorm:"column:dict_type;type:varchar(100);not null;comment:字典类型"` // 字典类型
	Status uint   `json:"status" form:"status" gorm:"column:status;type:char(1);default:0;comment:状态（0正常 1停用）"`                // 状态（0正常 1停用）

	core.RuoyiModel
}

// DictTypeSearchRequest 字典类型搜索请求结构体
type DictTypeSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword string `json:"keyword" form:"keyword"` // 关键字
	Name    string `json:"name" form:"name"`       // 字典名称
	Type    string `json:"type" form:"type"`       // 字典类型
	ID      *uint  `json:"id" form:"id"`           // 字典主键
	Status  *uint  `json:"status" form:"status"`   // 状态（0正常 1停用）

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表

	core.RouyiSearchRequest
}

// DictTypeCreateRequest 字典类型添加请求结构体
type DictTypeCreateRequest struct {
	DictTypeBase
}

// DictTypeUpdateRequest 字典类型编辑请求结构体
type DictTypeUpdateRequest struct {
	request.GetById
	DictTypeBase
}
