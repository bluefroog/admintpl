package database

import (
	"os"
	"time"

	"github.com/bluefroog/admintpl/common/config"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DB 全局数据库句柄（各业务模块统一使用）
var DB *gorm.DB

// InitDB 初始化数据库连接（连接池 + 调试开关），并写入全局句柄 DB
func InitDB() *gorm.DB {
	log.Info("app database init")
	prefix := config.String("database.admin.prefix")
	dsn := config.String("database.admin.dsn")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   prefix, // 表名前缀，`User` 的表名应该是 `prefix_users`
			SingularTable: true,   // 使用单数表名，启用该选项，此时，`User` 的表名应该是 `prefix_user`
		},
	})
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	maxIdle := config.Int("database.admin.max-idle")         // 连接池最大闲置的连接数
	maxOpen := config.Int("database.admin.max-open")         // 连接池最大打开的连接数
	maxLifetime := config.Int("database.admin.max-lifetime") // 连接对象可重复使用的时间长度(秒)
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(maxIdle)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(maxOpen)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime))

	if config.Bool("database.admin.debug") {
		db = db.Debug()
	}
	DB = db
	return DB
}

// AutoMigrateAll 按模块批量迁移表结构（各模块 domain.Models() 依次传入）
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
