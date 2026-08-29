package domain

import (
	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// DictData 字典数据 model
type DictData struct {
	//core.Model
	ID uint `json:"id" form:"id" gorm:"column:dict_code;type:bigint(20) auto_increment;primary_key;not null;comment:字典编码"` // 字典编码

	DictDataBase
}

// TableName 指定表名
func (m *DictData) TableName() string {
	return "sys_dict_data"
}

// DictDataBase 字典数据基础结构
type DictDataBase struct {
	Sort      uint   `json:"sort" form:"sort" gorm:"column:dict_sort;type:int(4);default:0;comment:字典排序"`                          // 字典排序
	Label     string `json:"label" form:"label" validate:"required" gorm:"column:dict_label;type:varchar(100);not null;comment:字典标签"` // 字典标签
	Value     string `json:"value" form:"value" validate:"required" gorm:"column:dict_value;type:varchar(100);not null;comment:字典键值"` // 字典键值
	Type      string `json:"type" form:"type" validate:"required" gorm:"column:dict_type;type:varchar(100);not null;comment:字典类型"`   // 字典类型
	CssClass  string `json:"css_class" form:"css_class" gorm:"column:css_class;type:varchar(100);comment:样式属性（其他样式扩展）"`               // 样式属性（其他样式扩展）
	ListClass string `json:"list_class" form:"list_class" gorm:"column:list_class;type:varchar(100);comment:表格回显样式"`                 // 表格回显样式
	IsDefault string `json:"is_default" form:"is_default" gorm:"column:is_default;type:char(1);default:'N';comment:是否默认（Y是 N否）"`      // 是否默认（Y是 N否）
	Status    uint   `json:"status" form:"status" gorm:"column:status;type:char(1);default:0;comment:状态（0正常 1停用）"`                  // 状态（0正常 1停用）

	core.RuoyiModel
}

// DictDataSearchRequest 字典数据搜索请求结构体
type DictDataSearchRequest struct {
	request.PageInfo
	request.SortInfo

	Keyword string `json:"keyword" form:"keyword"` // 关键字
	Label   string `json:"label" form:"label"`     // 字典标签
	Value   string `json:"value" form:"value"`     // 字典键值
	Type    string `json:"type" form:"type"`       // 字典类型
	ID      *uint  `json:"id" form:"id"`           // 字典编码
	Status  *uint  `json:"status" form:"status"`   // 状态（0正常 1停用）

	IDNotIn []uint `json:"idNotIn"` // 不包含的ID列表
	IDIn    []uint `json:"idIn"`    // 包含的ID列表

	core.RouyiSearchRequest
}

// DictDataCreateRequest 字典数据添加请求结构体
type DictDataCreateRequest struct {
	DictDataBase
}

// DictDataUpdateRequest 字典数据编辑请求结构体
type DictDataUpdateRequest struct {
	request.GetById
	DictDataBase
}
