package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/model/request"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
)

type JobLog struct{}

var (
	jobLog = service.JobLog{}
)

// List 获取定时任务日志列表
// @Summary 获取定时任务日志列表
// @Description 定时任务日志列表
// @Tags 定时任务日志管理
// @Param data query domain.JobLogSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.JobLog}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/job-log/list [get]
// @Security
func (u *JobLog) List(c *gin.Context) {
	var searchParams domain.JobLogSearchRequest
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
	err, list, total := jobLog.GetList(&searchParams)
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

// Delete 删除定时任务日志
// @Summary 删除定时任务日志
// @Description 删除定时任务日志
// @Tags 定时任务日志管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/job-log/delete [delete]
// @Security
func (u *JobLog) Delete(c *gin.Context) {
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
	err := jobLog.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// Detail 获取指定ID的定时任务日志详情
// @Summary 获取指定ID的定时任务日志详情
// @Description 获取指定ID的定时任务日志详情
// @Tags 定时任务日志管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.JobLog} "{"code": 0, "msg":"success","data": {}}"
// @Router /system/job-log/detail [get]
// @Security
func (u *JobLog) Detail(c *gin.Context) {
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
	record, err := jobLog.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}
