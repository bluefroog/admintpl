package system

import (
	"github.com/gin-gonic/gin"
	"github.com/bluefroog/admintpl/app/system/controller"
	"github.com/bluefroog/admintpl/app/system/domain"
)

func Initialize(r *gin.Engine)  {
	// 注册路由
	controller.InitRouter(r)
	// 注册DB
	domain.InitDB(r)
}
