// Package database 提供数据库连接管理：支持在配置中声明多个连接（database.connections.<名字>），
// 业务代码默认使用全局 DB（database.default 指定的连接），需要其它库时用 Get/Conn 选择连接。
//
// 配置示例（config/config.yaml）：
//
//	database:
//	  default: admin
//	  connections:
//	    admin: { type: mysql, dsn: "...", prefix: "", singular: true, debug: true }
//	    log:   { type: mysql, dsn: "",    prefix: "", singular: true, debug: false } # dsn 为空则跳过
//
// 用法：
//
//	database.DB                                // 默认连接（各模块既有的用法不变）
//	database.Get("log")                        // 指定连接（不存在时回退默认并告警）
//	db, err := database.Conn("log")            // 指定连接（严格模式，不存在返回错误）
//	_ = database.Use("log")                    // 把全局默认切换为 log 连接
package database

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/bluefroog/admintpl/common/config"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// 数据库配置项（config/config.yaml -> database.*）
const (
	// KeyDefault 默认连接名（database.default / DB_DEFAULT）
	KeyDefault = "database.default"
	// KeyConnections 连接列表前缀（database.connections.<名字>.*）
	KeyConnections = "database.connections"
	// DefaultConnectionName 未配置 database.default 时使用的连接名
	DefaultConnectionName = "admin"
	// DefaultDriverType 未配置 type 时使用的数据库类型
	DefaultDriverType = "mysql"
)

// ErrUnknownConnection 指定的数据库连接不存在（未配置或 dsn 为空被跳过）
var ErrUnknownConnection = errors.New("数据库连接不存在")

// DB 默认数据库句柄（database.default 指定的连接，各业务模块统一使用）
var DB *gorm.DB

var (
	connMu      sync.RWMutex
	connections = make(map[string]*gorm.DB) // 已初始化成功的连接（键为配置中的连接名）
	defaultName = DefaultConnectionName     // 当前默认连接名
	settings    Source                      // 配置来源（可替换，便于单测）
)

// Source 数据库配置读取口（默认读 common/config，单测可注入假实现）
type Source interface {
	String(key string) string
	Int(key string) int
	Bool(key string) bool
	IsSet(key string) bool
	Keys(key string) []string
}

// Connection 单个数据库连接配置
type Connection struct {
	Name        string // 连接名（配置中的键名，如 admin / log）
	Type        string // 数据库类型（目前支持 mysql）
	DSN         string // 连接串（为空表示不启用该连接）
	Prefix      string // 表前缀
	Singular    bool   // 是否使用单数表名
	MaxIdle     int    // 连接池最大闲置连接数
	MaxOpen     int    // 连接池最大打开连接数
	MaxLifetime int    // 连接可复用时长（秒）
	Debug       bool   // 是否打印 SQL
}

func init() {
	settings = configSource{}
}

// configSource 基于 common/config 的配置读取实现
type configSource struct{}

// String 读取字符串配置
func (configSource) String(key string) string { return config.String(key) }

// Int 读取整型配置
func (configSource) Int(key string) int { return config.Int(key) }

// Bool 读取布尔配置
func (configSource) Bool(key string) bool { return config.Bool(key) }

// IsSet 判断配置项是否存在
func (configSource) IsSet(key string) bool { return config.IsSet(key) }

// Keys 读取配置节点下的子键名
func (configSource) Keys(key string) []string { return config.Keys(key) }

// InitDB 初始化 database.connections 下所有已配置 dsn 的连接，并把 database.default 指定的连接写入全局 DB
// @return: *gorm.DB 默认连接句柄
func InitDB() *gorm.DB {
	log.Info("app database init")

	conf, err := LoadConnections()
	if err != nil {
		panic(err)
	}
	if len(conf) == 0 {
		panic(fmt.Errorf("未配置任何数据库连接：请在 config/config.yaml 的 database.connections 下至少配置一个 dsn（%s）", KeyConnections))
	}

	opened := make(map[string]*gorm.DB, len(conf))
	for _, item := range conf {
		db, err := open(item)
		if err != nil {
			panic(fmt.Errorf("数据库连接 %q 初始化失败: %w", item.Name, err))
		}
		opened[item.Name] = db
		log.Infof("数据库连接已初始化: %s（type=%s, prefix=%q, debug=%v）", item.Name, item.Type, item.Prefix, item.Debug)
	}

	name := settings.String(KeyDefault)
	if name == "" {
		name = DefaultConnectionName
	}
	if _, ok := opened[name]; !ok {
		// 默认连接被跳过（如 dsn 留空）时，退而选择第一个可用连接，避免整个应用无法启动
		fallback := firstKey(opened)
		log.Warnf("默认数据库连接 %q 不可用（未配置或 dsn 为空），已改用 %q", name, fallback)
		name = fallback
	}

	connMu.Lock()
	connections = opened
	defaultName = name
	DB = opened[name]
	connMu.Unlock()
	return DB
}

// LoadConnections 读取全部连接配置（按连接名排序，跳过 dsn 为空的连接）
// @return: []Connection 已启用（dsn 非空）的连接配置
// @return: error 配置非法（如 type 不支持）时返回错误
func LoadConnections() ([]Connection, error) {
	names := settings.Keys(KeyConnections)
	conf := make([]Connection, 0, len(names))
	for _, name := range names {
		item := loadConnection(name)
		if item.DSN == "" {
			log.Debugf("数据库连接 %q 未配置 dsn，已跳过", name)
			continue
		}
		if err := validateType(item.Type); err != nil {
			return nil, err
		}
		conf = append(conf, item)
	}
	return conf, nil
}

// loadConnection 读取单个连接的配置
// @param: name string 连接名
// @return: Connection 连接配置
func loadConnection(name string) Connection {
	key := KeyConnections + "." + name
	return Connection{
		Name:        name,
		Type:        stringOrDefault(settings.String(key+".type"), DefaultDriverType),
		DSN:         settings.String(key + ".dsn"),
		Prefix:      settings.String(key + ".prefix"),
		Singular:    !settings.IsSet(key+".singular") || settings.Bool(key+".singular"), // 默认单数表名
		MaxIdle:     intOrDefault(settings.Int(key+".max-idle"), 10),
		MaxOpen:     intOrDefault(settings.Int(key+".max-open"), 10),
		MaxLifetime: intOrDefault(settings.Int(key+".max-lifetime"), 30),
		Debug:       settings.Bool(key + ".debug"),
	}
}

// validateType 校验数据库类型（新增类型时在此登记并在 open 中补对应拨号器）
// @param: dbType string 数据库类型
// @return: error 类型不支持时返回错误
func validateType(dbType string) error {
	switch dbType {
	case "", "mysql":
		return nil
	default:
		return fmt.Errorf("不支持的数据库类型 %q（当前支持：mysql）", dbType)
	}
}

// open 按连接配置打开 gorm 句柄（连接池 + 调试开关 + 命名策略）
// @param: item Connection 连接配置
// @return: *gorm.DB, error
func open(item Connection) (*gorm.DB, error) {
	if err := validateType(item.Type); err != nil {
		return nil, err
	}
	db, err := gorm.Open(mysql.Open(item.DSN), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   item.Prefix,   // 表名前缀，`User` 的表名应该是 `prefix_user`
			SingularTable: item.Singular, // 单数表名，开启后 `User` 的表名是 `user`
		},
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if item.MaxIdle > 0 {
		sqlDB.SetMaxIdleConns(item.MaxIdle) // 连接池最大闲置连接数
	}
	if item.MaxOpen > 0 {
		sqlDB.SetMaxOpenConns(item.MaxOpen) // 连接池最大打开连接数
	}
	if item.MaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(item.MaxLifetime) * time.Second) // 连接可复用时长
	}
	if item.Debug {
		db = db.Debug()
	}
	return db, nil
}

// Default 默认连接句柄（等价于 DB，便于链式调用）
// @return: *gorm.DB 默认连接
func Default() *gorm.DB {
	connMu.RLock()
	defer connMu.RUnlock()
	return DB
}

// DefaultName 当前默认连接名
// @return: string 连接名
func DefaultName() string {
	connMu.RLock()
	defer connMu.RUnlock()
	return defaultName
}

// Names 已初始化的连接名（排序）
// @return: []string 连接名列表
func Names() []string {
	connMu.RLock()
	defer connMu.RUnlock()
	return sortedNames(connections)
}

// Conn 严格获取指定连接（name 为空返回默认连接）
// @param: name string 连接名
// @return: *gorm.DB 连接句柄
// @return: error 连接不存在时返回 ErrUnknownConnection
func Conn(name string) (*gorm.DB, error) {
	if name == "" {
		return Default(), nil
	}
	connMu.RLock()
	db, ok := connections[name]
	connMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s（可用连接：%v）", ErrUnknownConnection, name, Names())
	}
	return db, nil
}

// Get 选择指定连接（name 为空返回默认连接；不存在时回退默认连接并告警）
// @param: name string 连接名
// @return: *gorm.DB 连接句柄（一定非 nil，除非 InitDB 未执行）
func Get(name string) *gorm.DB {
	db, err := Conn(name)
	if err != nil {
		log.Warnf("%s，已回退默认连接 %q", err.Error(), DefaultName())
		return Default()
	}
	return db
}

// Use 把全局默认连接切换为指定连接（运行期按需切换，谨慎用于跨库查询）
// @param: name string 连接名
// @return: error 连接不存在时返回 ErrUnknownConnection
func Use(name string) error {
	db, err := Conn(name)
	if err != nil {
		return err
	}
	connMu.Lock()
	defaultName = name
	DB = db
	connMu.Unlock()
	return nil
}

// Close 关闭全部数据库连接（进程退出时调用）
func Close() {
	connMu.Lock()
	defer connMu.Unlock()
	for name, db := range connections {
		sqlDB, err := db.DB()
		if err != nil {
			log.Warnf("获取数据库连接 %q 失败: %s", name, err)
			continue
		}
		if err = sqlDB.Close(); err != nil {
			log.Warnf("关闭数据库连接 %q 失败: %s", name, err)
		}
	}
	connections = make(map[string]*gorm.DB)
	DB = nil
}

// AutoMigrateAll 按模块批量迁移表结构（各模块 domain.Models() 依次传入，使用默认连接）
func AutoMigrateAll(groups ...[]interface{}) {
	var models []interface{}
	for _, group := range groups {
		models = append(models, group...)
	}
	if len(models) == 0 {
		return
	}
	if err := DB.AutoMigrate(models...); err != nil {
		log.Fatalf("migrate error: %v", err.Error())
		os.Exit(0)
	}
}

// TableEmpty 判断表是否为空（基础数据初始化用）
func TableEmpty(tx *gorm.DB, model interface{}) (bool, error) {
	var count int64
	if err := tx.Model(model).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// firstKey 返回 map 中排序后的第一个键（用于默认连接回退）
func firstKey(m map[string]*gorm.DB) string {
	names := sortedNames(m)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// sortedNames 返回 map 的键名并排序
func sortedNames(m map[string]*gorm.DB) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// stringOrDefault 字符串配置为空时使用默认值
func stringOrDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

// intOrDefault 整型配置 <=0 时使用默认值
func intOrDefault(value, def int) int {
	if value <= 0 {
		return def
	}
	return value
}

