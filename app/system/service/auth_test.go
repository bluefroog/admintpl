package service

import (
	"testing"

	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/lib"
)

// TestHasPermission 校验权限判定（超级管理员放行、通配权限、精确匹配）
func TestHasPermission(t *testing.T) {
	auth := Auth{}
	cases := []struct {
		name       string
		loginUser  *domain.LoginUser
		permission string
		want       bool
	}{
		{name: "未登录", loginUser: nil, permission: "system:user:list", want: false},
		{
			name:       "超级管理员直接放行",
			loginUser:  &domain.LoginUser{IsAdmin: true},
			permission: "system:user:list",
			want:       true,
		},
		{
			name:       "通配权限",
			loginUser:  &domain.LoginUser{Permissions: []string{"*:*:*"}},
			permission: "system:user:list",
			want:       true,
		},
		{
			name:       "精确匹配",
			loginUser:  &domain.LoginUser{Permissions: []string{"system:user:query"}},
			permission: "system:user:query",
			want:       true,
		},
		{
			name:       "无该权限",
			loginUser:  &domain.LoginUser{Permissions: []string{"system:user:query"}},
			permission: "system:user:remove",
			want:       false,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if got := auth.HasPermission(item.loginUser, item.permission); got != item.want {
				t.Errorf("HasPermission() = %v，期望 %v", got, item.want)
			}
		})
	}
}

// TestMenuListToRouters 校验菜单转前端路由（层级/目录组件/外链/路径补全/隐藏）
func TestMenuListToRouters(t *testing.T) {
	menus := []domain.Menu{
		{ID: 1, MenuBase: domain.MenuBase{Name: "系统管理", ParentID: 0, Path: "system", MenuType: "M", Icon: "system", IsFrame: 1, Visible: "0"}},
		{ID: 100, MenuBase: domain.MenuBase{Name: "用户管理", ParentID: 1, Path: "user", Component: "system/user/index", MenuType: "C", Perms: "system:user:list", IsFrame: 1, Visible: "0", IsCache: 1}},
		{ID: 101, MenuBase: domain.MenuBase{Name: "角色管理", ParentID: 1, Path: "role", Component: "system/role/index", MenuType: "C", IsFrame: 1, Visible: "1"}},
		{ID: 4, MenuBase: domain.MenuBase{Name: "项目主页", ParentID: 0, Path: "https://github.com/bluefroog/admintpl", MenuType: "M", IsFrame: 0, Visible: "0"}},
		{ID: 108, MenuBase: domain.MenuBase{Name: "日志管理", ParentID: 1, Path: "log", MenuType: "M", IsFrame: 1, Visible: "0"}},
		{ID: 500, MenuBase: domain.MenuBase{Name: "操作日志", ParentID: 108, Path: "operlog", Component: "monitor/operlog/index", MenuType: "C", IsFrame: 1, Visible: "0"}},
	}

	routers := menuListToRouters(menus)
	if len(routers) != 2 {
		t.Fatalf("顶层路由应为 2 个，实际 %d", len(routers))
	}

	system := routers[0]
	if system.Path != "/system" {
		t.Errorf("顶层目录路径应补全为 /system，实际 %s", system.Path)
	}
	if system.Component != "Layout" {
		t.Errorf("顶层目录组件应为 Layout，实际 %s", system.Component)
	}
	if !system.AlwaysShow || system.Redirect != "noRedirect" {
		t.Errorf("含子路由的目录应设置 alwaysShow 与 noRedirect，实际 %+v", system)
	}
	if system.Name != "System" {
		t.Errorf("路由名称应为首字母大写，实际 %s", system.Name)
	}
	if len(system.Children) != 3 {
		t.Fatalf("系统管理应有 3 个子路由，实际 %d", len(system.Children))
	}

	user := system.Children[0]
	if user.Path != "user" || user.Component != "system/user/index" {
		t.Errorf("子菜单路径/组件错误：%+v", user)
	}
	if user.Meta == nil || user.Meta.Title != "用户管理" || !user.Meta.NoCache {
		t.Errorf("子菜单元信息错误：%+v", user.Meta)
	}

	// 隐藏菜单
	if !system.Children[1].Hidden {
		t.Error("visible=1 的菜单应隐藏")
	}

	// 子目录使用 ParentView
	logNode := system.Children[2]
	if logNode.Component != "ParentView" {
		t.Errorf("子目录组件应为 ParentView，实际 %s", logNode.Component)
	}
	if len(logNode.Children) != 1 || logNode.Children[0].Path != "operlog" {
		t.Errorf("子目录下菜单错误：%+v", logNode.Children)
	}

	// 外链菜单
	guide := routers[1]
	if guide.Path != "/" || guide.Meta.Link != "https://github.com/bluefroog/admintpl" {
		t.Errorf("外链菜单应为 path=/ 且 meta.link 为外链地址，实际 %+v", guide)
	}

	if got := menuListToRouters(nil); len(got) != 0 {
		t.Errorf("空菜单应返回空切片，实际 %d", len(got))
	}
}

// TestFrontendConfig 校验前端初始化配置接口：hash 算法合法且与后端配置一致，验证码开关可读
func TestFrontendConfig(t *testing.T) {
	data := Auth{}.FrontendConfig()
	if data == nil {
		t.Fatal("FrontendConfig() 不应返回 nil")
	}
	if _, err := lib.NormalizeHashAlgorithm(data.HashAlgorithm); err != nil {
		t.Errorf("下发的前端 hash 算法应合法，实际 %q（%v）", data.HashAlgorithm, err)
	}
	if data.HashAlgorithm != lib.FrontendHashAlgorithm() {
		t.Errorf("hash 算法应与配置一致，期望 %q，实际 %q", lib.FrontendHashAlgorithm(), data.HashAlgorithm)
	}
	if data.CaptchaEnabled != config.Bool("system.captcha.enabled") {
		t.Errorf("验证码开关应与配置一致，期望 %v，实际 %v", config.Bool("system.captcha.enabled"), data.CaptchaEnabled)
	}
}
