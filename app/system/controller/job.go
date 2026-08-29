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

type Job struct{}

var (
	job = service.Job{}
)

// List 获取定时任务列表
// @Summary 获取定时任务列表
// @Description 定时任务列表
// @Tags 定时任务管理
// @Param data query domain.JobSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.Job}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/job/list [get]
// @Security
func (u *Job) List(c *gin.Context) {
	var searchParams domain.JobSearchRequest
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
	err, list, total := job.GetList(&searchParams)
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

// Create 添加定时任务
// @Summary 添加定时任务
// @Description 添加定时任务
// @Tags 定时任务管理
// @Param data body domain.JobCreateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/job/create [post]
// @Security
func (u *Job) Create(c *gin.Context) {
	var data *domain.JobCreateRequest
	// 获取参数
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	// 验证参数
	if err := validate.Struct(data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := job.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// Update 修改定时任务
// @Summary 修改定时任务
// @Description 修改定时任务
// @Tags 定时任务管理
// @Param data body domain.JobUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/job/update [put]
// @Security
func (u *Job) Update(c *gin.Context) {
	var data *domain.JobUpdateRequest
	// 获取参数
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	// 参数验证
	if err := validate.Struct(data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := job.Update(data); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
	} else {
		response.Success("修改成功", c)
	}
}

// Delete 删除定时任务
// @Summary 删除定时任务
// @Description 删除定时任务
// @Tags 定时任务管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/job/delete [delete]
// @Security
func (u *Job) Delete(c *gin.Context) {
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
	err := job.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// Detail 获取指定ID的定时任务详情
// @Summary 获取指定ID的定时任务详情
// @Description 获取指定ID的定时任务详情
// @Tags 定时任务管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.Job} "{"code": 0, "msg":"success","data": {}}"
// @Router /system/job/detail [get]
// @Security
func (u *Job) Detail(c *gin.Context) {
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
	record, err := job.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}
