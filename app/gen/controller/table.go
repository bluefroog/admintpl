package controller

import (
	"github.com/bluefroog/admintpl/app/gen/domain"
	"github.com/bluefroog/admintpl/app/gen/service"
	"github.com/bluefroog/admintpl/app/system/middleware"
	"github.com/bluefroog/admintpl/common/model/request"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
	"github.com/gin-gonic/gin"
)

type Gen struct{}

var (
	genTable = service.Table{}
)

// List 获取代码生成表列表
// @Summary 获取代码生成表列表
// @Description 已导入 gen_table 的代码生成业务表列表
// @Tags 代码生成管理
// @Param data query domain.TableSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.Table}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /tool/gen/list [get]
// @Security
func (g *Gen) List(c *gin.Context) {
	var searchParams domain.TableSearchRequest
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	normalizePage(&searchParams.PageInfo)
	err, list, total := genTable.GetList(&searchParams)
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

// DbList 获取数据库表列表（未导入代码生成的表）
// @Summary 获取数据库表列表
// @Description 查询 information_schema.tables，返回尚未导入代码生成的数据库表
// @Tags 代码生成管理
// @Param data query domain.DbTableSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.DbTable}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /tool/gen/db-list [get]
// @Security
func (g *Gen) DbList(c *gin.Context) {
	var searchParams domain.DbTableSearchRequest
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	normalizePage(&searchParams.PageInfo)
	err, list, total := genTable.DbTableList(&searchParams)
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

// ColumnList 获取代码生成表字段列表
// @Summary 获取代码生成表字段列表
// @Description 获取指定代码生成表的字段配置列表
// @Tags 代码生成管理
// @Param data query domain.TableColumnSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.TableColumn}} "{"code": 0, "msg": "success","data": { "list": [],"total": 0 } }"
// @Router /tool/gen/column-list [get]
// @Security
func (g *Gen) ColumnList(c *gin.Context) {
	var searchParams domain.TableColumnSearchRequest
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if searchParams.TableID == nil || *searchParams.TableID == 0 {
		response.Fail(-1, "表编号不能为空", c)
		return
	}
	normalizePage(&searchParams.PageInfo)
	err, list := genTable.Columns(*searchParams.TableID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(resp.PageResult{
		List:     list,
		Total:    int64(len(list)),
		Page:     searchParams.Page,
		PageSize: searchParams.PageSize,
	}, c)
}

// Detail 获取代码生成表详情
// @Summary 获取代码生成表详情
// @Description 获取代码生成表详情，包含字段配置、主键列与树表参数
// @Tags 代码生成管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.Table} "{"code": 0, "msg": "success","data": {} }"
// @Router /tool/gen/detail [get]
// @Security
func (g *Gen) Detail(c *gin.Context) {
	var data request.GetById
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	detail, err := genTable.Detail(data.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(detail, c)
}

// Import 导入表结构
// @Summary 导入表结构
// @Description 按表名导入数据库表结构到代码生成业务表（已导入的表自动跳过）
// @Tags 代码生成管理
// @Param data body domain.TableImportRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /tool/gen/import [post]
// @Security
func (g *Gen) Import(c *gin.Context) {
	var data domain.TableImportRequest
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	imported, err := genTable.Import(data.Tables, middleware.GetUsername(c))
	if err != nil {
		response.Fail(-1, "导入失败: "+err.Error(), c)
		return
	}
	response.OK(gin.H{"imported": imported}, c)
}

// Update 修改代码生成业务
// @Summary 修改代码生成业务
// @Description 修改代码生成业务配置（表信息 + 字段配置）
// @Tags 代码生成管理
// @Param data body domain.TableUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /tool/gen/update [put]
// @Security
func (g *Gen) Update(c *gin.Context) {
	var data *domain.TableUpdateRequest
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := genTable.Update(data, middleware.GetUsername(c)); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}

// SynchDb 同步数据库表结构
// @Summary 同步数据库表结构
// @Description 按数据库最新结构同步代码生成业务表（新增列 / 删除已移除列 / 列类型变化重置列配置）
// @Tags 代码生成管理
// @Param data body domain.TableSynchRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /tool/gen/synch-db [post]
// @Security
func (g *Gen) SynchDb(c *gin.Context) {
	var data domain.TableSynchRequest
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := genTable.SynchDb(data.ID, middleware.GetUsername(c)); err != nil {
		response.Fail(-1, "同步失败: "+err.Error(), c)
		return
	}
	response.Success("同步成功", c)
}

// Delete 删除代码生成业务
// @Summary 删除代码生成业务
// @Description 删除代码生成业务，同时清理其字段配置
// @Tags 代码生成管理
// @Param data body request.IdsRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /tool/gen/delete [delete]
// @Security
func (g *Gen) Delete(c *gin.Context) {
	var data request.IdsRequest
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := genTable.Delete(data.Ids); err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// normalizePage 归一化分页参数（页码最小 1，每页最小 1 条，未传时默认 20 条）
// @param: page *request.PageInfo 分页参数
func normalizePage(page *request.PageInfo) {
	if page.Page <= 0 {
		page.Page = 1
	}
	if page.PageSize <= 0 {
		page.PageSize = 20
	}
}
