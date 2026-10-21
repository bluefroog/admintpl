package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	monitorDomain "github.com/bluefroog/admintpl/app/monitor/domain"
	monitorService "github.com/bluefroog/admintpl/app/monitor/service"
	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/captcha"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/database"
	"github.com/bluefroog/admintpl/common/lib"
	log "github.com/sirupsen/logrus"
)

// Auth 认证服务
type Auth struct{}

// SignIn 用户登录
// @author: [bluefrog](https://github.com/freewu)
// @function: SignIn
// @description: 校验验证码/锁定状态/账号密码，成功后生成令牌并写入在线用户表
// @param: req *domain.LoginRequest 登录请求
// @param: ip string 客户端IP
// @param: userAgent string 请求头 User-Agent
// @return: resp *domain.LoginResponse 登录响应, err error
func (s Auth) SignIn(req *domain.LoginRequest, ip string, userAgent string) (resp *domain.LoginResponse, err error) {
	// 1、验证码校验
	if config.Bool("system.captcha.enabled") {
		if req.Code == "" || req.UUID == "" {
			return nil, errors.New("请输入验证码")
		}
		if !captcha.Verify(req.UUID, req.Code) {
			return nil, errors.New("验证码错误或已过期")
		}
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password
	browser, osName := lib.ParseUserAgent(userAgent)
	maxRetry := config.Int("system.auth.max-retry")
	lockTime := config.Int("system.auth.lock-time")

	// 2、登录失败次数校验：lockTime 分钟内失败 maxRetry 次则锁定
	var failed int64
	if maxRetry > 0 && lockTime > 0 {
		if countErr, count := (monitorService.LoginLog{}).CountRecentFailure(username, lockTime); countErr == nil {
			failed = count
		}
		if int(failed) >= maxRetry {
			s.recordLoginLog(username, ip, browser, osName, "账号已锁定", 1)
			return nil, fmt.Errorf("账号已锁定，请 %d 分钟后再试", lockTime)
		}
	}

	// 3、账号校验
	user, err := User{}.GetByUsername(username)
	if err != nil {
		s.recordLoginLog(username, ip, browser, osName, "账号或密码不正确", 1)
		return nil, errors.New("账号或密码不正确")
	}
	if user.Status != 1 {
		s.recordLoginLog(username, ip, browser, osName, "账号已停用", 1)
		return nil, errors.New("您的账号已停用，请联系管理员")
	}

	// 4、密码校验（前端 hash 值 + 库中盐 → 后端同一算法 hash 后对比）
	if !lib.CheckPassword(user.Password, user.Salt, password) {
		s.recordLoginLog(username, ip, browser, osName, "账号或密码不正确", 1)
		if maxRetry > 0 && lockTime > 0 {
			if remaining := maxRetry - int(failed) - 1; remaining > 0 {
				return nil, fmt.Errorf("账号或密码不正确，还可尝试 %d 次", remaining)
			}
			return nil, fmt.Errorf("账号已锁定，请 %d 分钟后再试", lockTime)
		}
		return nil, errors.New("账号或密码不正确")
	}

	// 5、生成令牌并记录在线状态
	token := lib.Token()
	online := &monitorDomain.UserOnlineCreateRequest{
		UUID:     lib.UUID(),
		Token:    token,
		UserName: user.Username,
		IP:       ip,
		Explorer: browser,
		OS:       osName,
	}
	if err = (monitorService.UserOnline{}).Create(online); err != nil {
		return nil, errors.New("登录失败：" + err.Error())
	}

	// 6、更新最后登录信息
	_ = User{}.UpdateByMap(user.ID, map[string]interface{}{
		"LastLoginTime": time.Now(),
		"LastLoginIP":   ip,
	})

	// 7、记录登录成功日志
	s.recordLoginLog(username, ip, browser, osName, "登录成功", 0)

	expire := config.Int("system.auth.token-expire")
	return &domain.LoginResponse{Token: token, ExpiresIn: int64(expire) * 60}, nil
}

// SignOut 注销登录（删除在线用户记录）
// @author: [bluefrog](https://github.com/freewu)
// @function: SignOut
// @description: 根据令牌删除在线用户记录
// @param: token string 令牌
// @return: err error
func (s Auth) SignOut(token string) error {
	return monitorService.UserOnline{}.DeleteByToken(token)
}

// GetLoginUser 根据令牌获取登录用户信息（含角色与权限，认证中间件调用）
// @author: [bluefrog](https://github.com/freewu)
// @function: GetLoginUser
// @description: 校验令牌有效性/有效期/账号状态，返回登录用户信息
// @param: token string 令牌
// @return: *domain.LoginUser, error
func (s Auth) GetLoginUser(token string) (*domain.LoginUser, error) {
	if token == "" {
		return nil, errors.New("认证失败，无法访问系统资源")
	}
	online, err := monitorService.UserOnline{}.GetByToken(token)
	if err != nil {
		return nil, errors.New("认证失败，无法访问系统资源")
	}

	// 令牌有效期校验
	expire := config.Int("system.auth.token-expire")
	if expire > 0 {
		expireAt := int64(online.CreateTime) + int64(expire)*60*1000
		if time.Now().UnixMilli() > expireAt {
			_ = monitorService.UserOnline{}.DeleteByToken(token)
			return nil, errors.New("登录状态已过期，请重新登录")
		}
	}

	user, err := User{}.GetByUsername(online.UserName)
	if err != nil {
		_ = monitorService.UserOnline{}.DeleteByToken(token)
		return nil, errors.New("认证失败，无法访问系统资源")
	}
	if user.Status != 1 {
		_ = monitorService.UserOnline{}.DeleteByToken(token)
		return nil, errors.New("您的账号已停用，请联系管理员")
	}

	roles, permissions, isAdmin := s.getRolesAndPermissions(user)
	return &domain.LoginUser{
		UserID:       user.ID,
		Username:     user.Username,
		Nickname:     user.Nickname,
		DepartmentID: user.DepartmentID,
		Token:        token,
		IsAdmin:      isAdmin,
		Roles:        roles,
		Permissions:  permissions,
	}, nil
}

// GetUserInfo 获取登录用户信息（用户资料 + 角色 + 权限）
// @author: [bluefrog](https://github.com/freewu)
// @function: GetUserInfo
// @description: 获取指定用户的资料、角色标识与权限标识
// @param: userID uint 用户ID
// @return: *domain.UserInfoResponse, error
func (s Auth) GetUserInfo(userID uint) (*domain.UserInfoResponse, error) {
	user, err := User{}.Detail(userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	roles, permissions, isAdmin := s.getRolesAndPermissions(user)
	return &domain.UserInfoResponse{
		User:        user,
		Roles:       roles,
		Permissions: permissions,
		IsAdmin:     isAdmin,
	}, nil
}

// GetRouters 获取登录用户的前端路由（菜单树）
// @author: [bluefrog](https://github.com/freewu)
// @function: GetRouters
// @description: 超级管理员返回全部菜单，其余按角色授权菜单返回，并转换为前端路由结构
// @param: userID uint 用户ID
// @return: []*domain.RouterResponse, error
func (s Auth) GetRouters(userID uint) ([]*domain.RouterResponse, error) {
	user, err := User{}.Detail(userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	routers := make([]*domain.RouterResponse, 0)
	menus := make([]domain.Menu, 0)
	db := database.DB.Model(&domain.Menu{}).
		Where("menu_type in (?) and status = ?", []string{"M", "C"}, 0)
	if !s.isAdmin(user) {
		menuIds := s.getMenuIds(user.ID)
		if len(menuIds) == 0 {
			return routers, nil
		}
		db = db.Where("menu_id in (?)", menuIds)
	}
	if err = db.Order("parent_id, order_num").Find(&menus).Error; err != nil {
		return nil, err
	}
	return menuListToRouters(menus), nil
}

// HasPermission 校验登录用户是否拥有指定权限标识
// @author: [bluefrog](https://github.com/freewu)
// @function: HasPermission
// @description: 超级管理员或拥有 *:*:* 的用户直接放行
// @param: user *domain.LoginUser 登录用户
// @param: permission string 权限标识
// @return: bool
func (s Auth) HasPermission(user *domain.LoginUser, permission string) bool {
	if user == nil {
		return false
	}
	if user.IsAdmin {
		return true
	}
	for _, item := range user.Permissions {
		if item == "*:*:*" || item == permission {
			return true
		}
	}
	return false
}

// Captcha 生成图形验证码
// @author: [bluefrog](https://github.com/freewu)
// @function: Captcha
// @description: 按配置决定是否开启验证码，开启时返回验证码标识与图片
// @return: *domain.CaptchaResponse, error
func (s Auth) Captcha() (*domain.CaptchaResponse, error) {
	if !config.Bool("system.captcha.enabled") {
		return &domain.CaptchaResponse{Enabled: false}, nil
	}
	length := config.Int("system.captcha.length")
	expire := config.Int("system.captcha.expire")
	id, image, code, err := captcha.Generate(length, time.Duration(expire)*time.Second)
	if err != nil {
		return nil, err
	}
	// 调试模式下输出验证码，便于开发环境自测
	log.Debugf("图形验证码 %s = %s", id, code)
	return &domain.CaptchaResponse{Enabled: true, UUID: id, Img: image}, nil
}

// FrontendConfig 获取前端初始化配置（登录页在登录前读取，无需令牌）
// @author: [bluefrog](https://github.com/freewu)
// @function: FrontendConfig
// @description: 返回密码 hash 算法与是否开启验证码等前端启动参数
// @return: *domain.FrontendConfigResponse
func (s Auth) FrontendConfig() *domain.FrontendConfigResponse {
	return &domain.FrontendConfigResponse{
		HashAlgorithm:  lib.FrontendHashAlgorithm(),
		CaptchaEnabled: config.Bool("system.captcha.enabled"),
	}
}

// CleanExpiredOnline 清理过期的在线用户记录（服务启动时调用）
// @author: [bluefrog](https://github.com/freewu)
// @function: CleanExpiredOnline
// @description: 删除超过令牌有效期的在线用户记录
func (s Auth) CleanExpiredOnline() {
	expire := config.Int("system.auth.token-expire")
	if expire <= 0 {
		return
	}
	deadline := time.Now().Add(-time.Duration(expire) * time.Minute).UnixMilli()
	if err := database.DB.Where("create_time < ?", deadline).Unscoped().Delete(&monitorDomain.UserOnline{}).Error; err != nil {
		log.Warnf("清理过期在线用户失败：%v", err)
	}
}

// superAdminUserId 内置超级管理员用户ID（与角色表中的约定保持一致）
const superAdminUserId = 1

// isAdmin 是否超级管理员（用户标记为管理员，或为内置超级管理员）
// @param: user *domain.User 用户
// @return: bool
func (s Auth) isAdmin(user *domain.User) bool {
	if user == nil {
		return false
	}
	return user.IsAdmin || user.ID == superAdminUserId
}

// getMenuIds 获取用户角色关联的菜单ID（仅有效角色）
// @param: userID uint 用户ID
// @return: []uint
func (s Auth) getMenuIds(userID uint) []uint {
	roleIds := s.getRoleIds(userID)
	if len(roleIds) == 0 {
		return make([]uint, 0)
	}
	menuIds := make([]uint, 0)
	if err := database.DB.Model(&domain.RoleMenu{}).Where("role_id in (?)", roleIds).Pluck("menu_id", &menuIds).Error; err != nil {
		log.Warnf("查询用户角色菜单失败：%v", err)
	}
	return menuIds
}

// getRoleIds 获取用户的有效角色ID（状态正常且未删除）
// @param: userID uint 用户ID
// @return: []uint
func (s Auth) getRoleIds(userID uint) []uint {
	roleIds := make([]uint, 0)
	if err := database.DB.Model(&domain.UserRole{}).Where("user_id = ?", userID).Pluck("role_id", &roleIds).Error; err != nil {
		log.Warnf("查询用户角色失败：%v", err)
		return roleIds
	}
	if len(roleIds) == 0 {
		return roleIds
	}
	validIds := make([]uint, 0)
	if err := database.DB.Model(&domain.Role{}).
		Where("role_id in (?) and status = ? and del_flag = ?", roleIds, 0, 0).
		Pluck("role_id", &validIds).Error; err != nil {
		log.Warnf("查询有效角色失败：%v", err)
		return validIds
	}
	return validIds
}

// getRolesAndPermissions 获取用户的角色标识与权限标识
// @param: user *domain.User 用户
// @return: roles []string 角色标识, permissions []string 权限标识, isAdmin bool 是否超级管理员
func (s Auth) getRolesAndPermissions(user *domain.User) (roles []string, permissions []string, isAdmin bool) {
	roles = make([]string, 0)
	permissions = make([]string, 0)
	if s.isAdmin(user) {
		return append(roles, "admin"), append(permissions, "*:*:*"), true
	}

	roleIds := s.getRoleIds(user.ID)
	if len(roleIds) == 0 {
		return roles, permissions, false
	}
	if err := database.DB.Model(&domain.Role{}).
		Where("role_id in (?)", roleIds).
		Distinct().
		Pluck("role_key", &roles).Error; err != nil {
		log.Warnf("查询角色标识失败：%v", err)
	}
	menuIds := s.getMenuIds(user.ID)
	if len(menuIds) == 0 {
		return roles, permissions, false
	}
	if err := database.DB.Model(&domain.Menu{}).
		Where("menu_id in (?) and status = ? and perms <> ''", menuIds, 0).
		Distinct().
		Pluck("perms", &permissions).Error; err != nil {
		log.Warnf("查询权限标识失败：%v", err)
	}
	return roles, permissions, false
}

// recordLoginLog 记录登录日志
// @param: username string 登录账号
// @param: ip string 客户端IP
// @param: browser string 浏览器
// @param: osName string 操作系统
// @param: message string 提示消息
// @param: status uint 登录状态（0成功 1失败）
func (s Auth) recordLoginLog(username, ip, browser, osName, message string, status uint) {
	req := &monitorDomain.LoginLogCreateRequest{
		LoginLogBase: monitorDomain.LoginLogBase{
			Name:      username,
			IPAddress: ip,
			Location:  lib.IPLocation(ip),
			Browser:   browser,
			OS:        osName,
			Message:   message,
			Module:    "后台管理",
			Time:      strconv.FormatInt(time.Now().UnixMilli(), 10),
			Status:    status,
		},
	}
	if err := (monitorService.LoginLog{}).Create(req); err != nil {
		log.Warnf("记录登录日志失败：%v", err)
	}
}

// menuListToRouters 菜单列表转前端路由树
// @param: menus []domain.Menu 菜单列表
// @return: []*domain.RouterResponse
func menuListToRouters(menus []domain.Menu) []*domain.RouterResponse {
	roots := make([]*domain.RouterResponse, 0)
	if len(menus) == 0 {
		return roots
	}
	nodes := make(map[uint]*routerNode, len(menus))
	order := make([]uint, 0, len(menus))
	for _, menu := range menus {
		name := menu.RouteName
		if name == "" {
			name = capitalise(menu.Path)
		}
		nodes[menu.ID] = &routerNode{
			menu:     menu,
			parentID: menu.ParentID,
			resp: &domain.RouterResponse{
				Name:      name,
				Path:      menu.Path,
				Hidden:    menu.Visible == "1",
				Component: menu.Component,
				Meta: &domain.RouterMeta{
					Title:   menu.Name,
					Icon:    menu.Icon,
					NoCache: menu.IsCache == 1,
				},
			},
		}
		order = append(order, menu.ID)
	}
	// 组装父子层级
	for _, id := range order {
		node := nodes[id]
		if parent, ok := nodes[node.parentID]; ok && node.parentID != 0 {
			parent.resp.Children = append(parent.resp.Children, node.resp)
			continue
		}
		roots = append(roots, node.resp)
	}
	// 按前端约定补全路由信息
	for _, id := range order {
		applyRouterRule(nodes[id], nodes)
	}
	return roots
}

// routerNode 路由构建时的中间结构（保留菜单原始数据便于补全路由）
type routerNode struct {
	menu     domain.Menu
	parentID uint
	resp     *domain.RouterResponse
}

// applyRouterRule 按前端约定补全路由层级信息（Layout/ParentView、外链、重定向等）
// @param: node *routerNode 当前路由节点
// @param: nodes map[uint]*routerNode 全部路由节点
func applyRouterRule(node *routerNode, nodes map[uint]*routerNode) {
	_, hasParent := nodes[node.parentID]
	topLevel := node.parentID == 0 || !hasParent

	// 外链菜单：地址写入 meta.link，顶层路径统一为 /
	if node.menu.IsFrame == 0 {
		node.resp.Meta.Link = node.menu.Path
		if topLevel {
			node.resp.Path = "/"
		}
	}

	if node.menu.MenuType == "M" {
		// 目录：顶层用 Layout，子级用 ParentView
		if topLevel {
			node.resp.Component = "Layout"
			node.resp.Path = ensureLeadingSlash(node.resp.Path)
		} else {
			node.resp.Component = "ParentView"
		}
		if len(node.resp.Children) > 0 {
			node.resp.AlwaysShow = true
			node.resp.Redirect = "noRedirect"
		}
		return
	}

	// 菜单：顶层补全前导斜杠，子级保持相对路径
	if topLevel {
		node.resp.Path = ensureLeadingSlash(node.resp.Path)
	}
}

// capitalise 首字母大写（用于生成路由名称）
// @param: str string 原始字符串
// @return: string
func capitalise(str string) string {
	if str == "" {
		return str
	}
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ensureLeadingSlash 保证路径以 / 开头
// @param: path string 路径
// @return: string
func ensureLeadingSlash(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
