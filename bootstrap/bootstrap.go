package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/bluefroog/admintpl/app/swagger"
	"github.com/bluefroog/admintpl/app/system"
)

func Bootstrap(r *gin.Engine) error {
	// todo 后期可以做成配置反射调用
	system.Initialize(r)  // 后台管理服务
	swagger.Initialize(r) // swagger 服务
	return nil
}
