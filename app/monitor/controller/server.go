package controller

import (
	"github.com/bluefroog/admintpl/app/monitor/service"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/gin-gonic/gin"
)

type Server struct{}

var (
	server = service.Server{}
)

// Info 获取服务器监控信息
// @Summary 获取服务器监控信息
// @Description 服务器监控：CPU 使用率/平均负载、内存、Go 运行时、主机与磁盘信息
// @Tags 服务监控
// @Success 0 {object} response.Response{data=domain.ServerInfo} "{"code": 0, "msg": "success","data": { "cpu": {}, "mem": {}, "go": {}, "sys": {}, "sysFiles": [] } }"
// @Router /monitor/server [get]
// @Security
func (u *Server) Info(c *gin.Context) {
	response.OK(server.Info(), c)
}
