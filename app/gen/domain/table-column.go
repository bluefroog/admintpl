package domain

import (
	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// TableColumn 代码生成业务表字段 model（gen_table_column）
type TableColumn struct {
	ID uint `json:"id" form:"id" gorm:"column:column_id;type:bigint(20) auto_increment;primary_key;not null;comment:编号"` // 编号

	TableColumnBase
}

// TableName 指定表名
func (m *TableColumn) TableName() string {
	return "gen_table_column"
}

// TableColumnBase 代码生成业务表字段字段
type TableColumnBase struct {
	TableID       uint   `json:"table_id" form:"table_id" gorm:"column:table_id;type:bigint(20);not null;comment:归属表编号"`          // 归属表编号
	ColumnName    string `json:"column_name" form:"column_name" gorm:"column:column_name;type:varchar(200);comment:列名称"`          // 列名称
	ColumnComment string `json:"column_comment" form:"column_comment" gorm:"column:column_comment;type:varchar(500);comment:列描述"` // 列描述
	ColumnType    string `json:"column_type" form:"column_type" gorm:"column:column_type;type:varchar(100);comment:列类型"`          // 列类型

	GoType    string `json:"go_type" form:"go_type" gorm:"column:go_type;type:varchar(100);comment:GO类型"`             // GO 类型
	GoField   string `json:"go_field" form:"go_field" gorm:"column:go_field;type:varchar(200);comment:GO字段名"`         // GO 字段名
	HtmlField string `json:"html_field" form:"html_field" gorm:"column:html_field;type:varchar(200);comment:html字段名"` // html 字段名

	IsPk        string `json:"is_pk" form:"is_pk" gorm:"column:is_pk;type:char(1);comment:是否主键（1是）"`                      // 是否主键（1是）
	IsIncrement string `json:"is_increment" form:"is_increment" gorm:"column:is_increment;type:char(1);comment:是否自增（1是）"` // 是否自增（1是）
	IsRequired  string `json:"is_required" form:"is_required" gorm:"column:is_required;type:char(1);comment:是否必填（1是）"`    // 是否必填（1是）
	IsInsert    string `json:"is_insert" form:"is_insert" gorm:"column:is_insert;type:char(1);comment:是否为插入字段（1是）"`       // 是否为插入字段（1是）
	IsEdit      string `json:"is_edit" form:"is_edit" gorm:"column:is_edit;type:char(1);comment:是否编辑字段（1是）"`              // 是否编辑字段（1是）
	IsList      string `json:"is_list" form:"is_list" gorm:"column:is_list;type:char(1);comment:是否列表字段（1是）"`              // 是否列表字段（1是）
	IsQuery     string `json:"is_query" form:"is_query" gorm:"column:is_query;type:char(1);comment:是否查询字段（1是）"`           // 是否查询字段（1是）

	QueryType string `json:"query_type" form:"query_type" gorm:"column:query_type;type:varchar(20);comment:查询方式（等于、不等于、大于、小于、范围）"`       // 查询方式
	HtmlType  string `json:"html_type" form:"html_type" gorm:"column:html_type;type:varchar(20);comment:显示类型（文本框、文本域、下拉框、复选框、单选框、日期控件）"` // 显示类型
	DictType  string `json:"dict_type" form:"dict_type" gorm:"column:dict_type;type:varchar(200);comment:字典类型"`                          // 字典类型
	Sort      uint   `json:"sort" form:"sort" gorm:"column:sort;type:int(4);comment:排序"`                                                 // 排序

	core.RuoyiModel
}

// TableColumnSearchRequest 代码生成业务表字段搜索请求结构体
type TableColumnSearchRequest struct {
	request.PageInfo
	request.SortInfo

	ID      *uint  `json:"id" form:"id"`                   // 编号
	TableID *uint  `json:"table_id" form:"table_id"`       // 归属表编号
	Column  string `json:"column_name" form:"column_name"` // 列名称
}

// TableColumnUpdateRequest 修改代码生成业务表字段请求结构体
type TableColumnUpdateRequest struct {
	ID uint `json:"id" form:"id" validate:"required"` // 编号

	ColumnComment string `json:"column_comment" form:"column_comment"` // 列描述
	GoType        string `json:"go_type" form:"go_type"`               // GO 类型
	GoField       string `json:"go_field" form:"go_field"`             // GO 字段名
	HtmlField     string `json:"html_field" form:"html_field"`         // html 字段名
	DictType      string `json:"dict_type" form:"dict_type"`           // 字典类型
	QueryType     string `json:"query_type" form:"query_type"`         // 查询方式
	HtmlType      string `json:"html_type" form:"html_type"`           // 显示类型

	IsRequired string `json:"is_required" form:"is_required"` // 是否必填（1是）
	IsInsert   string `json:"is_insert" form:"is_insert"`     // 是否为插入字段（1是）
	IsEdit     string `json:"is_edit" form:"is_edit"`         // 是否编辑字段（1是）
	IsList     string `json:"is_list" form:"is_list"`         // 是否列表字段（1是）
	IsQuery    string `json:"is_query" form:"is_query"`       // 是否查询字段（1是）
	Sort       uint   `json:"sort" form:"sort"`               // 排序
}
