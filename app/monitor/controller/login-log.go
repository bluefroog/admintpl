package controller

import (
	"github.com/bluefroog/admintpl/app/monitor/domain"
	"github.com/bluefroog/admintpl/app/monitor/service"
	"github.com/bluefroog/admintpl/common/model/request"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
	"github.com/gin-gonic/gin"
)

type LoginLog struct{}

var (
	loginLog = service.LoginLog{}
)

// List 获取访问日志列表
// @Summary 获取访问日志列表
// @Description 访问日志列表
// @Tags 访问日志管理
// @Param data query domain.LoginLogSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.LoginLog}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /monitor/logininfor/list [get]
// @Security
func (u *LoginLog) List(c *gin.Context) {
	var searchParams domain.LoginLogSearchRequest
	// 获取查询参数
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	// 验证参数
	if err := validate.Struct(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, list, total := loginLog.GetList(&searchParams)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(resp.PageResult{
		List:     list,
		Total:    total,
		Page:     searchParams.Page,
		PageSize: searchParams.PageSize,
	}, c)
}

// Delete 删除访问日志
// @Summary 删除访问日志
// @Description 删除访问日志
// @Tags 访问日志管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/logininfor/delete [delete]
// @Security
func (u *LoginLog) Delete(c *gin.Context) {
	var data *request.IdsRequest
	// 获取参数
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if len(data.Ids) == 0 {
		response.Fail(-1, "参数不能为空", c)
		return
	}
	err := loginLog.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// Detail 获取指定ID的访问日志详情
// @Summary 获取指定ID的访问日志详情
// @Description 获取指定ID的访问日志详情
// @Tags 访问日志管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.LoginLog} "{"code": 0, "msg":"success","data": {}}"
// @Router /monitor/logininfor/detail [get]
// @Security
func (u *LoginLog) Detail(c *gin.Context) {
	var param request.GetById
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	// 验证参数
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	record, err := loginLog.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}

// @Summary 清空登录日志
// @Description 清空登录日志，status 为空时清空全部
// @Tags 登录日志
// @Param status query uint false "登录状态（0成功 1失败）"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/logininfor/clear [delete]
// @Security
func (u *LoginLog) Clear(c *gin.Context) {
	var param struct {
		Status *uint `form:"status"`
	}
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := loginLog.Clear(param.Status); err != nil {
		response.Fail(-1, "清空失败: "+err.Error(), c)
		return
	}
	response.Success("清空成功", c)
}
