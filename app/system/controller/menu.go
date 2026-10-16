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

type Menu struct{}

var (
	menu = service.Menu{}
)

// List 获取菜单列表
// @Summary 获取菜单列表
// @Description 菜单列表
// @Tags 菜单管理
// @Param data query domain.MenuSearchRequest true "data"
// @Success 0 {response.PageResult} response.Response{data=response.PageResult{list=[]domain.Menu}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/menu/list [get]
// @Security
func (u *Menu) List(c *gin.Context) {
	var searchParams domain.MenuSearchRequest
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
	err, list, total := menu.GetList(&searchParams)
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

// Create 添加菜单
// @Summary 添加菜单
// @Description 添加菜单
// @Tags 菜单管理
// @Param data body domain.MenuCreateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/menu/create [post]
// @Security
func (u *Menu) Create(c *gin.Context) {
	var data *domain.MenuCreateRequest
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
	if err := menu.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// Update 修改菜单
// @Summary 修改菜单
// @Description 修改菜单
// @Tags 菜单管理
// @Param data body domain.MenuUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/menu/update [put]
// @Security
func (u *Menu) Update(c *gin.Context) {
	var data *domain.MenuUpdateRequest
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
	if err := menu.Update(data); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
	} else {
		response.Success("修改成功", c)
	}
}

// Delete 删除菜单
// @Summary 删除菜单
// @Description 删除菜单
// @Tags 菜单管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/menu/delete [delete]
// @Security
func (u *Menu) Delete(c *gin.Context) {
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
	// 如果有子菜单存在，禁止删除,需要先删除完子菜单
	_, _, total := menu.GetList(&domain.MenuSearchRequest{
		ParentIDIn: data.Ids,
		PageInfo:   request.PageInfo{PageSize: 1},
	})
	if total > 0 {
		response.Fail(-1, "存在子菜单，禁止删除", c)
		return
	}
	err := menu.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// Detail 获取指定ID的菜单详情
// @Summary 获取指定ID的菜单详情
// @Description 获取指定ID的菜单详情
// @Tags 菜单管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.Menu} "{"code": 0, "msg":"success","data": {}}"
// @Router /system/menu/detail [get]
// @Security
func (u *Menu) Detail(c *gin.Context) {
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
	record, err := menu.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}

// @Summary 获取菜单树
// @Description 获取菜单树数据（返回扁平列表，由前端组装成树）
// @Tags 菜单管理
// @Param data query domain.MenuTreeRequest true "data"
// @Success 0 {object} response.Response{data=[]domain.Ztree} "{"code": 0, "msg":"success","data": []}"
// @Router /system/menu/tree [get]
// @Security
func (u *Menu) TreeData(c *gin.Context) {
	var param domain.MenuTreeRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, list := menu.TreeData(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(list, c)
}

// @Summary 获取角色菜单树
// @Description 根据角色ID获取菜单树（勾选已授权菜单）
// @Tags 菜单管理
// @Param data query domain.RoleMenuTreeRequest true "data"
// @Success 0 {object} response.Response{data=[]domain.Ztree} "{"code": 0, "msg":"success","data": []}"
// @Router /system/menu/role-menu-tree [get]
// @Security
func (u *Menu) RoleMenuTree(c *gin.Context) {
	var param domain.RoleMenuTreeRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, list := menu.RoleMenuTreeData(param.RoleID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(list, c)
}

// @Summary 校验菜单名称是否唯一
// @Description 校验菜单名称是否唯一
// @Tags 菜单管理
// @Param data query domain.MenuCheckNameRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/menu/check-name [get]
// @Security
func (u *Menu) CheckNameUnique(c *gin.Context) {
	var param domain.MenuCheckNameRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := menu.CheckNameUnique(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}

// @Summary 校验权限标识是否唯一
// @Description 校验权限标识是否唯一
// @Tags 菜单管理
// @Param data query domain.MenuCheckPermsRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/menu/check-perms [get]
// @Security
func (u *Menu) CheckPermsUnique(c *gin.Context) {
	var param domain.MenuCheckPermsRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := menu.CheckPermsUnique(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}
