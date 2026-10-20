package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bluefroog/admintpl/app/system/domain"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// newDryRunDB 构造不会连接数据库的 GORM 实例，用于校验生成的 SQL
func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "root:123456@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DisableAutomaticPing: true,
		DryRun:               true,
		NamingStrategy:       schema.NamingStrategy{TablePrefix: "sys_", SingularTable: true},
	})
	if err != nil {
		t.Fatalf("初始化测试用 GORM 失败：%v", err)
	}
	return db
}

func TestNormalizeDataScope(t *testing.T) {
	cases := []struct {
		name      string
		in        uint
		want      uint
		wantError bool
	}{
		{"未传按默认值 1", 0, DataScopeAll, false},
		{"全部数据权限", 1, 1, false},
		{"自定义数据权限", 2, 2, false},
		{"本部门数据权限", 3, 3, false},
		{"本部门及以下数据权限", 4, 4, false},
		{"仅本人数据权限", 5, 5, false},
		{"超出取值范围", 6, 0, true},
		{"非法取值", 99, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeDataScope(c.in)
			if c.wantError {
				if err == nil {
					t.Fatalf("NormalizeDataScope(%d) 期望报错，实际返回 %d", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDataScope(%d) 报错：%v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("NormalizeDataScope(%d) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestDataScopeCondition(t *testing.T) {
	const (
		deptID = 100
		userID = 7
	)
	customCondition := "department_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)"
	deptAndChildCondition := "department_id IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR FIND_IN_SET(?, ancestors))"
	cases := []struct {
		name        string
		scopes      []roleScope
		options     DataScopeOptions
		wantCond    string
		wantArgs    []interface{}
		wantLimited bool
	}{
		{
			name:        "无角色不追加条件",
			scopes:      nil,
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantLimited: false,
		},
		{
			name:        "全部数据权限不追加条件",
			scopes:      []roleScope{{ID: 2, DataScope: DataScopeAll}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantLimited: false,
		},
		{
			name:        "任一角色为全部数据权限即不限制",
			scopes:      []roleScope{{ID: 3, DataScope: DataScopeDepartment}, {ID: 2, DataScope: DataScopeAll}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantLimited: false,
		},
		{
			name:        "自定义数据权限按角色关联部门过滤",
			scopes:      []roleScope{{ID: 7, DataScope: DataScopeCustom}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(" + customCondition + ")",
			wantArgs:    []interface{}{uint(7)},
			wantLimited: true,
		},
		{
			name:        "本部门数据权限",
			scopes:      []roleScope{{ID: 3, DataScope: DataScopeDepartment}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(department_id = ?)",
			wantArgs:    []interface{}{uint(deptID)},
			wantLimited: true,
		},
		{
			name:        "本部门及以下数据权限（ancestors 递归）",
			scopes:      []roleScope{{ID: 4, DataScope: DataScopeDeptAndChild}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(" + deptAndChildCondition + ")",
			wantArgs:    []interface{}{uint(deptID), uint(deptID)},
			wantLimited: true,
		},
		{
			name:        "仅本人数据权限",
			scopes:      []roleScope{{ID: 5, DataScope: DataScopeSelf}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(id = ?)",
			wantArgs:    []interface{}{uint(userID)},
			wantLimited: true,
		},
		{
			name:        "无用户列时仅本人恒不成立",
			scopes:      []roleScope{{ID: 5, DataScope: DataScopeSelf}},
			options:     DataScopeOptions{DeptColumn: "dept_id"},
			wantCond:    "(1 = 0)",
			wantArgs:    []interface{}{},
			wantLimited: true,
		},
		{
			name:        "多个数据范围按 OR 拼接",
			scopes:      []roleScope{{ID: 3, DataScope: DataScopeDepartment}, {ID: 5, DataScope: DataScopeSelf}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(department_id = ? OR id = ?)",
			wantArgs:    []interface{}{uint(deptID), uint(userID)},
			wantLimited: true,
		},
		{
			name:        "相同数据范围去重",
			scopes:      []roleScope{{ID: 3, DataScope: DataScopeDepartment}, {ID: 4, DataScope: DataScopeDepartment}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(department_id = ?)",
			wantArgs:    []interface{}{uint(deptID)},
			wantLimited: true,
		},
		{
			name:        "多个自定义数据权限分别生成条件",
			scopes:      []roleScope{{ID: 7, DataScope: DataScopeCustom}, {ID: 9, DataScope: DataScopeCustom}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(" + customCondition + " OR " + customCondition + ")",
			wantArgs:    []interface{}{uint(7), uint(9)},
			wantLimited: true,
		},
		{
			name:        "无用户列时仅本人不影响其它数据范围",
			scopes:      []roleScope{{ID: 5, DataScope: DataScopeSelf}, {ID: 3, DataScope: DataScopeDepartment}},
			options:     DataScopeOptions{DeptColumn: "dept_id"},
			wantCond:    "(1 = 0 OR dept_id = ?)",
			wantArgs:    []interface{}{uint(deptID)},
			wantLimited: true,
		},
		{
			name:        "无法识别的数据范围被跳过",
			scopes:      []roleScope{{ID: 9, DataScope: 9}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantLimited: false,
		},
		{
			name:        "无法识别的数据范围不影响已识别条件",
			scopes:      []roleScope{{ID: 9, DataScope: 9}, {ID: 3, DataScope: DataScopeDepartment}},
			options:     DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"},
			wantCond:    "(department_id = ?)",
			wantArgs:    []interface{}{uint(deptID)},
			wantLimited: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cond, args, limited := DataScope{}.condition(c.scopes, c.options, deptID, userID)
			if limited != c.wantLimited {
				t.Fatalf("limited = %v, want %v", limited, c.wantLimited)
			}
			if cond != c.wantCond {
				t.Fatalf("conditions = %q, want %q", cond, c.wantCond)
			}
			if !reflect.DeepEqual(args, c.wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, c.wantArgs)
			}
		})
	}
}

func TestDataScopeApply(t *testing.T) {
	options := DataScopeOptions{DeptColumn: "department_id", UserColumn: "id"}

	// 未登录（认证关闭）不追加条件
	db := DataScope{}.Apply(newDryRunDB(t), nil, options)
	var list []domain.User
	stmt := db.Model(&domain.User{}).Session(&gorm.Session{DryRun: true}).Find(&list).Statement
	if sql := stmt.SQL.String(); strings.Contains(sql, "department_id") {
		t.Fatalf("未登录时不应追加数据权限条件，实际 SQL：%s", sql)
	}

	// 超级管理员不追加条件
	db = DataScope{}.Apply(newDryRunDB(t), &domain.LoginUser{UserID: 1, IsAdmin: true}, options)
	stmt = db.Model(&domain.User{}).Session(&gorm.Session{DryRun: true}).Find(&list).Statement
	if sql := stmt.SQL.String(); strings.Contains(sql, "department_id") {
		t.Fatalf("超级管理员不应追加数据权限条件，实际 SQL：%s", sql)
	}

	// 数据范围条件应拼入 SQL，并携带登录用户ID参数
	db = DataScope{}.applyConditions(newDryRunDB(t), "(department_id = ? OR id = ?)", []interface{}{uint(100), uint(7)}, true)
	stmt = db.Model(&domain.User{}).Session(&gorm.Session{DryRun: true}).Find(&list).Statement
	sql := stmt.SQL.String()
	if !strings.Contains(sql, "(department_id = ? OR id = ?)") {
		t.Fatalf("数据权限条件未拼入 SQL：%s", sql)
	}
	want := []interface{}{uint(100), uint(7)}
	if !reflect.DeepEqual(stmt.Vars, want) {
		t.Fatalf("数据权限参数不正确：%#v, want %#v", stmt.Vars, want)
	}

	// limited=false 时查询原样返回
	var plain = newDryRunDB(t)
	if got := (DataScope{}).applyConditions(plain, "1 = 0", nil, false); got != plain {
		t.Fatal("limited=false 时不应修改查询")
	}
}
