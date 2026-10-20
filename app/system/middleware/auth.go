package middleware

import (
	"net/http"
	"strings"

	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/response"
	"github.com/gin-gonic/gin"
)

const (
	// ContextLoginUser 登录用户在 gin.Context 中的键
	ContextLoginUser = "loginUser"
	// ContextToken 登录令牌在 gin.Context 中的键
	ContextToken = "token"
)

// selfServicePaths 登录用户自助接口，不参与权限校验
var selfServicePaths = map[string]bool{
	"system/user/profile":  true, // 修改个人资料
	"system/user/password": true, // 修改个人密码
	"system/user/avatar":   true, // 修改个人头像
}

// permissionPrefixes 参与权限校验的模块路径前缀（系统管理 / 系统监控 / 定时任务 / 代码生成 / 系统工具）
var permissionPrefixes = []string{"system/", "monitor/", "cron/", "gen/", "tool/"}

// pathPermissionAlias 路径无法推导出权限标识时的映射（沿用既有权限标识）
var pathPermissionAlias = map[string]string{
	"monitor/server":     "monitor:server:list",     // 服务监控
	"monitor/cache":      "monitor:cache:list",      // 缓存监控
	"monitor/cache/list": "monitor:cache-list:list", // 缓存列表
	"gen/download":       "gen:code",                // 下载生成代码（沿用「生成代码」权限）
}

// authService 认证服务实例（中间件内部使用）
var authService = service.Auth{}

// Auth 认证中间件
// @author: [bluefrog](https://github.com/freewu)
// @function: Auth
// @description: 校验请求令牌并把登录用户写入上下文；认证关闭（system.auth.enabled=false）时直接放行；
// 开启权限校验（system.auth.permission-enabled=true）时按请求路径推导权限标识并与用户权限比对
// @return: gin.HandlerFunc
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Bool("system.auth.enabled") {
			c.Next()
			return
		}

		token := Token(c)
		if token == "" {
			abort(c, http.StatusUnauthorized, 401, "认证失败，无法访问系统资源")
			return
		}

		loginUser, err := authService.GetLoginUser(token)
		if err != nil {
			abort(c, http.StatusUnauthorized, 401, err.Error())
			return
		}
		c.Set(ContextToken, token)
		c.Set(ContextLoginUser, loginUser)

		// 权限校验（可选开关）
		if config.Bool("system.auth.permission-enabled") {
			if permission := PathPermission(c.Request.URL.Path); permission != "" &&
				!authService.HasPermission(loginUser, permission) {
				abort(c, http.StatusForbidden, 403, "没有权限，请联系管理员授权")
				return
			}
		}

		c.Next()
	}
}

// Token 从请求中提取令牌
// @author: [bluefrog](https://github.com/freewu)
// @function: Token
// @description: 依次从请求头 Authorization: Bearer xxx、请求头 token、查询参数 token 中提取
// @param: c *gin.Context
// @return: string
func Token(c *gin.Context) string {
	if authorization := c.GetHeader("Authorization"); authorization != "" {
		if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			return strings.TrimSpace(authorization[7:])
		}
		return strings.TrimSpace(authorization)
	}
	if token := c.GetHeader("token"); token != "" {
		return strings.TrimSpace(token)
	}
	return strings.TrimSpace(c.Query("token"))
}

// PathPermission 由请求路径推导权限标识
// @author: [bluefrog](https://github.com/freewu)
// @function: PathPermission
// @description: 约定 /system/user/list -> system:user:list；路径先按 KebabPath 规范化（大写字母转「-小写」），
// 因此 /cron/jobLog/list 与 /cron/job-log/list 都推导为 cron:job-log:list；非业务接口与自助接口返回空（不校验）
// @param: path string 请求路径
// @return: string 权限标识，空表示无需校验
func PathPermission(path string) string {
	path = KebabPath(strings.Trim(path, "/"))
	if path == "" || selfServicePaths[path] {
		return ""
	}
	if alias, exists := pathPermissionAlias[path]; exists {
		return alias
	}
	matched := false
	for _, prefix := range permissionPrefixes {
		if strings.HasPrefix(path, prefix) {
			matched = true
			break
		}
	}
	if !matched {
		return ""
	}
	return strings.ReplaceAll(path, "/", ":")
}

// KebabPath 把路径中的大写字母规范为「-小写」（如 userOnline -> user-online，
// jobLog/list -> job-log/list）；路由前缀必须全小写 + 中划线连接，权限标识由规范化后的路径推导，
// 保证 /cron/job-log/list -> cron:job-log:list 这类标识不因大小写写法不同而漂移
// @param: path string 已去除首尾斜杠的路径
// @return: string 规范化后的路径
func KebabPath(path string) string {
	if path == "" {
		return path
	}
	var builder strings.Builder
	builder.Grow(len(path) + 8)
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			builder.WriteByte('-')
			builder.WriteRune(r - 'A' + 'a')
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// GetLoginUser 从上下文中获取登录用户
// @param: c *gin.Context
// @return: *domain.LoginUser 未认证时返回 nil
func GetLoginUser(c *gin.Context) *domain.LoginUser {
	if value, exists := c.Get(ContextLoginUser); exists {
		if loginUser, ok := value.(*domain.LoginUser); ok {
			return loginUser
		}
	}
	return nil
}

// GetUserID 从上下文中获取登录用户ID
// @param: c *gin.Context
// @return: uint
func GetUserID(c *gin.Context) uint {
	if loginUser := GetLoginUser(c); loginUser != nil {
		return loginUser.UserID
	}
	return 0
}

// GetUsername 从上下文中获取登录账号
// @param: c *gin.Context
// @return: string
func GetUsername(c *gin.Context) string {
	if loginUser := GetLoginUser(c); loginUser != nil {
		return loginUser.Username
	}
	return ""
}

// GetToken 从上下文中获取登录令牌
// @param: c *gin.Context
// @return: string
func GetToken(c *gin.Context) string {
	if value, exists := c.Get(ContextToken); exists {
		if token, ok := value.(string); ok {
			return token
		}
	}
	return ""
}

// abort 中止请求并返回统一的响应结构
// @param: c *gin.Context
// @param: httpStatus int HTTP 状态码
// @param: code int 业务状态码
// @param: msg string 提示消息
func abort(c *gin.Context, httpStatus int, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, &response.Response{Code: code, Msg: msg})
}
