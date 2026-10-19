package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestContext 构造用于提取令牌的 gin 上下文
func newTestContext(headers map[string]string, query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/system/user/list"+query, nil)
	for key, value := range headers {
		c.Request.Header.Set(key, value)
	}
	return c
}

// TestToken 校验令牌提取顺序与格式
func TestToken(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		query   string
		want    string
	}{
		{name: "Bearer 前缀", headers: map[string]string{"Authorization": "Bearer abc123"}, want: "abc123"},
		{name: "Bearer 小写前缀", headers: map[string]string{"Authorization": "bearer abc123"}, want: "abc123"},
		{name: "Authorization 不带前缀", headers: map[string]string{"Authorization": "abc123"}, want: "abc123"},
		{name: "token 请求头", headers: map[string]string{"token": "abc123"}, want: "abc123"},
		{name: "查询参数", query: "?token=abc123", want: "abc123"},
		{name: "Authorization 优先于查询参数", headers: map[string]string{"Authorization": "Bearer head"}, query: "?token=query", want: "head"},
		{name: "无令牌", want: ""},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if got := Token(newTestContext(item.headers, item.query)); got != item.want {
				t.Errorf("Token() = %q，期望 %q", got, item.want)
			}
		})
	}
}

// TestPathPermission 校验请求路径到权限标识的推导规则
func TestPathPermission(t *testing.T) {
	cases := map[string]string{
		"/system/user/list":     "system:user:list",
		"/system/dept/tree":     "system:dept:tree",
		"/system/user/profile":  "", // 自助接口不校验
		"/system/user/password": "", // 自助接口不校验
		"/system/user/avatar":   "", // 自助接口不校验
		"/getInfo":              "", // 非业务接口不校验
		"/login":                "",
		"/swagger.json":         "",
		"":                      "",
		"/system/user/list/":    "system:user:list",
	}
	for path, want := range cases {
		if got := PathPermission(path); got != want {
			t.Errorf("PathPermission(%q) = %q，期望 %q", path, got, want)
		}
	}
}

// TestGetLoginUser 校验上下文登录用户读取
func TestGetLoginUser(t *testing.T) {
	c := newTestContext(nil, "")
	if GetLoginUser(c) != nil {
		t.Error("未登录时应返回 nil")
	}
	if GetUserID(c) != 0 || GetUsername(c) != "" || GetToken(c) != "" {
		t.Error("未登录时上下文取值应为空")
	}
}
