package service

import (
	"archive/zip"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"go/format"
	"path"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/bluefroog/admintpl/app/gen/domain"
	"github.com/bluefroog/admintpl/common/config"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// 代码生成模板
const (
	tplDomain     = "domain.tmpl"     // 实体 + 请求结构体
	tplService    = "service.tmpl"    // 业务逻辑
	tplController = "controller.tmpl" // HTTP 控制器
	tplHTTP       = "http.tmpl"       // 接口测试文件
	tplSQL        = "sql.tmpl"        // 建表语句与菜单语句
	tplRegister   = "register.tmpl"   // 接入说明（模型注册 / 路由注册 / 权限标识）
)

// 生成代码的默认配置（配置缺失时兜底，避免生成产物缺少包路径）
const (
	defaultPackageName = "github.com/bluefroog/admintpl"
	defaultModuleName  = "system"
	defaultAuthor      = "bluefrog"
)

// 生成文件的相对路径（相对项目根目录）
var (
	templateFiles = map[string]string{
		tplDomain:     "%s/app/%s/domain/%s.go",
		tplService:    "%s/app/%s/service/%s.go",
		tplController: "%s/app/%s/controller/%s.go",
		tplHTTP:       "%s/app/%s/test/http/%s.http",
		tplSQL:        "docs/sql/%s.sql",
		tplRegister:   "%s/app/%s/%s.register.md",
	}
	moduleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	classNamePattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	fileNamePattern   = regexp.MustCompile(`[^a-z0-9_]+`)
)

// Generate 代码生成（模板渲染）服务
type Generate struct{}

// renderColumn 渲染用的列信息（在 gen_table_column 基础上补充 Go 类型与查询语义）
type renderColumn struct {
	domain.TableColumn
	Comment     string // 处理后的列注释（去掉引号与分号，可直接写入 gorm tag）
	GoTypeDecl  string // Go 字段类型
	SearchType  string // 搜索请求字段类型（基础类型加指针，便于判断是否传值）
	QueryOp     string // 查询运算符
	IsRange     bool   // 是否按区间查询（时间列或 BETWEEN）
	IsTextField bool   // 是否大文本（用于说明）
}

// renderData 模板渲染上下文
type renderData struct {
	Table       *domain.Table
	Pk          *renderColumn
	AllField    []renderColumn // 全部列（保持导入顺序，供建表语句使用）
	Fields      []renderColumn // 实体字段（不含主键与公共审计列）
	QueryField  []renderColumn // 搜索条件字段
	EditField   []renderColumn // 编辑字段
	InsertField []renderColumn // 新增字段
	ListField   []renderColumn // 列表字段
	KeyField    []renderColumn // 参与关键字模糊查询的字符串列

	UseAuditModel bool // 是否嵌入 core.AuditModel（表含 create_by / update_by）
	ImportTime    bool // 实体是否需要 import "time"（存在 time.Time 字段）
	ServiceTime   bool // service 是否需要 import "time"（审计列赋值）
	ImportStrings bool // service 是否需要 import "strings"（关键字模糊查询）

	PackageName    string
	ModuleName     string
	RoutePrefix    string // 路由前缀，如 /system
	PermissionPre  string // 权限标识前缀，如 system:post
	ClassName      string // 实体类名，如 Post
	ClassNameLower string // 控制器变量名，如 post
	FileName       string // 文件名（蛇形，去表前缀），如 post
	FunctionName   string // 功能名（表描述去「表」）
	Author         string
	DateTime       string
	TableName      string
	TableComment   string

	IsTree         bool
	TreeCode       string
	TreeParentCode string
	TreeName       string
}

// Preview 预览生成结果（相对路径 -> 文件内容）
// @param: tableID uint 代码生成表编号
// @return: map[string]string, error
func (g Generate) Preview(tableID uint) (map[string]string, error) {
	data, err := g.prepare(tableID)
	if err != nil {
		return nil, err
	}
	return renderAll(data)
}

// Download 打包生成结果（返回下载文件名与 zip 字节流）
// @param: tableID uint 代码生成表编号
// @return: string, []byte, error
func (g Generate) Download(tableID uint) (string, []byte, error) {
	data, err := g.prepare(tableID)
	if err != nil {
		return "", nil, err
	}
	files, err := renderAll(data)
	if err != nil {
		return "", nil, err
	}
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	// 固定顺序，保证同样的配置生成同样的 zip
	for _, name := range renderOrder(data) {
		content, ok := files[name]
		if !ok {
			continue
		}
		file, err := writer.Create(name)
		if err != nil {
			return "", nil, err
		}
		if _, err := file.Write([]byte(content)); err != nil {
			return "", nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return "", nil, err
	}
	return data.FileName + ".zip", buf.Bytes(), nil
}

// renderOrder 生成文件顺序（按层级排列，便于阅读）
func renderOrder(data *renderData) []string {
	return []string{
		renderPath(data, tplDomain),
		renderPath(data, tplService),
		renderPath(data, tplController),
		renderPath(data, tplHTTP),
		renderPath(data, tplSQL),
		renderPath(data, tplRegister),
	}
}

// renderPath 计算模板对应的生成文件路径
// @param: data *renderData 渲染上下文
// @param: tpl string 模板文件名
// @return: string
func renderPath(data *renderData, tpl string) string {
	format, ok := templateFiles[tpl]
	if !ok {
		return tpl
	}
	if tpl == tplSQL {
		return fmt.Sprintf(format, data.FileName)
	}
	return fmt.Sprintf(format, data.PackageName, data.ModuleName, data.FileName)
}

// renderAll 渲染全部模板（Go 代码经 gofmt 格式化，同时验证语法）
// @param: data *renderData 渲染上下文
// @return: map[string]string, error
func renderAll(data *renderData) (map[string]string, error) {
	result := make(map[string]string, len(templateFiles))
	for _, tpl := range []string{tplDomain, tplService, tplController, tplHTTP, tplSQL, tplRegister} {
		content, err := renderOne(tpl, data)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(tpl, "domain.tmpl") || strings.HasSuffix(tpl, "service.tmpl") || strings.HasSuffix(tpl, "controller.tmpl") {
			// Go 源码统一格式化（语法错误会在此暴露，避免生成不可编译的代码）
			formatted, err := format.Source([]byte(content))
			if err != nil {
				return nil, fmt.Errorf("生成 %s 失败：%v", renderPath(data, tpl), err)
			}
			content = string(formatted)
		}
		result[renderPath(data, tpl)] = content
	}
	return result, nil
}

// renderOne 渲染单个模板
// @param: tpl string 模板文件名
// @param: data *renderData 渲染上下文
// @return: string, error
func renderOne(tpl string, data *renderData) (string, error) {
	raw, err := templateFS.ReadFile(path.Join("templates", tpl))
	if err != nil {
		return "", err
	}
	tplIns, err := template.New(tpl).Parse(string(raw))
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err := tplIns.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// prepare 读取表结构并组装渲染上下文
// @param: tableID uint 代码生成表编号
// @return: *renderData, error
func (g Generate) prepare(tableID uint) (*renderData, error) {
	table, err := Table{}.Detail(tableID)
	if err != nil {
		return nil, err
	}
	return buildRenderData(table)
}

// buildRenderData 校验表配置并组装渲染上下文（与数据库解耦，便于单元测试）
// @param: table *domain.Table 表结构（含 gen_table_column 列信息）
// @return: *renderData, error
func buildRenderData(table *domain.Table) (*renderData, error) {
	if table == nil {
		return nil, errors.New("表信息为空")
	}
	if len(table.Columns) == 0 {
		return nil, errors.New("表字段信息为空，请先导入表结构")
	}
	if table.PkColumn == nil || table.PkColumn.ColumnName == "" {
		return nil, errors.New("未找到主键列，无法生成代码")
	}
	if !IsNumberObject(DbType(table.PkColumn.ColumnType)) {
		return nil, errors.New("主键列必须为整型（bigint / int）才能生成 CRUD 代码")
	}
	// 树表参数存在 options JSON 中，渲染前先解析（数据库读取的原始值不会自动展开）
	ParseOptions(table)

	// 生成配置：优先取 gen_table 中按表配置的值（页面可编辑），缺失时回落全局配置与推导值
	packageName := firstNonEmpty(table.PackageName, config.String("system.gen.package-name"), defaultPackageName)
	moduleName := firstNonEmpty(table.ModuleName, config.String("system.gen.module-name"), defaultModuleName)
	if !moduleNamePattern.MatchString(moduleName) {
		return nil, fmt.Errorf("模块名 %q 不合法（只允许小写字母、数字与下划线，且以字母开头）", moduleName)
	}
	author := firstNonEmpty(table.FunctionAuthor, config.String("system.gen.author"), defaultAuthor)
	autoRemovePre := !config.IsSet("system.gen.auto-remove-pre") || config.Bool("system.gen.auto-remove-pre")
	tablePrefix := config.String("system.gen.table-prefix")

	// 文件名沿用项目约定：表名去前缀后的蛇形名（不取 business_name，避免多段表名丢失信息）
	trimmed := TrimTablePrefix(table.Name, autoRemovePre, tablePrefix)
	fileName := strings.Trim(fileNamePattern.ReplaceAllString(strings.ToLower(trimmed), "_"), "_")
	if fileName == "" {
		return nil, fmt.Errorf("表名 %q 无法转换为合法文件名", table.Name)
	}
	className := firstNonEmpty(table.ClassName, classNameBy(table.Name, autoRemovePre, tablePrefix))
	if !classNamePattern.MatchString(className) {
		return nil, fmt.Errorf("实体类名 %q 不合法（只允许字母开头，含字母与数字）", className)
	}
	functionName := firstNonEmpty(table.FunctionName, FunctionName(table.TableComment), className)

	data := &renderData{
		Table:          table,
		Pk:             newRenderColumnPtr(table.PkColumn),
		PackageName:    packageName,
		ModuleName:     moduleName,
		RoutePrefix:    "/" + moduleName,
		PermissionPre:  strings.ReplaceAll(strings.TrimPrefix("/"+moduleName+"/"+fileName, "/"), "/", ":"),
		ClassName:      className,
		ClassNameLower: CamelCaseLower(fileName),
		FileName:       fileName,
		FunctionName:   functionName,
		Author:         author,
		DateTime:       time.Now().Format("2006-01-02 15:04:05"),
		TableName:      table.Name,
		TableComment:   table.TableComment,
		IsTree:         table.TplCategory == "tree",
		TreeCode:       table.TreeCode,
		TreeParentCode: table.TreeParentCode,
		TreeName:       table.TreeName,
	}
	// 公共审计列由 core.AuditModel 提供（create_by/update_by 同时存在时启用）
	data.UseAuditModel = columnExist(table.Columns, "create_by") && columnExist(table.Columns, "update_by")
	autoKeys := map[string]bool{}
	if data.UseAuditModel {
		for _, key := range []string{"create_by", "create_time", "update_by", "update_time", "remark"} {
			autoKeys[key] = true
		}
	}
	autoKeys[table.PkColumn.ColumnName] = true

	for i := range table.Columns {
		column := newRenderColumn(&table.Columns[i])
		data.AllField = append(data.AllField, column)
		if autoKeys[column.ColumnName] {
			continue
		}
		data.Fields = append(data.Fields, column)
		if column.IsQuery == "1" {
			data.QueryField = append(data.QueryField, column)
			if column.GoTypeDecl == "string" && !column.IsRange {
				data.KeyField = append(data.KeyField, column)
			}
		}
		if column.IsEdit == "1" {
			data.EditField = append(data.EditField, column)
		}
		if column.IsInsert == "1" {
			data.InsertField = append(data.InsertField, column)
		}
		if column.IsList == "1" {
			data.ListField = append(data.ListField, column)
		}
		if column.GoTypeDecl == "time.Time" {
			data.ImportTime = true
		}
	}
	data.ServiceTime = data.ImportTime || data.UseAuditModel
	data.ImportStrings = len(data.KeyField) > 0
	return data, nil
}

// newRenderColumn 组装渲染列信息
// @param: column *domain.TableColumn 数据库列配置
// @return: renderColumn
func newRenderColumn(column *domain.TableColumn) renderColumn {
	item := renderColumn{
		TableColumn: *column,
		Comment:     cleanComment(column.ColumnComment),
		GoTypeDecl:  goTypeDecl(column.ColumnType),
		QueryOp:     queryOpBy(column.QueryType),
	}
	item.IsTextField = item.GoTypeDecl == "string" && (ColumnLength(column.ColumnType) >= 500 || IsTextType(column.ColumnType))
	item.IsRange = item.GoTypeDecl == "time.Time" && column.IsQuery == "1"
	if strings.EqualFold(strings.TrimSpace(column.QueryType), "BETWEEN") && item.GoTypeDecl != "string" {
		item.IsRange = true
	}
	if item.GoTypeDecl == "string" {
		item.SearchType = "string"
	} else {
		item.SearchType = "*" + item.GoTypeDecl
	}
	return item
}

// newRenderColumnPtr 组装渲染列信息（指针形式）
// @param: column *domain.TableColumn 数据库列配置
// @return: *renderColumn
func newRenderColumnPtr(column *domain.TableColumn) *renderColumn {
	item := newRenderColumn(column)
	return &item
}

// firstNonEmpty 返回第一个非空字符串
// @param: values ...string 候选值（按优先级排列）
// @return: string
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// columnExist 判断列是否存在
// @param: columns []domain.TableColumn 列列表
// @param: name string 列名
// @return: bool
func columnExist(columns []domain.TableColumn, name string) bool {
	for i := range columns {
		if columns[i].ColumnName == name {
			return true
		}
	}
	return false
}

// goTypeDecl 数据库列类型映射为框架约定的 Go 类型
// 说明：char(1) 沿用框架中状态/标记列习惯用 uint；时间列用 time.Time；主键统一用 uint（由 prepare 校验为整型）
// @param: columnType string 列类型
// @return: string
func goTypeDecl(columnType string) string {
	switch strings.ToLower(DbType(columnType)) {
	case "char":
		if ColumnLength(columnType) == 1 {
			return "uint"
		}
		return "string"
	case "tinyint", "smallint", "mediumint", "int", "integer", "year":
		return "int"
	case "bigint":
		return "int64"
	case "float", "double", "decimal", "numeric":
		return "float64"
	case "date", "datetime", "timestamp":
		return "time.Time"
	case "bit", "bool", "boolean":
		return "bool"
	}
	return "string"
}

// queryOpBy 查询类型映射为 SQL 运算符
// @param: queryType string 查询类型（EQ/NE/GT/GTE/LT/LTE/LIKE/BETWEEN）
// @return: string
func queryOpBy(queryType string) string {
	switch strings.ToUpper(strings.TrimSpace(queryType)) {
	case "NE":
		return "<>"
	case "GT":
		return ">"
	case "GTE":
		return ">="
	case "LT":
		return "<"
	case "LTE":
		return "<="
	case "LIKE":
		return "LIKE"
	}
	return "="
}

// cleanComment 清理注释中的特殊字符（分号、引号、换行会破坏 struct tag 与 SQL）
// @param: comment string 原始注释
// @return: string
func cleanComment(comment string) string {
	replacer := strings.NewReplacer("`", "", "\"", "", ";", "；", "\n", " ", "\r", " ", "'", "")
	return strings.TrimSpace(replacer.Replace(comment))
}

// TrimTablePrefix 按配置去除表名前缀（多个前缀以英文逗号分隔）
// @param: tableName string 表名
// @param: autoRemovePre bool 是否自动去前缀
// @param: tablePrefix string 表前缀配置
// @return: string
func TrimTablePrefix(tableName string, autoRemovePre bool, tablePrefix string) string {
	if !autoRemovePre || tablePrefix == "" {
		return tableName
	}
	for _, prefix := range strings.Split(tablePrefix, ",") {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" {
			tableName = strings.ReplaceAll(tableName, prefix, "")
		}
	}
	return tableName
}
