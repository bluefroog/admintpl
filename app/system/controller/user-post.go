package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
)

type UserPost struct{}

var (
	userPost = service.UserPost{}
)

// List 获取用户岗位关联列表
// @Summary 获取用户岗位关联列表
// @Description 用户岗位关联列表
// @Tags 用户岗位关联管理
// @Param data query domain.UserPostSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.UserPost}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/user-post/list [get]
// @Security
func (u *UserPost) List(c *gin.Context) {
	var searchParams domain.UserPostSearchRequest
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
	err, list, total := userPost.GetList(&searchParams)
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

// Create 添加用户岗位关联
// @Summary 添加用户岗位关联
// @Description 添加用户岗位关联
// @Tags 用户岗位关联管理
// @Param data body domain.UserPostCreateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user-post/create [post]
// @Security
func (u *UserPost) Create(c *gin.Context) {
	var data *domain.UserPostCreateRequest
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
	if err := userPost.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// Delete 删除用户岗位关联
// @Summary 删除用户岗位关联
// @Description 删除用户岗位关联
// @Tags 用户岗位关联管理
// @Param data body domain.UserPostDeleteRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user-post/delete [delete]
// @Security
func (u *UserPost) Delete(c *gin.Context) {
	var data *domain.UserPostDeleteRequest
	// 获取参数
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if data.UserID == nil && data.PostID == nil {
		response.Fail(-1, "参数不能为空", c)
		return
	}
	err := userPost.Delete(data)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}
