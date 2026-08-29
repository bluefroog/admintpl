package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
)

type RoleDept struct{}

var (
	roleDept = service.RoleDept{}
)

// List 获取角色部门关联列表
// @Summary 获取角色部门关联列表
// @Description 角色部门关联列表
// @Tags 角色部门关联管理
// @Param data query domain.RoleDeptSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.RoleDept}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/role-dept/list [get]
// @Security
func (u *RoleDept) List(c *gin.Context) {
	var searchParams domain.RoleDeptSearchRequest
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
	err, list, total := roleDept.GetList(&searchParams)
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

// Create 添加角色部门关联
// @Summary 添加角色部门关联
// @Description 添加角色部门关联
// @Tags 角色部门关联管理
// @Param data body domain.RoleDeptCreateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role-dept/create [post]
// @Security
func (u *RoleDept) Create(c *gin.Context) {
	var data *domain.RoleDeptCreateRequest
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
	if err := roleDept.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// Delete 删除角色部门关联
// @Summary 删除角色部门关联
// @Description 删除角色部门关联
// @Tags 角色部门关联管理
// @Param data body domain.RoleDeptDeleteRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role-dept/delete [delete]
// @Security
func (u *RoleDept) Delete(c *gin.Context) {
	var data *domain.RoleDeptDeleteRequest
	// 获取参数
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if data.RoleID == nil && data.DeptID == nil {
		response.Fail(-1, "参数不能为空", c)
		return
	}
	err := roleDept.Delete(data)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}
