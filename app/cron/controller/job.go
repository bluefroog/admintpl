package controller

import (
	"github.com/bluefroog/admintpl/app/cron/domain"
	"github.com/bluefroog/admintpl/app/cron/service"
	"github.com/bluefroog/admintpl/common/model/request"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
	"github.com/gin-gonic/gin"
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
// @Router /monitor/job/list [get]
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
// @Router /monitor/job/create [post]
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
// @Router /monitor/job/update [put]
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
// @Router /monitor/job/delete [delete]
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
// @Router /monitor/job/detail [get]
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

// @Summary 修改任务状态
// @Description 修改定时任务状态（0正常 1暂停）
// @Tags 定时任务
// @Param data body domain.JobChangeStatusRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/job/change-status [put]
// @Security
func (u *Job) ChangeStatus(c *gin.Context) {
	var param domain.JobChangeStatusRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := job.ChangeStatus(&param); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}

// Run 立即执行一次定时任务
// @Summary 立即执行一次定时任务
// @Description 立即执行一次定时任务（状态为暂停的任务同样可以执行），执行结果写入定时任务日志
// @Tags 定时任务管理
// @Param data body domain.JobRunRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /monitor/job/run [put]
// @Security
func (u *Job) Run(c *gin.Context) {
	var param domain.JobRunRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := job.Run(param.ID); err != nil {
		response.Fail(-1, "执行失败: "+err.Error(), c)
		return
	}
	response.Success("执行成功", c)
}

// @Summary 校验任务名称是否唯一
// @Description 校验任务名称是否唯一
// @Tags 定时任务
// @Param data query domain.JobCheckNameRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /monitor/job/check-name [get]
// @Security
func (u *Job) CheckNameUnique(c *gin.Context) {
	var param domain.JobCheckNameRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	var excludeIds *[]uint
	if param.ID != nil {
		ids := []uint{*param.ID}
		excludeIds = &ids
	}
	err, unique := job.CheckNameUnique(param.Name, excludeIds)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}
