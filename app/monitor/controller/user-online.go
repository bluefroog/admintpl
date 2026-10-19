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

type UserOnline struct{}

var (
	userOnline = service.UserOnline{}
)

// @Summary 获取在线用户列表
// @Description 在线用户列表
// @Tags 在线用户
// @Param data query domain.UserOnlineSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.UserOnline}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /monitor/online/list [get]
// @Security
func (u *UserOnline) List(c *gin.Context) {
	var searchParams domain.UserOnlineSearchRequest
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, list, total := userOnline.GetList(&searchParams)
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

// @Summary 删除在线用户
// @Description 删除在线用户记录
// @Tags 在线用户
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/online/delete [delete]
// @Security
func (u *UserOnline) Delete(c *gin.Context) {
	var data *request.IdsRequest
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if len(data.Ids) == 0 {
		response.Fail(-1, "参数不能为空", c)
		return
	}
	if err := userOnline.Delete(data.Ids); err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// @Summary 获取指定ID的在线用户详情
// @Description 获取指定ID的在线用户详情
// @Tags 在线用户
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.UserOnline} "{"code": 0, "msg":"success","data": {}}"
// @Router /monitor/online/detail [get]
// @Security
func (u *UserOnline) Detail(c *gin.Context) {
	var param request.GetById
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	record, err := userOnline.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}

// @Summary 强退在线用户
// @Description 根据用户token强退在线用户
// @Tags 在线用户
// @Param data body domain.UserOnlineForceLogoutRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/online/force-logout [delete]
// @Security
func (u *UserOnline) ForceLogout(c *gin.Context) {
	var param domain.UserOnlineForceLogoutRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := userOnline.ForceLogout(&param); err != nil {
		response.Fail(-1, "强退失败: "+err.Error(), c)
		return
	}
	response.Success("强退成功", c)
}
