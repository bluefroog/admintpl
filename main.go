package main

import (
	"github.com/gin-gonic/gin"
	"github.com/bluefroog/admintpl/bootstrap"
	"github.com/bluefroog/admintpl/common/config"
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
	_ = r.Run(config.String("service.port")) // listen and serve on 0.0.0.0:8080
}
