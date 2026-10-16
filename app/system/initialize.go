package system

import (
	"github.com/bluefroog/admintpl/app/system/controller"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/gin-gonic/gin"
)

func Initialize(r *gin.Engine) {
	// 注册路由
	controller.InitRouter(r)
	// 注册DB
	domain.InitDB(r)
}
