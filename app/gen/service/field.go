package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/bluefroog/admintpl/app/gen/domain"
	"github.com/bluefroog/admintpl/common/config"
)

// 数据库类型分组（与若依保持一致）
var (
	// columnTypeStr 数据库字符串类型
	columnTypeStr = []string{"char", "varchar", "narchar", "varchar2", "tinytext", "text", "mediumtext", "longtext"}
	// columnTypeTime 数据库时间类型
	columnTypeTime = []string{"datetime", "time", "date", "timestamp"}
	// columnTypeNumber 数据库数字类型
	columnTypeNumber = []string{"tinyint", "smallint", "mediumint", "int", "number", "integer", "bigint", "float", "double", "decimal"}
	// columnNameNotEdit 页面不需要编辑的字段
	columnNameNotEdit = []string{"id", "create_by", "create_time", "del_flag", "update_by", "update_time"}
	// columnNameNotList 页面不需要显示的列表字段
	columnNameNotList = []string{"id", "create_by", "create_time", "del_flag", "update_by", "update_time"}
	// columnNameNotQuery 页面不需要查询的字段
	columnNameNotQuery = []string{"id", "create_by", "create_time", "del_flag", "update_by", "update_time", "remark"}
)

// options 树表模板参数（gen_table.options 的内容）
type options struct {
	TreeCode       string `json:"treeCode"`
	TreeParentCode string `json:"treeParentCode"`
	TreeName       string `json:"treeName"`
}

// IsExistInArray 判断字符串是否存在于数组中
// @param: value string
// @param: array []string
// @return: bool
func IsExistInArray(value string, array []string) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}
	return false
}

// IsStringObject 判断是否是数据库字符串类型
// @param: value string 数据库类型
// @return: bool
func IsStringObject(value string) bool {
	return IsExistInArray(strings.ToLower(value), columnTypeStr)
}

// IsTimeObject 判断是否是数据库时间类型
// @param: value string 数据库类型
// @return: bool
func IsTimeObject(value string) bool {
	return IsExistInArray(strings.ToLower(value), columnTypeTime)
}

// IsNumberObject 判断是否是数据库数字类型
// @param: value string 数据库类型
// @return: bool
func IsNumberObject(value string) bool {
	return IsExistInArray(strings.ToLower(value), columnTypeNumber)
}

// IsNotEdit 页面不需要编辑的字段
// @param: value string 列名称
// @return: bool
func IsNotEdit(value string) bool {
	return !IsExistInArray(value, columnNameNotEdit)
}

// IsNotList 页面不需要显示的列表字段
// @param: value string 列名称
// @return: bool
func IsNotList(value string) bool {
	return !IsExistInArray(value, columnNameNotList)
}

// IsNotQuery 页面不需要查询的字段
// @param: value string 列名称
// @return: bool
func IsNotQuery(value string) bool {
	return !IsExistInArray(value, columnNameNotQuery)
}

// DbType 获取数据库类型字段（去掉长度部分，如 varchar(64) -> varchar）
// @param: columnType string 列类型
// @return: string
func DbType(columnType string) string {
	if index := strings.Index(columnType, "("); index > 0 {
		return columnType[0:index]
	}
	return columnType
}

// ColumnLength 获取字段长度（varchar(64) -> 64），无长度返回 0
// @param: columnType string 列类型
// @return: int
func ColumnLength(columnType string) int {
	start := strings.Index(columnType, "(")
	end := strings.Index(columnType, ")")
	if start < 0 || end <= start+1 {
		return 0
	}
	length := 0
	for _, char := range columnType[start+1 : end] {
		if char < '0' || char > '9' {
			return length
		}
		length = length*10 + int(char-'0')
	}
	return length
}

// CamelCase 下划线命名转大驼峰（user_name -> UserName），不含下划线时仅首字母大写
// @param: name string 原始名称
// @return: string
func CamelCase(name string) string {
	if name == "" {
		return ""
	}
	if !strings.Contains(name, "_") {
		return strings.ToUpper(name[0:1]) + name[1:]
	}
	result := ""
	for _, item := range strings.Split(name, "_") {
		if item == "" {
			continue
		}
		result = result + strings.ToUpper(item[0:1]) + strings.ToLower(item[1:])
	}
	return result
}

// CamelCaseLower 下划线命名转小驼峰（user_name -> userName），不含下划线时原样返回
// @param: name string 原始名称
// @return: string
func CamelCaseLower(name string) string {
	if name == "" {
		return ""
	}
	if !strings.Contains(name, "_") {
		return name
	}
	result := ""
	for _, item := range strings.Split(name, "_") {
		if item == "" {
			continue
		}
		if result == "" {
			result = strings.ToLower(item[0:1]) + strings.ToLower(item[1:])
		} else {
			result = result + strings.ToUpper(item[0:1]) + strings.ToLower(item[1:])
		}
	}
	return result
}

// ClassName 表名转换成类名（按配置去除表前缀后转大驼峰）
// @param: tableName string 表名称
// @return: string
func ClassName(tableName string) string {
	return classNameBy(tableName, config.Bool("system.gen.auto-remove-pre"), config.String("system.gen.table-prefix"))
}

// classNameBy 表名转类名的纯函数实现（便于单元测试）
// @param: tableName string 表名称
// @param: autoRemovePre bool 是否自动去除表前缀
// @param: tablePrefix string 表前缀，多个以英文逗号分隔
// @return: string
func classNameBy(tableName string, autoRemovePre bool, tablePrefix string) string {
	if autoRemovePre && tablePrefix != "" {
		for _, prefix := range strings.Split(tablePrefix, ",") {
			prefix = strings.TrimSpace(prefix)
			if prefix != "" {
				tableName = strings.ReplaceAll(tableName, prefix, "")
			}
		}
	}
	return CamelCase(tableName)
}

// BusinessName 获取业务名（表名最后一个下划线之后的部分，如 sys_user_role -> role）
// @param: tableName string 表名称
// @return: string
func BusinessName(tableName string) string {
	if index := strings.LastIndex(tableName, "_"); index >= 0 && index+1 < len(tableName) {
		return tableName[index+1:]
	}
	return tableName
}

// FunctionName 获取功能名（表描述去掉「表」字）
// @param: tableComment string 表描述
// @return: string
func FunctionName(tableComment string) string {
	return strings.ReplaceAll(tableComment, "表", "")
}

// CheckSexColumn 检查字段名后 3 位是否是 sex
// @param: columnName string 列名称
// @return: bool
func CheckSexColumn(columnName string) bool {
	return len(columnName) >= 3 && columnName[len(columnName)-3:] == "sex"
}

// CheckTypeColumn 检查字段名后 4 位是否是 type
// @param: columnName string 列名称
// @return: bool
func CheckTypeColumn(columnName string) bool {
	return len(columnName) >= 4 && columnName[len(columnName)-4:] == "type"
}

// CheckNameColumn 检查字段名后 4 位是否是 name
// @param: columnName string 列名称
// @return: bool
func CheckNameColumn(columnName string) bool {
	return len(columnName) >= 4 && columnName[len(columnName)-4:] == "name"
}

// CheckStatusColumn 检查字段名后 6 位是否是 status
// @param: columnName string 列名称
// @return: bool
func CheckStatusColumn(columnName string) bool {
	return len(columnName) >= 6 && columnName[len(columnName)-6:] == "status"
}

// InitTable 初始化表信息（类名/包名/模块名/业务名/功能名/作者等）
// @param: table *domain.Table 表信息
// @param: operName string 操作人
func InitTable(table *domain.Table, operName string) {
	table.ClassName = ClassName(table.Name)
	table.PackageName = config.String("system.gen.package-name")
	table.ModuleName = config.String("system.gen.module-name")
	table.BusinessName = BusinessName(table.Name)
	table.FunctionName = FunctionName(table.TableComment)
	table.FunctionAuthor = config.String("system.gen.author")
	table.CreateBy = operName
	table.UpdateBy = operName
	table.TplCategory = "crud"
	table.GenType = "0"
	table.CreateTime = time.Now()
	table.UpdateTime = time.Now()
}

// InitColumnField 初始化列属性字段（GO 类型 / 字段名 / 编辑与查询配置 / 显示类型）
// @param: column *domain.TableColumn 列信息
// @param: table *domain.Table 表信息
func InitColumnField(column *domain.TableColumn, table *domain.Table) {
	dataType := DbType(column.ColumnType)
	columnName := column.ColumnName
	column.TableID = table.ID
	column.CreateBy = table.CreateBy
	column.UpdateBy = table.UpdateBy
	column.CreateTime = time.Now()
	column.UpdateTime = time.Now()
	// 设置字段名
	column.GoField = CamelCase(columnName)
	column.HtmlField = CamelCaseLower(columnName)

	switch {
	case IsStringObject(dataType):
		// 字段为字符串类型
		column.GoType = "string"
		if IsExistInArray(strings.ToLower(dataType), []string{"text", "tinytext", "mediumtext", "longtext"}) || ColumnLength(column.ColumnType) >= 500 {
			column.HtmlType = "textarea"
		} else {
			column.HtmlType = "input"
		}
	case IsTimeObject(dataType):
		// 字段为时间类型
		column.GoType = "Time"
		column.HtmlType = "datetime"
	case IsNumberObject(dataType):
		// 字段为数字类型
		column.HtmlType = "input"
		column.GoType = numberGoType(column.ColumnType)
	default:
		// 其它类型（二进制、枚举等）按字符串处理
		column.GoType = "string"
		column.HtmlType = "input"
	}

	// 新增字段：时间与操作人字段不参与新增表单（由框架统一维护）
	if columnName == "create_by" || columnName == "create_time" || columnName == "update_by" || columnName == "update_time" {
		column.IsRequired = "0"
		column.IsInsert = "0"
	} else {
		column.IsInsert = "1"
		if strings.Contains(columnName, "name") || strings.Contains(columnName, "status") {
			column.IsRequired = "1"
		}
	}

	// 编辑 / 列表 / 查询字段（主键列不参与编辑、列表与查询，与若依一致）
	isPk := column.IsPk == "1"
	if !isPk && IsNotEdit(columnName) {
		column.IsEdit = "1"
	} else {
		column.IsEdit = "0"
	}
	if !isPk && IsNotList(columnName) {
		column.IsList = "1"
	} else {
		column.IsList = "0"
	}
	if !isPk && IsNotQuery(columnName) {
		column.IsQuery = "1"
	} else {
		column.IsQuery = "0"
	}

	// 查询字段类型
	if CheckNameColumn(columnName) {
		column.QueryType = "LIKE"
	} else {
		column.QueryType = "EQ"
	}

	// 状态字段设置单选框；类型与性别字段设置下拉框
	if CheckStatusColumn(columnName) {
		column.HtmlType = "radio"
	} else if CheckTypeColumn(columnName) || CheckSexColumn(columnName) {
		column.HtmlType = "select"
	}
}

// numberGoType 根据数据库数字类型推导 GO 类型（float/double 与带小数位的 decimal 为 float64）
// 与若依一致：整型长度 <= 10 为 int，否则为 int64；decimal(m,0) 也按整型处理。
// @param: columnType string 列类型
// @return: string
func numberGoType(columnType string) string {
	dataType := strings.ToLower(DbType(columnType))
	if dataType == "float" || dataType == "double" {
		return "float64"
	}
	start := strings.Index(columnType, "(")
	end := strings.Index(columnType, ")")
	if start < 0 || end <= start+1 {
		return "int64"
	}
	parts := strings.Split(columnType[start+1:end], ",")
	if len(parts) == 2 {
		// decimal(10,2) 之类带小数位的列按浮点处理（decimal(10,0) 仍是整型）
		if scale, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && scale > 0 {
			return "float64"
		}
		return "int64"
	}
	if length, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && length > 0 && length <= 10 {
		return "int"
	}
	return "int64"
}

// SetPkColumn 设置主键列信息（找不到主键时使用第一列）
// @param: table *domain.Table 表信息
func SetPkColumn(table *domain.Table) {
	table.PkColumn = nil
	for index := range table.Columns {
		if table.Columns[index].IsPk == "1" {
			table.PkColumn = &table.Columns[index]
			return
		}
	}
	if len(table.Columns) > 0 {
		table.PkColumn = &table.Columns[0]
	}
}

// ParseOptions 解析表扩展选项（树表模板参数）
// @param: table *domain.Table 表信息
func ParseOptions(table *domain.Table) {
	if table == nil || table.Options == "" {
		return
	}
	var opt options
	if err := json.Unmarshal([]byte(table.Options), &opt); err != nil {
		return
	}
	table.TreeCode = opt.TreeCode
	table.TreeParentCode = opt.TreeParentCode
	table.TreeName = opt.TreeName
}

// BuildOptions 组装表扩展选项（树表模板参数）
// 显式传入 options 时以请求为准；未传树表参数时清空，避免切换模板后残留旧配置
// @param: req *domain.TableUpdateRequest 修改请求
// @return: string
func BuildOptions(req *domain.TableUpdateRequest) string {
	if req.Options != "" {
		return req.Options
	}
	if req.TreeCode == "" && req.TreeParentCode == "" && req.TreeName == "" {
		return ""
	}
	content, err := json.Marshal(options{
		TreeCode:       req.TreeCode,
		TreeParentCode: req.TreeParentCode,
		TreeName:       req.TreeName,
	})
	if err != nil {
		return req.Options
	}
	return string(content)
}
