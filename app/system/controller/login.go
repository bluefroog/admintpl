package controller

import (
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/middleware"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/bluefroog/admintpl/common/validate"
	"github.com/gin-gonic/gin"
)

// Login 登录认证控制器
type Login struct{}

var (
	auth = service.Auth{}
)

// Captcha 获取图形验证码
// @Summary 获取图形验证码
// @Description 获取图形验证码（system.captcha.enabled=false 时返回 captchaEnabled=false）
// @Tags 登录认证
// @Success 0 {object} response.Response{data=domain.CaptchaResponse} "{"code": 0, "msg":"success","data": {"captchaEnabled": true, "uuid": "", "img": ""}}"
// @Router /captchaImage [get]
// @Security
func (l *Login) Captcha(c *gin.Context) {
	data, err := auth.Captcha()
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(data, c)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录，成功后返回访问令牌
// @Tags 登录认证
// @Param data body domain.LoginRequest true "data"
// @Success 0 {object} response.Response{data=domain.LoginResponse} "{"code": 0, "msg":"success","data": {"token": "", "expiresIn": 720}}"
// @Router /login [post]
// @Security
func (l *Login) Login(c *gin.Context) {
	var data domain.LoginRequest
	if err := c.Bind(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	if err := validate.Struct(&data); err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	resp, err := auth.SignIn(&data, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Fail(401, err.Error(), c)
		return
	}
	response.OK(resp, c)
}

// Logout 退出登录
// @Summary 退出登录
// @Description 退出登录，删除当前令牌对应的在线用户记录
// @Tags 登录认证
// @Success 0 {object} response.Response "{"code": 0, "msg":"success","data": null}"
// @Router /logout [post]
// @Security
func (l *Login) Logout(c *gin.Context) {
	token := middleware.GetToken(c)
	if token == "" {
		token = middleware.Token(c)
	}
	if token != "" {
		if err := auth.SignOut(token); err != nil {
			response.Fail(-1, "退出失败: "+err.Error(), c)
			return
		}
	}
	response.Success("退出成功", c)
}

// GetInfo 获取登录用户信息
// @Summary 获取登录用户信息
// @Description 获取当前登录用户的资料、角色标识与权限标识
// @Tags 登录认证
// @Param userId query uint false "用户ID（未携带令牌时用于调试）"
// @Success 0 {object} response.Response{data=domain.UserInfoResponse} "{"code": 0, "msg":"success","data": {}}"
// @Router /getInfo [get]
// @Security
func (l *Login) GetInfo(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		response.Fail(401, "认证失败，无法访问系统资源", c)
		return
	}
	data, err := auth.GetUserInfo(userID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(data, c)
}

// GetRouters 获取登录用户的前端路由
// @Summary 获取登录用户的前端路由
// @Description 获取当前登录用户的菜单路由树
// @Tags 登录认证
// @Param userId query uint false "用户ID（未携带令牌时用于调试）"
// @Success 0 {object} response.Response{data=[]domain.RouterResponse} "{"code": 0, "msg":"success","data": []}"
// @Router /getRouters [get]
// @Security
func (l *Login) GetRouters(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		response.Fail(401, "认证失败，无法访问系统资源", c)
		return
	}
	list, err := auth.GetRouters(userID)
	if err != nil {
		response.Fail(-1, err.Error(), c)
		return
	}
	response.OK(list, c)
}

// currentUserID 获取当前登录用户ID（未认证时允许通过 userId 参数指定，便于关闭认证时调试）
// @param: c *gin.Context
// @return: uint
func currentUserID(c *gin.Context) uint {
	if userID := middleware.GetUserID(c); userID > 0 {
		return userID
	}
	var param domain.UserIdRequest
	if err := c.Bind(&param); err != nil {
		return 0
	}
	return param.UserId
}
