package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bluefroog/admintpl/bootstrap"
	"github.com/bluefroog/admintpl/common/cache"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/database"
	"github.com/gin-gonic/gin"
)

// @title github.com/bluefroog/admintpl API文档
// @version 1.0
// @description github.com/bluefroog/admintpl 在线API文档
// @host localhost
// @BasePath /
func main() {
	if config.Bool("service.debug") {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	_ = bootstrap.Bootstrap(r)

	// 优雅退出：先把本地文件缓存落盘、关闭 redis/memcache 与数据库连接（容器收到 SIGTERM 时同样生效）
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cache.Shutdown()
		database.Close()
		os.Exit(0)
	}()

	_ = r.Run(config.String("service.port")) // listen and serve on 0.0.0.0:8080
}
