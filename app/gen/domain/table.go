package domain

import (
	"time"

	"github.com/bluefroog/admintpl/common/model/core"
	"github.com/bluefroog/admintpl/common/model/request"
)

// Table 代码生成业务表 model（gen_table）
type Table struct {
	ID uint `json:"id" form:"id" gorm:"column:table_id;type:bigint(20) auto_increment;primary_key;not null;comment:编号"` // 编号

	TableBase

	// 以下字段不落库：由 options 解析或查询列信息后回填
	TreeCode       string        `json:"tree_code" form:"tree_code" gorm:"-"`               // 树编码字段
	TreeParentCode string        `json:"tree_parent_code" form:"tree_parent_code" gorm:"-"` // 树父编码字段
	TreeName       string        `json:"tree_name" form:"tree_name" gorm:"-"`               // 树名称字段
	Columns        []TableColumn `json:"columns" gorm:"-"`                                  // 表列信息
	PkColumn       *TableColumn  `json:"pk_column" gorm:"-"`                                // 主键列信息
}

// TableName 指定表名
func (m *Table) TableName() string {
	return "gen_table"
}

// TableBase 代码生成业务表字段
type TableBase struct {
	Name           string `json:"table_name" form:"table_name" validate:"required" gorm:"column:table_name;type:varchar(200);not null;comment:表名称"` // 表名称（表名与 GORM 的 TableName() 方法同名会冲突，故字段名用 Name）
	TableComment   string `json:"table_comment" form:"table_comment" gorm:"column:table_comment;type:varchar(500);comment:表描述"`                     // 表描述
	ClassName      string `json:"class_name" form:"class_name" gorm:"column:class_name;type:varchar(100);comment:实体类名称"`                            // 实体类名称
	TplCategory    string `json:"tpl_category" form:"tpl_category" gorm:"column:tpl_category;type:varchar(20);comment:使用的模板（crud单表操作 tree树表操作）"`    // 使用的模板（crud单表操作 tree树表操作）
	PackageName    string `json:"package_name" form:"package_name" gorm:"column:package_name;type:varchar(100);comment:生成包路径"`                      // 生成包路径
	ModuleName     string `json:"module_name" form:"module_name" gorm:"column:module_name;type:varchar(30);comment:生成模块名"`                          // 生成模块名
	BusinessName   string `json:"business_name" form:"business_name" gorm:"column:business_name;type:varchar(30);comment:生成业务名"`                    // 生成业务名
	FunctionName   string `json:"function_name" form:"function_name" gorm:"column:function_name;type:varchar(50);comment:生成功能名"`                    // 生成功能名
	FunctionAuthor string `json:"function_author" form:"function_author" gorm:"column:function_author;type:varchar(50);comment:生成功能作者"`             // 生成功能作者
	GenType        string `json:"gen_type" form:"gen_type" gorm:"column:gen_type;type:char(1);default:'0';comment:生成代码方式（0zip压缩包 1自定义路径）"`          // 生成代码方式（0zip压缩包 1自定义路径）
	GenPath        string `json:"gen_path" form:"gen_path" gorm:"column:gen_path;type:varchar(200);comment:生成路径（不填默认项目路径）"`                         // 生成路径（不填默认项目路径）
	Options        string `json:"options" form:"options" gorm:"column:options;type:varchar(1000);comment:其它生成选项"`                                   // 其它生成选项（树表模板参数等 JSON）

	core.RuoyiModel
}

// DbTable 数据库表信息（information_schema.tables 查询结果，非实体表）
type DbTable struct {
	TableName    string     `json:"table_name" gorm:"column:table_name"`       // 表名称
	TableComment string     `json:"table_comment" gorm:"column:table_comment"` // 表描述
	CreateTime   *time.Time `json:"create_time" gorm:"column:create_time"`     // 创建时间
	UpdateTime   *time.Time `json:"update_time" gorm:"column:update_time"`     // 更新时间
}

// DbColumn 数据库列信息（information_schema.columns 查询结果，非实体表）
type DbColumn struct {
	ColumnName    string `json:"column_name" gorm:"column:column_name"`       // 列名称
	ColumnComment string `json:"column_comment" gorm:"column:column_comment"` // 列描述
	ColumnType    string `json:"column_type" gorm:"column:column_type"`       // 列类型
	IsRequired    string `json:"is_required" gorm:"column:is_required"`       // 是否必填（1是）
	IsPk          string `json:"is_pk" gorm:"column:is_pk"`                   // 是否主键（1是）
	IsIncrement   string `json:"is_increment" gorm:"column:is_increment"`     // 是否自增（1是）
	Sort          uint   `json:"sort" gorm:"column:sort"`                     // 排序
}

// TableSearchRequest 代码生成业务表搜索请求结构体（已导入的表）
type TableSearchRequest struct {
	request.PageInfo
	request.SortInfo

	ID           *uint  `json:"id" form:"id"`                       // 编号
	TableName    string `json:"table_name" form:"table_name"`       // 表名称
	TableComment string `json:"table_comment" form:"table_comment"` // 表描述
	BeginTime    string `json:"begin_time" form:"begin_time"`       // 开始时间
	EndTime      string `json:"end_time" form:"end_time"`           // 结束时间

	core.RouyiSearchRequest
}

// DbTableSearchRequest 数据库表搜索请求结构体（未导入的表）
type DbTableSearchRequest struct {
	request.PageInfo
	request.SortInfo

	TableName    string `json:"table_name" form:"table_name"`       // 表名称
	TableComment string `json:"table_comment" form:"table_comment"` // 表描述
	BeginTime    string `json:"begin_time" form:"begin_time"`       // 开始时间
	EndTime      string `json:"end_time" form:"end_time"`           // 结束时间
}

// TableImportRequest 导入表结构请求结构体
// 支持 JSON 数组（{"tables":["a","b"]}）与逗号分隔（tables=a,b）
type TableImportRequest struct {
	Tables []string `json:"tables" form:"tables" validate:"required"` // 待导入的表名列表
}

// TableUpdateRequest 修改代码生成配置请求结构体
type TableUpdateRequest struct {
	request.GetById

	TableName      string `json:"table_name" form:"table_name" validate:"required"` // 表名称
	TableComment   string `json:"table_comment" form:"table_comment"`               // 表描述
	ClassName      string `json:"class_name" form:"class_name" validate:"required"` // 实体类名称
	TplCategory    string `json:"tpl_category" form:"tpl_category"`                 // 使用的模板（crud单表操作 tree树表操作）
	PackageName    string `json:"package_name" form:"package_name"`                 // 生成包路径
	ModuleName     string `json:"module_name" form:"module_name"`                   // 生成模块名
	BusinessName   string `json:"business_name" form:"business_name"`               // 生成业务名
	FunctionName   string `json:"function_name" form:"function_name"`               // 生成功能名
	FunctionAuthor string `json:"function_author" form:"function_author"`           // 生成功能作者
	GenType        string `json:"gen_type" form:"gen_type"`                         // 生成代码方式（0zip压缩包 1自定义路径）
	GenPath        string `json:"gen_path" form:"gen_path"`                         // 生成路径
	Options        string `json:"options" form:"options"`                           // 其它生成选项（JSON 字符串）
	Remark         string `json:"remark" form:"remark"`                             // 备注

	// 树表模板参数（同时传入时会覆盖 options 中的同名项）
	TreeCode       string `json:"tree_code" form:"tree_code"`               // 树编码字段
	TreeParentCode string `json:"tree_parent_code" form:"tree_parent_code"` // 树父编码字段
	TreeName       string `json:"tree_name" form:"tree_name"`               // 树名称字段

	Columns []TableColumnUpdateRequest `json:"columns" form:"columns"` // 列信息
}

// TableGenRequest 代码生成/预览请求结构体
type TableGenRequest struct {
	request.GetById
}

// TableSynchRequest 同步数据库表结构请求结构体
type TableSynchRequest struct {
	request.GetById
}
