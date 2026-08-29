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

type DictType struct{}

var (
	dictType = service.DictType{}
)

// List 获取字典类型列表
// @Summary 获取字典类型列表
// @Description 字典类型列表
// @Tags 字典类型管理
// @Param data query domain.DictTypeSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.DictType}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /system/dict-type/list [get]
// @Security
func (u *DictType) List(c *gin.Context) {
	var searchParams domain.DictTypeSearchRequest
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
	err, list, total := dictType.GetList(&searchParams)
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

// Create 添加字典类型
// @Summary 添加字典类型
// @Description 添加字典类型
// @Tags 字典类型管理
// @Param data body domain.DictTypeCreateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/dict-type/create [post]
// @Security
func (u *DictType) Create(c *gin.Context) {
	var data *domain.DictTypeCreateRequest
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
	if err := dictType.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// Update 修改字典类型
// @Summary 修改字典类型
// @Description 修改字典类型
// @Tags 字典类型管理
// @Param data body domain.DictTypeUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/dict-type/update [put]
// @Security
func (u *DictType) Update(c *gin.Context) {
	var data *domain.DictTypeUpdateRequest
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
	if err := dictType.Update(data); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
	} else {
		response.Success("修改成功", c)
	}
}

// Delete 删除字典类型
// @Summary 删除字典类型
// @Description 删除字典类型
// @Tags 字典类型管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/dict-type/delete [delete]
// @Security
func (u *DictType) Delete(c *gin.Context) {
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
	err := dictType.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// Detail 获取指定ID的字典类型详情
// @Summary 获取指定ID的字典类型详情
// @Description 获取指定ID的字典类型详情
// @Tags 字典类型管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.DictType} "{"code": 0, "msg":"success","data": {}}"
// @Router /system/dict-type/detail [get]
// @Security
func (u *DictType) Detail(c *gin.Context) {
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
	record, err := dictType.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}
