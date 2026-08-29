package domain

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"os"
	"github.com/bluefroog/admintpl/common/config"
	"time"
)

var (
	DB *gorm.DB
)

func InitDB(r *gin.Engine) {
	log.Info("app system db init")
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

	autoMigrate(db)

	if config.Bool("database.admin.debug") {
		db = db.Debug()
	}
	DB = db
}

func autoMigrate(db *gorm.DB) {
	// migrate model
	err := db.AutoMigrate(
		User{},       // 后台用户表
		Department{}, // 部门表
		Post{},       // 职位表
		Notice{},     // 公告表
		Config{},     // 参数表
		LoginLog{},   // 文章日志表
		Role{},       // 角色表
		Menu{},       // 菜单表
		DictType{},   // 字典类型表
		DictData{},   // 字典数据表
		Job{},        // 定时任务表
		JobLog{},     // 定时任务日志表
		OperationLog{}, // 操作日志表
		RoleDept{},   // 角色部门关联表
		RoleMenu{},   // 角色菜单关联表
		UserPost{},   // 用户岗位关联表
		UserRole{},   // 用户角色关联表
	)
	if err != nil {
		log.Fatalf("migrate error: %v", err.Error())
		os.Exit(0)
	}
}
