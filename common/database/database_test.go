package database

import (
	"errors"
	"reflect"
	"testing"

	"gorm.io/gorm"
)

// fakeSource 单测用的配置来源（避免依赖真实配置文件与 MySQL 连接）
type fakeSource struct {
	str  map[string]string
	num  map[string]int
	flag map[string]bool
	keys map[string][]string
}

// String 读取字符串配置
func (f fakeSource) String(key string) string { return f.str[key] }

// Int 读取整型配置
func (f fakeSource) Int(key string) int { return f.num[key] }

// Bool 读取布尔配置
func (f fakeSource) Bool(key string) bool { return f.flag[key] }

// IsSet 判断配置项是否存在
func (f fakeSource) IsSet(key string) bool {
	if _, ok := f.str[key]; ok {
		return true
	}
	if _, ok := f.num[key]; ok {
		return true
	}
	_, ok := f.flag[key]
	return ok
}

// Keys 读取子键名
func (f fakeSource) Keys(key string) []string { return f.keys[key] }

// withSource 临时替换配置来源，测试结束自动还原
func withSource(t *testing.T, src Source) {
	t.Helper()
	saved := settings
	settings = src
	t.Cleanup(func() { settings = saved })
}

// TestLoadConnections 校验多连接配置的读取、默认值与空 dsn 跳过
func TestLoadConnections(t *testing.T) {
	withSource(t, fakeSource{
		str: map[string]string{
			"database.connections.admin.dsn":    "root:123456@tcp(127.0.0.1:3306)/admintpl",
			"database.connections.admin.prefix": "sys_",
			"database.connections.log.dsn":      "root:123456@tcp(127.0.0.1:3307)/logs",
			"database.connections.log.prefix":   "",
			"database.connections.log.type":     "mysql",
		},
		num: map[string]int{
			"database.connections.admin.max-open": 20,
		},
		flag: map[string]bool{
			"database.connections.log.debug":     true,
			"database.connections.log.singular":  false,
			"database.connections.admin.singular": false,
		},
		keys: map[string][]string{
			KeyConnections: {"admin", "log", "empty"},
		},
	})

	conf, err := LoadConnections()
	if err != nil {
		t.Fatalf("读取连接配置失败: %v", err)
	}
	if len(conf) != 2 {
		t.Fatalf("应跳过 dsn 为空的连接，实际得到 %d 个: %+v", len(conf), conf)
	}

	admin := conf[0]
	if admin.Name != "admin" {
		t.Fatalf("连接未按名字排序: %+v", conf)
	}
	if admin.Type != DefaultDriverType {
		t.Errorf("type 未配置时应使用默认值 %q，实际 %q", DefaultDriverType, admin.Type)
	}
	if admin.Prefix != "sys_" {
		t.Errorf("prefix 解析错误: %q", admin.Prefix)
	}
	if admin.Singular {
		t.Error("singular 显式配置为 false 时应保留 false")
	}
	if admin.MaxIdle != 10 || admin.MaxLifetime != 30 {
		t.Errorf("连接池默认值错误: max-idle=%d max-lifetime=%d", admin.MaxIdle, admin.MaxLifetime)
	}
	if admin.MaxOpen != 20 {
		t.Errorf("max-open 解析错误: %d", admin.MaxOpen)
	}
	if admin.Debug {
		t.Error("admin.debug 未开启")
	}

	item := conf[1]
	if item.Singular {
		t.Error("singular 显式配置为 false 时应保留 false")
	}
	if !item.Debug {
		t.Error("log.debug 应为 true")
	}

	// singular 未配置时应默认 true（单数表名）
	withSource(t, fakeSource{
		str: map[string]string{
			"database.connections.admin.dsn": "root:123456@tcp(127.0.0.1:3306)/admintpl",
		},
		keys: map[string][]string{KeyConnections: {"admin"}},
	})
	conf, err = LoadConnections()
	if err != nil {
		t.Fatalf("读取连接配置失败: %v", err)
	}
	if len(conf) != 1 || !conf[0].Singular {
		t.Fatalf("singular 未配置时应默认 true: %+v", conf)
	}
}

// TestLoadConnectionsUnsupportedType 校验不支持的数据库类型及时报错
func TestLoadConnectionsUnsupportedType(t *testing.T) {
	withSource(t, fakeSource{
		str: map[string]string{
			"database.connections.report.dsn":  "postgres://127.0.0.1:5432/report",
			"database.connections.report.type": "postgres",
		},
		keys: map[string][]string{KeyConnections: {"report"}},
	})

	if _, err := LoadConnections(); err == nil {
		t.Fatal("不支持的数据库类型应返回错误")
	}
}

// TestLoadConnectionsEmpty 校验一个连接都没启用时返回空列表（由 InitDB 决定是否报错）
func TestLoadConnectionsEmpty(t *testing.T) {
	withSource(t, fakeSource{keys: map[string][]string{}})
	conf, err := LoadConnections()
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if len(conf) != 0 {
		t.Fatalf("应为空列表: %+v", conf)
	}
}

// TestConnSelection 校验按名称选择连接、缺失回退与默认连接切换（不依赖真实数据库）
func TestConnSelection(t *testing.T) {
	admin := &gorm.DB{Config: &gorm.Config{}}
	logs := &gorm.DB{Config: &gorm.Config{}}

	connMu.Lock()
	savedConns, savedDefault, savedDB := connections, defaultName, DB
	connections = map[string]*gorm.DB{"admin": admin, "log": logs}
	defaultName = "admin"
	DB = admin
	connMu.Unlock()
	t.Cleanup(func() {
		connMu.Lock()
		connections, defaultName, DB = savedConns, savedDefault, savedDB
		connMu.Unlock()
	})

	if got := Names(); !reflect.DeepEqual(got, []string{"admin", "log"}) {
		t.Fatalf("Names() = %v", got)
	}
	if got := DefaultName(); got != "admin" {
		t.Fatalf("DefaultName() = %q", got)
	}
	if got := Default(); got != admin {
		t.Fatal("Default() 应返回默认连接")
	}
	if got := Get(""); got != admin {
		t.Fatal("Get(\"\") 应返回默认连接")
	}

	db, err := Conn("log")
	if err != nil || db != logs {
		t.Fatalf("Conn(\"log\") = %v, %v", db, err)
	}

	// 未知连接：严格模式报错，便捷模式回退默认连接
	if _, err = Conn("not-exist"); !errors.Is(err, ErrUnknownConnection) {
		t.Fatalf("未知连接应返回 ErrUnknownConnection，实际 %v", err)
	}
	if got := Get("not-exist"); got != admin {
		t.Fatal("Get(未知连接) 应回退默认连接")
	}
	if err = Use("not-exist"); !errors.Is(err, ErrUnknownConnection) {
		t.Fatalf("Use(未知连接) 应返回 ErrUnknownConnection，实际 %v", err)
	}

	// 切换默认连接
	if err = Use("log"); err != nil {
		t.Fatalf("Use(\"log\") 失败: %v", err)
	}
	if got := DefaultName(); got != "log" {
		t.Fatalf("切换后 DefaultName() = %q", got)
	}
	if DB != logs {
		t.Fatal("切换后全局 DB 应指向 log 连接")
	}
}

// TestClose 校验关闭全部连接后注册表被清空
func TestClose(t *testing.T) {
	connMu.Lock()
	savedConns, savedDB := connections, DB
	connections = map[string]*gorm.DB{}
	DB = nil
	connMu.Unlock()
	t.Cleanup(func() {
		connMu.Lock()
		connections, DB = savedConns, savedDB
		connMu.Unlock()
	})

	Close() // 空注册表下不应 panic
	if DB != nil {
		t.Fatal("Close() 后 DB 应为 nil")
	}
	if len(Names()) != 0 {
		t.Fatalf("Close() 后连接列表应为空: %v", Names())
	}
}

// TestSortedNames 校验键名排序（保证 InitDB 日志与回退连接稳定）
func TestSortedNames(t *testing.T) {
	got := sortedNames(map[string]*gorm.DB{"log": nil, "admin": nil, "business": nil})
	want := []string{"admin", "business", "log"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedNames() = %v, want %v", got, want)
	}
}
