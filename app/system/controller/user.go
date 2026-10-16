package controller

import (
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/model/request"
	resp "github.com/bluefroog/admintpl/common/model/response"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

type User struct{}

var (
	user = service.User{}
)

// @Summary 获取用户列表
// @Description 用户列表
// @Tags 用户管理
// @Param data query domain.UserSearchRequest true "data"
// @Success 0 {object} response.Response{data=response.PageResult{list=[]domain.User}} "{"code": 0, "msg": "Success","data": { "list": [],"total": 0 } }"
// @Router /system/user/list [get]
// @Security
func (u *User) List(c *gin.Context) {
	var searchParams domain.UserSearchRequest
	// 获取查询参数
	if err := c.Bind(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
	}
	log.Printf("searchParam: %#v", searchParams)
	// 验证参数
	if err := validate.Struct(&searchParams); err != nil {
		response.Fail(-1, err.Error(), c)
	}
	err, list, total := user.GetList(&searchParams)
	if err != nil {
		response.Fail(-1, err.Error(), c)
	}
	response.OK(resp.PageResult{
		List:     list,
		Total:    total,
		Page:     searchParams.Page,
		PageSize: searchParams.PageSize,
	}, c)
}

// @Summary 添加用户
// @Description 添加用户
// @Tags 用户管理
// @Param data body domain.UserAddRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"Success","data": null}"
// @Router /system/user/create [post]
// @Security
func (u *User) Create(c *gin.Context) {
	var data *domain.UserAddRequest
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
	if err := user.Create(data); err != nil {
		response.Fail(-1, "添加失败: "+err.Error(), c)
	} else {
		response.Success("添加成功", c)
	}
}

// @Summary 修改用户
// @Description 修改用户
// @Tags 用户管理
// @Param data body domain.UserEditRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"Success","data": null}"
// @Router /system/user/update [put]
// @Security
func (u *User) Update(c *gin.Context) {
	var data *domain.UserEditRequest
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
	if err := user.Update(data); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
	} else {
		response.Success("修改成功", c)
	}
}

// @Summary 删除用户
// @Description 删除用户
// @Tags 用户管理
// @Param ids body request.IdsRequest true "{ids: [1,2']}"
// @Success 0 {object} response.Response "{"code": 0, "msg":"Success","data": null}"
// @Router /system/user/delete [delete]
// @Security
func (u *User) Delete(c *gin.Context) {
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
	err := user.Delete(data.Ids)
	if err != nil {
		response.Fail(-1, "删除失败: "+err.Error(), c)
		return
	}
	response.Success("删除成功", c)
}

// @Summary 获取指定ID的用户详情
// @Description 获取指定ID的用户详情
// @Tags 用户管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=domain.User} "{"code": 0, "msg":"Success","data": {}}"
// @Router /system/user/detail [get]
// @Security
func (u *User) Detail(c *gin.Context) {
	var param *request.GetById

	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	record, err := user.Detail(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(record, c)
}

// @Summary 重置用户密码
// @Description 重置指定用户密码
// @Tags 用户管理
// @Param data body domain.UserResetPwdRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/reset-pwd [put]
// @Security
func (u *User) ResetPwd(c *gin.Context) {
	var param domain.UserResetPwdRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.ResetPwd(&param); err != nil {
		response.Fail(-1, "重置失败: "+err.Error(), c)
		return
	}
	response.Success("重置成功", c)
}

// @Summary 修改用户状态
// @Description 修改用户状态（0:禁用,1:正常,2:未验证）
// @Tags 用户管理
// @Param data body domain.UserChangeStatusRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/change-status [put]
// @Security
func (u *User) ChangeStatus(c *gin.Context) {
	var param domain.UserChangeStatusRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.ChangeStatus(&param); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}

// @Summary 分配用户角色
// @Description 给用户分配角色
// @Tags 用户管理
// @Param data body domain.UserAssignRoleRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/roles [put]
// @Security
func (u *User) AssignRoles(c *gin.Context) {
	var param domain.UserAssignRoleRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.AssignRoles(&param); err != nil {
		response.Fail(-1, "授权失败: "+err.Error(), c)
		return
	}
	response.Success("授权成功", c)
}

// @Summary 获取用户已分配角色
// @Description 获取用户已分配的角色ID列表
// @Tags 用户管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=[]uint} "{"code": 0, "msg":"success","data": []}"
// @Router /system/user/roles [get]
// @Security
func (u *User) GetRoles(c *gin.Context) {
	var param request.GetById
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, roleIds := user.GetUserRoles(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(roleIds, c)
}

// @Summary 分配用户岗位
// @Description 给用户分配岗位
// @Tags 用户管理
// @Param data body domain.UserAssignPostRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/posts [put]
// @Security
func (u *User) AssignPosts(c *gin.Context) {
	var param domain.UserAssignPostRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.AssignPosts(&param); err != nil {
		response.Fail(-1, "授权失败: "+err.Error(), c)
		return
	}
	response.Success("授权成功", c)
}

// @Summary 获取用户已分配岗位
// @Description 获取用户已分配的岗位ID列表
// @Tags 用户管理
// @Param data query request.GetById true "data"
// @Success 0 {object} response.Response{data=[]uint} "{"code": 0, "msg":"success","data": []}"
// @Router /system/user/posts [get]
// @Security
func (u *User) GetPosts(c *gin.Context) {
	var param request.GetById
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, postIds := user.GetUserPosts(param.ID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(postIds, c)
}

// @Summary 校验用户名是否唯一
// @Description 校验用户名是否唯一
// @Tags 用户管理
// @Param data query domain.UserCheckUniqueRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/user/check-username [get]
// @Security
func (u *User) CheckUsernameUnique(c *gin.Context) {
	var param domain.UserCheckUniqueRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := user.CheckUniqueUsername(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}

// @Summary 校验手机号是否唯一
// @Description 校验手机号是否唯一
// @Tags 用户管理
// @Param data query domain.UserCheckUniqueRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/user/check-mobile [get]
// @Security
func (u *User) CheckMobileUnique(c *gin.Context) {
	var param domain.UserCheckUniqueRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := user.CheckUniqueMobile(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}

// @Summary 校验邮箱是否唯一
// @Description 校验邮箱是否唯一
// @Tags 用户管理
// @Param data query domain.UserCheckUniqueRequest true "data"
// @Success 0 {object} response.Response{data=bool} "{"code": 0, "msg":"success","data": true}"
// @Router /system/user/check-email [get]
// @Security
func (u *User) CheckEmailUnique(c *gin.Context) {
	var param domain.UserCheckUniqueRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	err, unique := user.CheckUniqueEmail(&param)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(unique, c)
}

// @Summary 修改个人资料
// @Description 修改用户个人资料
// @Tags 用户管理
// @Param data body domain.UserProfileUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/profile [put]
// @Security
func (u *User) UpdateProfile(c *gin.Context) {
	var param domain.UserProfileUpdateRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.UpdateProfile(&param); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}

// @Summary 修改密码
// @Description 用户修改密码（校验旧密码）
// @Tags 用户管理
// @Param data body domain.UserPasswordUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/password [put]
// @Security
func (u *User) UpdatePassword(c *gin.Context) {
	var param domain.UserPasswordUpdateRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.UpdatePassword(&param); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}

// @Summary 修改头像
// @Description 修改用户头像
// @Tags 用户管理
// @Param data body domain.UserAvatarUpdateRequest true "data"
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /system/user/avatar [put]
// @Security
func (u *User) UpdateAvatar(c *gin.Context) {
	var param domain.UserAvatarUpdateRequest
	if err := c.Bind(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&param); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := user.UpdateAvatar(&param); err != nil {
		response.Fail(-1, "修改失败: "+err.Error(), c)
		return
	}
	response.Success("修改成功", c)
}
