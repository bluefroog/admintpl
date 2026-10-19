package gen

import (
	"github.com/bluefroog/admintpl/app/gen/controller"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Initialize 代码生成模块初始化（路由注册）
func Initialize(r *gin.Engine) {
	controller.InitRouter(r)

	log.Infof("gen 初始化：代码生成配置 包名=%s 模块名=%s 作者=%s",
		config.String("system.gen.package-name"),
		config.String("system.gen.module-name"),
		config.String("system.gen.author"))
}
