package service

import (
	"testing"

	"github.com/bluefroog/admintpl/app/gen/domain"
)

// TestDbTypeAndColumnLength 数据库类型解析
func TestDbTypeAndColumnLength(t *testing.T) {
	cases := []struct {
		columnType string
		dataType   string
		length     int
	}{
		{"varchar(64)", "varchar", 64},
		{"bigint(20)", "bigint", 20},
		{"decimal(10,2)", "decimal", 10},
		{"text", "text", 0},
		{"datetime", "datetime", 0},
	}
	for _, item := range cases {
		if got := DbType(item.columnType); got != item.dataType {
			t.Errorf("DbType(%q) = %q, want %q", item.columnType, got, item.dataType)
		}
		if got := ColumnLength(item.columnType); got != item.length {
			t.Errorf("ColumnLength(%q) = %d, want %d", item.columnType, got, item.length)
		}
	}
}

// TestCamelCase 命名转换
func TestCamelCase(t *testing.T) {
	cases := []struct {
		name  string
		upper string
		lower string
	}{
		{"user_name", "UserName", "userName"},
		{"sys_user_role", "SysUserRole", "sysUserRole"},
		{"status", "Status", "status"},
		{"a_b_c", "ABC", "aBC"},
	}
	for _, item := range cases {
		if got := CamelCase(item.name); got != item.upper {
			t.Errorf("CamelCase(%q) = %q, want %q", item.name, got, item.upper)
		}
		if got := CamelCaseLower(item.name); got != item.lower {
			t.Errorf("CamelCaseLower(%q) = %q, want %q", item.name, got, item.lower)
		}
	}
}

// TestClassNameAndBusinessName 类名与业务名推导
func TestClassNameAndBusinessName(t *testing.T) {
	if got := classNameBy("sys_user", true, "sys_"); got != "User" {
		t.Errorf("classNameBy(sys_user) = %q, want User", got)
	}
	if got := classNameBy("sys_user", false, "sys_"); got != "SysUser" {
		t.Errorf("classNameBy(sys_user, false) = %q, want SysUser", got)
	}
	if got := classNameBy("biz_order_item", true, "sys_,biz_"); got != "OrderItem" {
		t.Errorf("classNameBy(biz_order_item) = %q, want OrderItem", got)
	}
	if got := BusinessName("sys_user_role"); got != "role" {
		t.Errorf("BusinessName(sys_user_role) = %q, want role", got)
	}
	if got := BusinessName("notice"); got != "notice" {
		t.Errorf("BusinessName(notice) = %q, want notice", got)
	}
	if got := FunctionName("用户信息表"); got != "用户信息" {
		t.Errorf("FunctionName(用户信息表) = %q, want 用户信息", got)
	}
}

// TestColumnTypeJudge 数据库类型判断
func TestColumnTypeJudge(t *testing.T) {
	if !IsStringObject("varchar") || !IsStringObject("TEXT") || IsStringObject("bigint") {
		t.Error("IsStringObject 判断错误")
	}
	if !IsTimeObject("datetime") || !IsTimeObject("date") || IsTimeObject("varchar") {
		t.Error("IsTimeObject 判断错误")
	}
	if !IsNumberObject("int") || !IsNumberObject("decimal") || IsNumberObject("varchar") {
		t.Error("IsNumberObject 判断错误")
	}
	if !IsNotEdit("remark") || IsNotEdit("create_time") {
		t.Error("IsNotEdit 判断错误")
	}
	if !IsNotList("remark") || IsNotList("id") {
		t.Error("IsNotList 判断错误")
	}
	if IsNotQuery("remark") || !IsNotQuery("user_name") {
		t.Error("IsNotQuery 判断错误")
	}
}

// TestColumnNameSuffixCheck 字段名后缀判断
func TestColumnNameSuffixCheck(t *testing.T) {
	if !CheckNameColumn("user_name") || CheckNameColumn("namex") {
		t.Error("CheckNameColumn 判断错误")
	}
	if !CheckStatusColumn("status") || CheckStatusColumn("status_x") {
		t.Error("CheckStatusColumn 判断错误")
	}
	if !CheckTypeColumn("notice_type") || CheckTypeColumn("type_x") {
		t.Error("CheckTypeColumn 判断错误")
	}
	if !CheckSexColumn("user_sex") || CheckSexColumn("sexx") {
		t.Error("CheckSexColumn 判断错误")
	}
}

// newColumn 构造测试用列
func newColumn(name, columnType string) domain.TableColumn {
	column := domain.TableColumn{}
	column.ColumnName = name
	column.ColumnType = columnType
	return column
}

// TestInitColumnField 列属性字段初始化
func TestInitColumnField(t *testing.T) {
	table := &domain.Table{}
	table.ID = 9
	table.CreateBy = "admin"
	table.UpdateBy = "admin"

	// 字符串字段 + name 后缀：必填、列表、查询 LIKE
	nameColumn := newColumn("user_name", "varchar(64)")
	InitColumnField(&nameColumn, table)
	if nameColumn.GoType != "string" || nameColumn.HtmlType != "input" {
		t.Errorf("字符串列初始化错误: goType=%s htmlType=%s", nameColumn.GoType, nameColumn.HtmlType)
	}
	if nameColumn.GoField != "UserName" || nameColumn.HtmlField != "userName" {
		t.Errorf("字段名初始化错误: goField=%s htmlField=%s", nameColumn.GoField, nameColumn.HtmlField)
	}
	if nameColumn.IsRequired != "1" || nameColumn.IsInsert != "1" || nameColumn.IsQuery != "1" || nameColumn.QueryType != "LIKE" {
		t.Errorf("name 列配置错误: %+v", nameColumn)
	}
	if nameColumn.TableID != 9 || nameColumn.CreateBy != "admin" {
		t.Errorf("列归属信息错误: %+v", nameColumn)
	}

	// 主键列：不参与编辑、列表与查询（与若依一致）
	pkColumn := newColumn("user_id", "bigint(20)")
	pkColumn.IsPk = "1"
	pkColumn.IsIncrement = "1"
	InitColumnField(&pkColumn, table)
	if pkColumn.GoType != "int64" || pkColumn.IsEdit != "0" || pkColumn.IsList != "0" || pkColumn.IsQuery != "0" {
		t.Errorf("主键列配置错误: %+v", pkColumn)
	}

	// 时间列：Time + datetime
	timeColumn := newColumn("create_time", "datetime")
	InitColumnField(&timeColumn, table)
	if timeColumn.GoType != "Time" || timeColumn.HtmlType != "datetime" {
		t.Errorf("时间列配置错误: %+v", timeColumn)
	}
	if timeColumn.IsInsert != "0" || timeColumn.IsEdit != "0" || timeColumn.IsList != "0" || timeColumn.IsQuery != "0" {
		t.Errorf("create_time 列配置错误: %+v", timeColumn)
	}

	// 状态列：radio
	statusColumn := newColumn("status", "char(1)")
	InitColumnField(&statusColumn, table)
	if statusColumn.HtmlType != "radio" || statusColumn.IsRequired != "1" || statusColumn.IsList != "1" || statusColumn.IsQuery != "1" {
		t.Errorf("状态列配置错误: %+v", statusColumn)
	}

	// 类型列：select
	typeColumn := newColumn("notice_type", "char(1)")
	InitColumnField(&typeColumn, table)
	if typeColumn.HtmlType != "select" {
		t.Errorf("类型列配置错误: %+v", typeColumn)
	}

	// 性别列：select
	sexColumn := newColumn("user_sex", "char(1)")
	InitColumnField(&sexColumn, table)
	if sexColumn.HtmlType != "select" {
		t.Errorf("性别列配置错误: %+v", sexColumn)
	}

	// 长文本：textarea
	textColumn := newColumn("content", "text")
	InitColumnField(&textColumn, table)
	if textColumn.HtmlType != "textarea" || textColumn.GoType != "string" {
		t.Errorf("文本列配置错误: %+v", textColumn)
	}

	// 小数：float64（decimal(m,0) 仍为整型）；tinyint(1)/int(4) -> int；int(11)/bigint(20) -> int64
	decimalCases := []struct {
		columnType string
		goType     string
	}{
		{"decimal(10,2)", "float64"},
		{"decimal(10,0)", "int64"},
		{"tinyint(1)", "int"},
		{"int(4)", "int"},
		{"int(11)", "int64"},
		{"bigint(20)", "int64"},
		{"double", "float64"},
	}
	for _, item := range decimalCases {
		col := newColumn("amount", item.columnType)
		InitColumnField(&col, table)
		if col.GoType != item.goType {
			t.Errorf("数字类型 %s 的 GoType = %s, want %s", item.columnType, col.GoType, item.goType)
		}
	}

	// 未知类型：兜底 string
	blobColumn := newColumn("payload", "blob")
	InitColumnField(&blobColumn, table)
	if blobColumn.GoType != "string" || blobColumn.HtmlType != "input" {
		t.Errorf("未知类型兜底错误: %+v", blobColumn)
	}
}

// TestSetPkColumn 主键列设置
func TestSetPkColumn(t *testing.T) {
	pkColumn := newColumn("user_id", "bigint(20)")
	pkColumn.IsPk = "1"
	table := &domain.Table{Columns: []domain.TableColumn{newColumn("user_name", "varchar(64)"), pkColumn}}
	SetPkColumn(table)
	if table.PkColumn == nil || table.PkColumn.ColumnName != "user_id" {
		t.Errorf("未取到主键列: %+v", table.PkColumn)
	}

	// 无主键时取第一列
	noPk := &domain.Table{Columns: []domain.TableColumn{newColumn("first", "varchar(10)"), newColumn("second", "varchar(10)")}}
	SetPkColumn(noPk)
	if noPk.PkColumn == nil || noPk.PkColumn.ColumnName != "first" {
		t.Errorf("无主键时应取第一列: %+v", noPk.PkColumn)
	}

	// 无列时主键为空
	empty := &domain.Table{}
	SetPkColumn(empty)
	if empty.PkColumn != nil {
		t.Errorf("无列时主键应为空: %+v", empty.PkColumn)
	}
}

// TestParseAndBuildOptions 树表参数解析与组装
func TestParseAndBuildOptions(t *testing.T) {
	table := &domain.Table{TableBase: domain.TableBase{Options: `{"treeCode":"dept_id","treeParentCode":"parent_id","treeName":"dept_name"}`}}
	ParseOptions(table)
	if table.TreeCode != "dept_id" || table.TreeParentCode != "parent_id" || table.TreeName != "dept_name" {
		t.Errorf("树表参数解析错误: %+v", table)
	}

	// 非法 JSON 不影响流程
	invalid := &domain.Table{TableBase: domain.TableBase{Options: "not-json"}}
	ParseOptions(invalid)
	if invalid.TreeCode != "" {
		t.Errorf("非法 JSON 应忽略: %+v", invalid)
	}

	// 未传树参数时沿用原 options
	keep := &domain.TableUpdateRequest{Options: `{"treeCode":"a"}`}
	if got := BuildOptions(keep); got != `{"treeCode":"a"}` {
		t.Errorf("BuildOptions 应沿用原值: %q", got)
	}

	// 传入树参数时重新组装
	build := &domain.TableUpdateRequest{TreeCode: "id", TreeParentCode: "pid", TreeName: "label"}
	want := `{"treeCode":"id","treeParentCode":"pid","treeName":"label"}`
	if got := BuildOptions(build); got != want {
		t.Errorf("BuildOptions = %q, want %q", got, want)
	}
}
