package controller

import (
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/model/request"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
	"github.com/gin-gonic/gin"
)

type Role struct{}

var (
	role = service.Role{}
)

// List 获取角色列表
// @Summary 获取角色列表
// @Description 角色列表
// @Tags 角色管理
// @Param data query domain.RoleSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.Role}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/role/list [get]
// @Security
func (u *Role) List(c *gin.Context) {
	var searchParams domain.RoleSearchRequest
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
	err, list, total := role.GetList(&searchParams)
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

// Create 添加角色
// @Summary 添加角色
// @Description 添加角色
// @Tags 角色管理
// @Param data body domain.RoleCreateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/create [post]
// @Security
func (u *Role) Create(c *gin.Context) {
	var data *domain.RoleCreateRequest
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
	if err := role.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// Update 修改角色
// @Summary 修改角色
// @Description 修改角色
// @Tags 角色管理
// @Param data body domain.RoleUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/update [put]
// @Security
func (u *Role) Update(c *gin.Context) {
	var data *domain.RoleUpdateRequest
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
	if err := role.Update(data); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
	} else {
		response.Success("修改成功", c)
	}
}

// Delete 删除角色
// @Summary 删除角色
// @Description 删除角色
// @Tags 角色管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/delete [delete]
// @Security
func (u *Role) Delete(c *gin.Context) {
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
	err := role.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// Detail 获取指定ID的角色详情
// @Summary 获取指定ID的角色详情
// @Description 获取指定ID的角色详情
// @Tags 角色管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.Role} "{"code": 0, "msg":"success","data": {}}"
// @Router /system/role/detail [get]
// @Security
func (u *Role) Detail(c *gin.Context) {
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
	record, err := role.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}

// @Summary 角色菜单授权
// @Description 给角色分配菜单权限
// @Tags 角色管理
// @Param data body domain.RoleMenuAssignRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/menu [put]
// @Security
func (u *Role) AssignMenus(c *gin.Context) {
	var param domain.RoleMenuAssignRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := role.AssignMenus(&param); err != nil {
		response.Fail(-1, "授权失败: "+err.Error(), c)
		return
	}
	response.Success("授权成功", c)
}

// @Summary 修改角色数据权限
// @Description 修改角色数据权限（数据范围 + 自定义部门）
// @Tags 角色管理
// @Param data body domain.RoleDataScopeRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/data-scope [put]
// @Security
func (u *Role) DataScope(c *gin.Context) {
	var param domain.RoleDataScopeRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := role.AuthDataScope(&param); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}

// @Summary 查询角色已分配用户
// @Description 查询角色已分配的用户列表
// @Tags 角色管理
// @Param data query domain.RoleUserListRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.User}} "{"code": 0, "msg":"success","data": { "list": [],"total": 0 } }"
// @Router /system/role/allocated-list [get]
// @Security
func (u *Role) AllocatedList(c *gin.Context) {
	var param domain.RoleUserListRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, list, total := role.AllocatedList(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(resp.PageResult{List: list, Total: total, Page: param.Page, PageSize: param.PageSize}, c)
}

// @Summary 查询角色未分配用户
// @Description 查询角色未分配的用户列表
// @Tags 角色管理
// @Param data query domain.RoleUserListRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.User}} "{"code": 0, "msg":"success","data": { "list": [],"total": 0 } }"
// @Router /system/role/unallocated-list [get]
// @Security
func (u *Role) UnallocatedList(c *gin.Context) {
	var param domain.RoleUserListRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, list, total := role.UnallocatedList(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(resp.PageResult{List: list, Total: total, Page: param.Page, PageSize: param.PageSize}, c)
}

// @Summary 批量选择用户授权
// @Description 批量选择用户授予角色
// @Tags 角色管理
// @Param data body domain.RoleUserAssignRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/select-users [post]
// @Security
func (u *Role) SelectAllUsers(c *gin.Context) {
	var param domain.RoleUserAssignRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := role.SelectAllUsers(&param); err != nil {
		response.Fail(-1, "授权失败: "+err.Error(), c)
		return
	}
	response.Success("授权成功", c)
}

// @Summary 取消用户授权
// @Description 取消单个用户的角色授权
// @Tags 角色管理
// @Param data body domain.RoleUserCancelRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/cancel-user [delete]
// @Security
func (u *Role) CancelUser(c *gin.Context) {
	var param domain.RoleUserCancelRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := role.CancelUser(&param); err != nil {
		response.Fail(-1, "取消失败: "+err.Error(), c)
		return
	}
	response.Success("取消成功", c)
}

// @Summary 批量取消用户授权
// @Description 批量取消用户的角色授权
// @Tags 角色管理
// @Param data body domain.RoleUserCancelAllRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/role/cancel-users [delete]
// @Security
func (u *Role) CancelAllUsers(c *gin.Context) {
	var param domain.RoleUserCancelAllRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := role.CancelAllUsers(&param); err != nil {
		response.Fail(-1, "取消失败: "+err.Error(), c)
		return
	}
	response.Success("取消成功", c)
}

// @Summary 校验角色名称是否唯一
// @Description 校验角色名称是否唯一
// @Tags 角色管理
// @Param data query domain.RoleCheckNameRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/role/check-name [get]
// @Security
func (u *Role) CheckNameUnique(c *gin.Context) {
	var param domain.RoleCheckNameRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := role.CheckNameUnique(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}

// @Summary 校验角色权限字符串是否唯一
// @Description 校验角色权限字符串是否唯一
// @Tags 角色管理
// @Param data query domain.RoleCheckKeyRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/role/check-key [get]
// @Security
func (u *Role) CheckKeyUnique(c *gin.Context) {
	var param domain.RoleCheckKeyRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := role.CheckKeyUnique(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}
