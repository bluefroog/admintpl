package domain

// UserIdRequest 用户ID请求参数（关闭认证时用于调试 /getInfo、/getRouters）
type UserIdRequest struct {
	UserId uint `json:"userId" form:"userId"` // 用户ID
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" form:"username" validate:"required"` // 登录账号
	Password string `json:"password" form:"password" validate:"required"` // 登录密码
	Code     string `json:"code" form:"code"`                             // 验证码（开启验证码时必填）
	UUID     string `json:"uuid" form:"uuid"`                             // 验证码标识
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Token     string `json:"token"`     // 令牌
	ExpiresIn int64  `json:"expiresIn"` // 令牌有效期（秒）
}

// CaptchaResponse 验证码响应结构体
type CaptchaResponse struct {
	Enabled bool   `json:"captchaEnabled"` // 是否开启验证码
	UUID    string `json:"uuid"`           // 验证码标识
	Img     string `json:"img"`            // 验证码图片（base64，前端拼接 data:image/png;base64, 前缀）
}

// LoginUser 登录用户信息（认证通过后写入请求上下文）
type LoginUser struct {
	UserID       uint     `json:"userId"`      // 用户ID
	Username     string   `json:"userName"`    // 登录账号
	Nickname     string   `json:"nickName"`    // 用户昵称
	DepartmentID uint     `json:"deptId"`      // 所属部门ID（数据权限使用）
	Token        string   `json:"token"`       // 令牌
	IsAdmin      bool     `json:"isAdmin"`     // 是否超级管理员
	Roles        []string `json:"roles"`       // 角色标识列表
	Permissions  []string `json:"permissions"` // 权限标识列表
}

// UserInfoResponse 登录用户信息响应结构体
type UserInfoResponse struct {
	User        *User    `json:"user"`        // 用户信息
	Roles       []string `json:"roles"`       // 角色标识列表
	Permissions []string `json:"permissions"` // 权限标识列表
	IsAdmin     bool     `json:"isAdmin"`     // 是否超级管理员
}

// RouterMeta 前端路由元信息
type RouterMeta struct {
	Title   string `json:"title"`             // 菜单名称
	Icon    string `json:"icon,omitempty"`    // 菜单图标
	NoCache bool   `json:"noCache,omitempty"` // 是否不缓存
	Link    string `json:"link,omitempty"`    // 外链地址
}

// RouterResponse 前端路由结构体（与若依前端约定保持一致）
type RouterResponse struct {
	Name       string            `json:"name,omitempty"`       // 路由名称
	Path       string            `json:"path"`                 // 路由地址
	Hidden     bool              `json:"hidden"`               // 是否隐藏
	Redirect   string            `json:"redirect,omitempty"`   // 重定向地址
	Component  string            `json:"component,omitempty"`  // 组件路径
	AlwaysShow bool              `json:"alwaysShow,omitempty"` // 是否始终显示根菜单
	Meta       *RouterMeta       `json:"meta,omitempty"`       // 元信息
	Children   []*RouterResponse `json:"children,omitempty"`   // 子路由
}
