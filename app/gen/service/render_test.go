package service

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bluefroog/admintpl/app/gen/domain"
)

// newColumnOf 构造表字段（TableColumn 的字段由 TableColumnBase 提升，无法用复合字面量直接赋值）
// @param: name string 列名
// @param: comment string 列注释
// @param: columnType string 列类型
// @param: required string 是否必填
// @param: query string 是否查询列
// @param: queryType string 查询方式
// @param: sort uint 排序
// @return: domain.TableColumn
func newColumnOf(name, comment, columnType, required, query, queryType string, sort uint) domain.TableColumn {
	column := domain.TableColumn{}
	column.ColumnName = name
	column.ColumnComment = comment
	column.ColumnType = columnType
	column.GoField = CamelCase(name)
	column.GoType = goTypeDecl(columnType)
	column.IsRequired = required
	column.IsQuery = query
	column.QueryType = queryType
	column.Sort = sort
	if name == "demo_id" {
		column.IsPk = "1"
		column.IsIncrement = "1"
		column.IsInsert = "0"
		column.IsEdit = "0"
		column.IsList = "0"
		column.IsQuery = "0"
	} else {
		column.IsInsert = "1"
		column.IsEdit = "1"
		column.IsList = "1"
	}
	return column
}

// newDemoTable 构造一张用于渲染测试的表结构（各类字段 + 公共审计列）
// @return: *domain.Table
func newDemoTable() *domain.Table {
	columns := []domain.TableColumn{
		newColumnOf("demo_id", "主键", "bigint(20)", "0", "0", "EQ", 1),
		newColumnOf("demo_name", "名称", "varchar(64)", "1", "1", "LIKE", 2),
		newColumnOf("status", "状态", "char(1)", "1", "1", "EQ", 3),
		newColumnOf("price", "价格", "decimal(10,2)", "0", "0", "EQ", 4),
		newColumnOf("content", "内容", "text", "0", "0", "EQ", 5),
		newColumnOf("create_by", "创建者", "varchar(64)", "0", "0", "EQ", 6),
		newColumnOf("create_time", "创建时间", "datetime", "0", "0", "EQ", 7),
		newColumnOf("update_by", "更新者", "varchar(64)", "0", "0", "EQ", 8),
		newColumnOf("update_time", "更新时间", "datetime", "0", "0", "EQ", 9),
		newColumnOf("remark", "备注", "varchar(500)", "0", "0", "EQ", 10),
		newColumnOf("publish_time", "发布时间", "datetime", "0", "1", "BETWEEN", 11),
	}
	table := &domain.Table{Columns: columns}
	table.ID = 1
	table.Name = "demo"
	table.TableComment = "示例表"
	table.ClassName = "Demo"
	table.TplCategory = "crud"
	table.ModuleName = "system"
	table.BusinessName = "demo"
	table.FunctionName = "示例"
	table.Options = ""
	table.PkColumn = &table.Columns[0]
	return table
}

// TestBuildRenderData 渲染上下文组装
func TestBuildRenderData(t *testing.T) {
	data, err := buildRenderData(newDemoTable())
	if err != nil {
		t.Fatalf("buildRenderData 失败: %v", err)
	}
	if data.ClassName != "Demo" {
		t.Errorf("ClassName = %q, want %q", data.ClassName, "Demo")
	}
	if data.FileName != "demo" {
		t.Errorf("FileName = %q, want %q", data.FileName, "demo")
	}
	if data.ClassNameLower != "demo" {
		t.Errorf("ClassNameLower = %q, want %q", data.ClassNameLower, "demo")
	}
	if data.FunctionName != "示例" {
		t.Errorf("FunctionName = %q, want %q", data.FunctionName, "示例")
	}
	if data.RoutePrefix != "/system" {
		t.Errorf("RoutePrefix = %q, want %q", data.RoutePrefix, "/system")
	}
	if data.PermissionPre != "system:demo" {
		t.Errorf("PermissionPre = %q, want %q", data.PermissionPre, "system:demo")
	}
	if !data.UseRuoYiModel {
		t.Errorf("UseRuoYiModel = %v, want true", data.UseRuoYiModel)
	}
	// 存在 datetime 字段，实体与 service 都需要 import "time"
	if !data.ImportTime || !data.ServiceTime {
		t.Errorf("ImportTime = %v, ServiceTime = %v, want true/true", data.ImportTime, data.ServiceTime)
	}
	if !data.ImportStrings {
		t.Error("ImportStrings = false, want true（存在 LIKE 字符串查询列）")
	}
	// 主键与审计列不进入实体字段
	for _, field := range data.Fields {
		switch field.ColumnName {
		case "demo_id", "create_by", "create_time", "update_by", "update_time", "remark":
			t.Errorf("Fields 不应包含 %q", field.ColumnName)
		}
	}
	if len(data.Fields) != 5 {
		t.Errorf("Fields 数量 = %d, want 5", len(data.Fields))
	}
	if len(data.QueryField) != 3 {
		t.Errorf("QueryField 数量 = %d, want 3", len(data.QueryField))
	}
	// 时间列按区间查询
	for _, field := range data.QueryField {
		if field.ColumnName == "publish_time" && !field.IsRange {
			t.Error("datetime 查询列应标记为区间查询")
		}
	}
	if len(data.KeyField) != 1 || data.KeyField[0].ColumnName != "demo_name" {
		t.Errorf("KeyField = %+v, want [demo_name]", data.KeyField)
	}
	// 类型映射
	wantTypes := map[string]string{
		"demo_name":    "string",
		"status":       "uint",
		"price":        "float64",
		"content":      "string",
		"publish_time": "time.Time",
	}
	for _, field := range data.Fields {
		if want, ok := wantTypes[field.ColumnName]; ok && field.GoTypeDecl != want {
			t.Errorf("%s GoTypeDecl = %q, want %q", field.ColumnName, field.GoTypeDecl, want)
		}
	}
	if len(data.AllField) != len(newDemoTable().Columns) {
		t.Errorf("AllField 数量 = %d, want %d", len(data.AllField), len(newDemoTable().Columns))
	}
}

// TestBuildRenderDataErrors 异常表结构
func TestBuildRenderDataErrors(t *testing.T) {
	if _, err := buildRenderData(nil); err == nil {
		t.Error("table 为 nil 时应返回错误")
	}
	empty := &domain.Table{}
	empty.Name = "sys_empty"
	if _, err := buildRenderData(empty); err == nil {
		t.Error("无字段时应返回错误")
	}
	noPk := newDemoTable()
	noPk.PkColumn = nil
	if _, err := buildRenderData(noPk); err == nil {
		t.Error("无主键时应返回错误")
	}
	strPk := newDemoTable()
	strPk.PkColumn = &strPk.Columns[1] // varchar 主键
	if _, err := buildRenderData(strPk); err == nil {
		t.Error("字符串主键应返回错误")
	}
}

// TestRenderAll 全量渲染（Go 代码需通过 gofmt 语法校验）
func TestRenderAll(t *testing.T) {
	data, err := buildRenderData(newDemoTable())
	if err != nil {
		t.Fatalf("buildRenderData 失败: %v", err)
	}
	files, err := renderAll(data)
	if err != nil {
		t.Fatalf("renderAll 失败: %v", err)
	}
	if len(files) != 6 {
		t.Fatalf("生成文件数量 = %d, want 6", len(files))
	}
	// GEN_DUMP=<目录> 时把生成结果落盘，便于人工检查生成代码
	if dir := os.Getenv("GEN_DUMP"); dir != "" {
		for name, content := range files {
			// 不同目录下同名文件（domain/service/controller 都是 demo.go）加目录前缀区分
			target := filepath.Join(dir, filepath.Base(filepath.Dir(name))+"_"+filepath.Base(name))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatalf("创建目录失败: %v", err)
			}
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				t.Fatalf("写入 %s 失败: %v", target, err)
			}
		}
	}
	domainFile := files["github.com/bluefroog/admintpl/app/system/domain/demo.go"]
	serviceFile := files["github.com/bluefroog/admintpl/app/system/service/demo.go"]
	controllerFile := files["github.com/bluefroog/admintpl/app/system/controller/demo.go"]
	httpFile := files["github.com/bluefroog/admintpl/app/system/test/http/demo.http"]
	sqlFile := files["docs/sql/demo.sql"]
	registerFile := files["github.com/bluefroog/admintpl/app/system/demo.register.md"]
	for name, content := range map[string]string{
		"domain": domainFile, "service": serviceFile, "controller": controllerFile,
		"http": httpFile, "sql": sqlFile, "register": registerFile,
	} {
		if strings.TrimSpace(content) == "" {
			t.Fatalf("%s 生成内容为空", name)
		}
	}

	// 生成的 Go 代码经 gofmt 对齐（多个空格），断言前先归一化空白
	flatDomain := strings.Join(strings.Fields(domainFile), " ")

	// 实体
	for _, want := range []string{
		"package domain",
		"type Demo struct {",
		"func (m *Demo) TableName() string {",
		`return "demo"`,
		"type DemoBase struct {",
		"core.RuoyiModel",
		"type DemoSearchRequest struct {",
		"type DemoCreateRequest struct {",
		"type DemoUpdateRequest struct {",
		"DemoName string",
		"Price float64",
		"Status uint",
		`gorm:"column:demo_id;type:bigint(20) auto_increment;primary_key;not null;comment:主键"`,
	} {
		if !strings.Contains(flatDomain, want) {
			t.Errorf("domain 生成内容缺少: %s", want)
		}
	}
	// 主键不进实体 Base（由 ID 承载），查询列/关键字
	if strings.Contains(domainFile, "DemoID int64") {
		t.Error("domain 不应重复生成主键字段")
	}
	if !strings.Contains(domainFile, "Keyword string") {
		t.Error("domain 缺少 keyword 字段")
	}
	if !strings.Contains(domainFile, `json:"demo_name" form:"demo_name" validate:"required"`) {
		t.Error("domain 的 demo_name 缺少 validate 标签")
	}
	if !strings.Contains(flatDomain, "PublishTime time.Time") {
		t.Error("domain 缺少 PublishTime time.Time 字段")
	}
	if !strings.Contains(flatDomain, "BeginPublishTime string") || !strings.Contains(flatDomain, "EndPublishTime string") {
		t.Error("domain 缺少时间区间查询字段")
	}
	if !strings.Contains(serviceFile, `db.Where("publish_time >= ?", searchParams.BeginPublishTime)`) ||
		!strings.Contains(serviceFile, `db.Where("publish_time <= ?", searchParams.EndPublishTime)`) {
		t.Error("service 缺少时间区间查询条件")
	}
	if !strings.Contains(domainFile, `json:"status" form:"status"`) ||
		strings.Contains(domainFile, `json:"status" form:"status" validate:"required"`) {
		t.Error("非字符串的必填列不应生成 validate:\"required\"（会把 0 判为非法）")
	}

	// 业务逻辑
	for _, want := range []string{
		"package service",
		"type Demo struct{}",
		"func (s Demo) GetCount(",
		"func (s Demo) GetList(",
		"func (s Demo) parseFilter(",
		"func (s Demo) Create(",
		"func (s Demo) Update(",
		"func (s Demo) UpdateByMap(",
		"func (s Demo) Delete(",
		"func (s Demo) Detail(",
		`strings.Trim(searchParams.Keyword`,
		`record["DemoName"] = data.DemoName`,
		"db.Order(\"demo_id desc\")",
	} {
		if !strings.Contains(serviceFile, want) {
			t.Errorf("service 生成内容缺少: %s", want)
		}
	}

	// 控制器（swag 注解）
	for _, want := range []string{
		"package controller",
		"demo = service.Demo{}",
		"@Summary 获取示例列表",
		"@Router /system/demo/list [get]",
		"@Router /system/demo/create [post]",
		"@Router /system/demo/update [put]",
		"@Router /system/demo/delete [delete]",
		"@Router /system/demo/detail [get]",
		"response.Success(\"添加成功\", c)",
	} {
		if !strings.Contains(controllerFile, want) {
			t.Errorf("controller 生成内容缺少: %s", want)
		}
	}

	// 接口测试文件
	if !strings.Contains(httpFile, "GET http://127.0.0.1:8080/system/demo/list") {
		t.Error("http 缺少列表用例")
	}
	if !strings.Contains(httpFile, `Bearer {{token}}`) {
		t.Error("http 缺少令牌占位符")
	}
	if !strings.Contains(httpFile, `"demo_name": "名称"`) {
		t.Error("http 添加用例缺少 demo_name")
	}

	// SQL
	for _, want := range []string{
		"DROP TABLE IF EXISTS `demo`;",
		"`demo_id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '主键'",
		"PRIMARY KEY (`demo_id`)",
		"COMMENT='示例表'",
	} {
		if !strings.Contains(sqlFile, want) {
			t.Errorf("sql 生成内容缺少: %s", want)
		}
	}

	// 接入说明
	for _, want := range []string{
		"# 示例（demo）代码生成接入说明",
		"domain.Demo{}",
		`group.GET("/demo/list", demo.List)`,
		"`system:demo:list`",
	} {
		if !strings.Contains(registerFile, want) {
			t.Errorf("register 生成内容缺少: %s", want)
		}
	}
}

// TestRenderZip 打包下载内容
func TestRenderZip(t *testing.T) {
	data, err := buildRenderData(newDemoTable())
	if err != nil {
		t.Fatalf("buildRenderData 失败: %v", err)
	}
	files, err := renderAll(data)
	if err != nil {
		t.Fatalf("renderAll 失败: %v", err)
	}
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	for _, name := range renderOrder(data) {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("zip Create 失败: %v", err)
		}
		if _, err := file.Write([]byte(files[name])); err != nil {
			t.Fatalf("zip Write 失败: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("zip Close 失败: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("读取 zip 失败: %v", err)
	}
	if len(reader.File) != len(files) {
		t.Errorf("zip 文件数量 = %d, want %d", len(reader.File), len(files))
	}
	for _, file := range reader.File {
		if _, ok := files[file.Name]; !ok {
			t.Errorf("zip 中出现未知文件: %s", file.Name)
		}
	}
}

// TestTrimTablePrefix 表前缀去除
func TestTrimTablePrefix(t *testing.T) {
	cases := []struct {
		tableName     string
		autoRemovePre bool
		tablePrefix   string
		want          string
	}{
		{"sys_post", true, "sys_", "post"},
		{"sys_post", false, "sys_", "sys_post"},
		{"biz_order", true, "sys_,biz_", "order"},
		{"post", true, "", "post"},
	}
	for _, item := range cases {
		if got := TrimTablePrefix(item.tableName, item.autoRemovePre, item.tablePrefix); got != item.want {
			t.Errorf("TrimTablePrefix(%q, %v, %q) = %q, want %q",
				item.tableName, item.autoRemovePre, item.tablePrefix, got, item.want)
		}
	}
}

// TestCleanComment 注释清理
func TestCleanComment(t *testing.T) {
	if got := cleanComment(" 状态;1是 '正常' `x` \"y\" "); strings.ContainsAny(got, ";`\"'") {
		t.Errorf("cleanComment 未清理特殊字符: %q", got)
	}
}

// TestRenderWithoutAuditColumns 无审计列的表不嵌入 core.RuoyiModel、不 import "time"
func TestRenderWithoutAuditColumns(t *testing.T) {
	table := newDemoTable()
	columns := make([]domain.TableColumn, 0, len(table.Columns))
	for _, column := range table.Columns {
		switch column.ColumnName {
		case "create_by", "update_by", "create_time", "update_time", "publish_time":
			continue
		}
		columns = append(columns, column)
	}
	table.Columns = columns

	data, err := buildRenderData(table)
	if err != nil {
		t.Fatalf("buildRenderData 失败: %v", err)
	}
	if data.UseRuoYiModel || data.ImportTime || data.ServiceTime {
		t.Errorf("无审计列的表不应标记审计/时间依赖: useRuoYiModel=%v importTime=%v serviceTime=%v",
			data.UseRuoYiModel, data.ImportTime, data.ServiceTime)
	}
	files, err := renderAll(data)
	if err != nil {
		t.Fatalf("renderAll 失败: %v", err)
	}
	domainFile := strings.Join(strings.Fields(files[data.PackageName+"/app/system/domain/demo.go"]), " ")
	if strings.Contains(domainFile, "core.RuoyiModel") || strings.Contains(domainFile, "\"time\"") {
		t.Error("无审计列实体的导入不应包含 core / time")
	}
	serviceFile := strings.Join(strings.Fields(files[data.PackageName+"/app/system/service/demo.go"]), " ")
	if strings.Contains(serviceFile, "\"time\"") {
		t.Error("无审计列的服务不应 import time")
	}
	if !strings.Contains(serviceFile, `"gorm.io/gorm"`) || !strings.Contains(serviceFile, `"`+data.PackageName+`/common/database"`) {
		t.Error("服务缺少必要导入")
	}
}

// TestRenderTreeTable 树表模板：列表按树编码升序，接入说明输出树参数
func TestRenderTreeTable(t *testing.T) {
	table := newDemoTable()
	table.TplCategory = "tree"
	table.Options = `{"treeCode":"demo_id","treeParentCode":"parent_id","treeName":"demo_name"}`

	data, err := buildRenderData(table)
	if err != nil {
		t.Fatalf("buildRenderData 失败: %v", err)
	}
	if !data.IsTree || data.TreeCode != "demo_id" || data.TreeParentCode != "parent_id" || data.TreeName != "demo_name" {
		t.Errorf("树表参数解析错误: isTree=%v code=%s parent=%s name=%s",
			data.IsTree, data.TreeCode, data.TreeParentCode, data.TreeName)
	}
	files, err := renderAll(data)
	if err != nil {
		t.Fatalf("renderAll 失败: %v", err)
	}
	serviceFile := strings.Join(strings.Fields(files[data.PackageName+"/app/system/service/demo.go"]), " ")
	if !strings.Contains(serviceFile, `db.Order("demo_id asc")`) {
		t.Error("树表列表应按树编码升序排序")
	}
	registerFile := files[data.PackageName+"/app/system/demo.register.md"]
	for _, want := range []string{"treeCode=demo_id", "treeParentCode=parent_id", "treeName=demo_name", "GetTreeList"} {
		if !strings.Contains(registerFile, want) {
			t.Errorf("树表接入说明缺少 %q", want)
		}
	}
}

// TestRenderHonorsTableConfig gen_table 中按表配置的值优先于全局配置
func TestRenderHonorsTableConfig(t *testing.T) {
	table := newDemoTable()
	table.PackageName = "example.com/demo"
	table.ModuleName = "biz"
	table.ClassName = "CustomDemo"
	table.FunctionName = "自定义功能"
	table.FunctionAuthor = "tester"

	data, err := buildRenderData(table)
	if err != nil {
		t.Fatalf("buildRenderData 失败: %v", err)
	}
	if data.ClassName != "CustomDemo" || data.FunctionName != "自定义功能" || data.Author != "tester" {
		t.Errorf("按表配置未生效: className=%s functionName=%s author=%s",
			data.ClassName, data.FunctionName, data.Author)
	}
	if data.RoutePrefix != "/biz" || data.PermissionPre != "biz:demo" {
		t.Errorf("路由/权限前缀错误: route=%s permission=%s", data.RoutePrefix, data.PermissionPre)
	}
	files, err := renderAll(data)
	if err != nil {
		t.Fatalf("renderAll 失败: %v", err)
	}
	domainPath := "example.com/demo/app/biz/domain/demo.go"
	content, ok := files[domainPath]
	if !ok {
		t.Fatalf("缺少生成文件 %s，实际：%v", domainPath, keysOf(files))
	}
	if !strings.Contains(content, "type CustomDemo struct {") {
		t.Error("实体类名未使用按表配置的 class_name")
	}
	if !strings.Contains(content, `json:"id" form:"id"`) {
		t.Error("实体缺少主键字段")
	}
}

// TestRenderRejectsInvalidConfig 非法配置直接报错，避免生成不可用代码
func TestRenderRejectsInvalidConfig(t *testing.T) {
	badModule := newDemoTable()
	badModule.ModuleName = "../system"
	if _, err := buildRenderData(badModule); err == nil {
		t.Error("非法模块名应返回错误")
	}
	badClass := newDemoTable()
	badClass.ClassName = "Demo-Hack"
	if _, err := buildRenderData(badClass); err == nil {
		t.Error("非法类名应返回错误")
	}
}

// keysOf 返回 map 的键（仅用于错误提示）
// @param: m map[string]string
// @return: []string
func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
