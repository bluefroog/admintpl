package service

import (
	"time"

	"github.com/bluefroog/admintpl/app/system/domain"
	"github.com/bluefroog/admintpl/common/lib"
	"github.com/bluefroog/admintpl/common/model/core"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Seed 基础数据初始化服务
type Seed struct{}

// 内置账号初始密码（与若依保持一致，登录后请立即修改）
const seedInitPassword = "admin123"

// menuSeed 菜单种子数据（紧凑结构，便于维护）
type menuSeed struct {
	id        uint
	parentID  uint
	name      string
	sort      uint
	path      string
	component string
	menuType  string // M目录 C菜单 F按钮
	perms     string
	icon      string
	external  bool   // 是否外链
	visible   string // 0显示 1隐藏
}

// jobSeed 定时任务种子数据
type jobSeed struct {
	name           string
	group          string
	executor       string
	invokeTarget   string
	jobArgs        string
	cronExpression string
	misfirePolicy  string
	concurrent     string
	status         uint
}

// jobSeeds 若依内置示例任务（默认「暂停」，不会自动执行；可在定时任务页面启用或立即执行一次）
var jobSeeds = []jobSeed{
	{
		name: "系统默认（无参）", group: "default", executor: "default",
		invokeTarget: "ryTask.ryNoParams()", cronExpression: "0/10 * * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentRefuse, status: JobStatusPaused,
	},
	{
		name: "系统默认（有参）", group: "default", executor: "default",
		invokeTarget: "ryTask.ryParams('ry')", cronExpression: "0/15 * * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentRefuse, status: JobStatusPaused,
	},
	{
		name: "系统默认（多参）", group: "default", executor: "default",
		invokeTarget: "ryTask.ryMultipleParams('ry', true, 2000, 1.0)", cronExpression: "0/20 * * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentRefuse, status: JobStatusPaused,
	},
	{
		name: "示例任务（无参，job_args 传参）", group: "default", executor: "default",
		invokeTarget: "testTask.test1()", jobArgs: "hello|2000", cronExpression: "0 0/5 * * * ?",
		misfirePolicy: JobMisfireDoNothing, concurrent: JobConcurrentAllow, status: JobStatusPaused,
	},
}

// menuSeeds 若依标准菜单（目录 + 菜单 + 按钮）
var menuSeeds = []menuSeed{
	// ---- 目录 ----
	{id: 1, parentID: 0, name: "系统管理", sort: 1, path: "system", menuType: "M", icon: "system"},
	{id: 2, parentID: 0, name: "系统监控", sort: 2, path: "monitor", menuType: "M", icon: "monitor"},
	{id: 3, parentID: 0, name: "系统工具", sort: 3, path: "tool", menuType: "M", icon: "tool"},
	{id: 4, parentID: 0, name: "若依官网", sort: 4, path: "http://ruoyi.vip", menuType: "M", icon: "guide", external: true},

	// ---- 系统管理 ----
	{id: 100, parentID: 1, name: "用户管理", sort: 1, path: "user", component: "system/user/index", menuType: "C", perms: "system:user:list", icon: "user"},
	{id: 101, parentID: 1, name: "角色管理", sort: 2, path: "role", component: "system/role/index", menuType: "C", perms: "system:role:list", icon: "peoples"},
	{id: 102, parentID: 1, name: "菜单管理", sort: 3, path: "menu", component: "system/menu/index", menuType: "C", perms: "system:menu:list", icon: "tree-table"},
	{id: 103, parentID: 1, name: "部门管理", sort: 4, path: "dept", component: "system/dept/index", menuType: "C", perms: "system:dept:list", icon: "tree"},
	{id: 104, parentID: 1, name: "岗位管理", sort: 5, path: "post", component: "system/post/index", menuType: "C", perms: "system:post:list", icon: "post"},
	{id: 105, parentID: 1, name: "字典管理", sort: 6, path: "dict", component: "system/dict/index", menuType: "C", perms: "system:dict:list", icon: "dict"},
	{id: 106, parentID: 1, name: "参数设置", sort: 7, path: "config", component: "system/config/index", menuType: "C", perms: "system:config:list", icon: "edit"},
	{id: 107, parentID: 1, name: "通知公告", sort: 8, path: "notice", component: "system/notice/index", menuType: "C", perms: "system:notice:list", icon: "message"},
	{id: 108, parentID: 1, name: "日志管理", sort: 9, path: "log", menuType: "M", icon: "log"},
	{id: 500, parentID: 108, name: "操作日志", sort: 1, path: "operlog", component: "monitor/operlog/index", menuType: "C", perms: "monitor:operlog:list", icon: "form"},
	{id: 501, parentID: 108, name: "登录日志", sort: 2, path: "logininfor", component: "monitor/logininfor/index", menuType: "C", perms: "monitor:logininfor:list", icon: "logininfor"},

	// ---- 系统监控 ----
	{id: 109, parentID: 2, name: "在线用户", sort: 1, path: "online", component: "monitor/online/index", menuType: "C", perms: "monitor:online:list", icon: "online"},
	{id: 110, parentID: 2, name: "定时任务", sort: 2, path: "job", component: "monitor/job/index", menuType: "C", perms: "monitor:job:list", icon: "job"},
	{id: 111, parentID: 2, name: "缓存监控", sort: 3, path: "cache", component: "monitor/cache/index", menuType: "C", perms: "monitor:cache:list", icon: "redis"},
	{id: 112, parentID: 2, name: "缓存列表", sort: 4, path: "cacheList", component: "monitor/cacheList/index", menuType: "C", perms: "monitor:cacheList:list", icon: "redis-list"},
	{id: 113, parentID: 2, name: "服务监控", sort: 5, path: "server", component: "monitor/server/index", menuType: "C", perms: "monitor:server:list", icon: "server"},

	// ---- 系统工具 ----
	{id: 114, parentID: 3, name: "表单构建", sort: 1, path: "build", component: "tool/build/index", menuType: "C", perms: "tool:build:list", icon: "build"},
	{id: 115, parentID: 3, name: "代码生成", sort: 2, path: "gen", component: "tool/gen/index", menuType: "C", perms: "tool:gen:list", icon: "code"},
	{id: 116, parentID: 3, name: "系统接口", sort: 3, path: "swagger", component: "tool/swagger/index", menuType: "C", perms: "tool:swagger:list", icon: "swagger"},

	// ---- 用户管理按钮 ----
	{id: 1001, parentID: 100, name: "用户查询", sort: 1, menuType: "F", perms: "system:user:query"},
	{id: 1002, parentID: 100, name: "用户新增", sort: 2, menuType: "F", perms: "system:user:add"},
	{id: 1003, parentID: 100, name: "用户修改", sort: 3, menuType: "F", perms: "system:user:edit"},
	{id: 1004, parentID: 100, name: "用户删除", sort: 4, menuType: "F", perms: "system:user:remove"},
	{id: 1005, parentID: 100, name: "用户导出", sort: 5, menuType: "F", perms: "system:user:export"},
	{id: 1006, parentID: 100, name: "用户导入", sort: 6, menuType: "F", perms: "system:user:import"},
	{id: 1007, parentID: 100, name: "重置密码", sort: 7, menuType: "F", perms: "system:user:resetPwd"},

	// ---- 角色管理按钮 ----
	{id: 1008, parentID: 101, name: "角色查询", sort: 1, menuType: "F", perms: "system:role:query"},
	{id: 1009, parentID: 101, name: "角色新增", sort: 2, menuType: "F", perms: "system:role:add"},
	{id: 1010, parentID: 101, name: "角色修改", sort: 3, menuType: "F", perms: "system:role:edit"},
	{id: 1011, parentID: 101, name: "角色删除", sort: 4, menuType: "F", perms: "system:role:remove"},
	{id: 1012, parentID: 101, name: "角色导出", sort: 5, menuType: "F", perms: "system:role:export"},

	// ---- 菜单管理按钮 ----
	{id: 1013, parentID: 102, name: "菜单查询", sort: 1, menuType: "F", perms: "system:menu:query"},
	{id: 1014, parentID: 102, name: "菜单新增", sort: 2, menuType: "F", perms: "system:menu:add"},
	{id: 1015, parentID: 102, name: "菜单修改", sort: 3, menuType: "F", perms: "system:menu:edit"},
	{id: 1016, parentID: 102, name: "菜单删除", sort: 4, menuType: "F", perms: "system:menu:remove"},

	// ---- 部门管理按钮 ----
	{id: 1017, parentID: 103, name: "部门查询", sort: 1, menuType: "F", perms: "system:dept:query"},
	{id: 1018, parentID: 103, name: "部门新增", sort: 2, menuType: "F", perms: "system:dept:add"},
	{id: 1019, parentID: 103, name: "部门修改", sort: 3, menuType: "F", perms: "system:dept:edit"},
	{id: 1020, parentID: 103, name: "部门删除", sort: 4, menuType: "F", perms: "system:dept:remove"},

	// ---- 岗位管理按钮 ----
	{id: 1021, parentID: 104, name: "岗位查询", sort: 1, menuType: "F", perms: "system:post:query"},
	{id: 1022, parentID: 104, name: "岗位新增", sort: 2, menuType: "F", perms: "system:post:add"},
	{id: 1023, parentID: 104, name: "岗位修改", sort: 3, menuType: "F", perms: "system:post:edit"},
	{id: 1024, parentID: 104, name: "岗位删除", sort: 4, menuType: "F", perms: "system:post:remove"},
	{id: 1025, parentID: 104, name: "岗位导出", sort: 5, menuType: "F", perms: "system:post:export"},

	// ---- 字典管理按钮 ----
	{id: 1026, parentID: 105, name: "字典查询", sort: 1, menuType: "F", perms: "system:dict:query"},
	{id: 1027, parentID: 105, name: "字典新增", sort: 2, menuType: "F", perms: "system:dict:add"},
	{id: 1028, parentID: 105, name: "字典修改", sort: 3, menuType: "F", perms: "system:dict:edit"},
	{id: 1029, parentID: 105, name: "字典删除", sort: 4, menuType: "F", perms: "system:dict:remove"},
	{id: 1030, parentID: 105, name: "字典导出", sort: 5, menuType: "F", perms: "system:dict:export"},

	// ---- 参数设置按钮 ----
	{id: 1031, parentID: 106, name: "参数查询", sort: 1, menuType: "F", perms: "system:config:query"},
	{id: 1032, parentID: 106, name: "参数新增", sort: 2, menuType: "F", perms: "system:config:add"},
	{id: 1033, parentID: 106, name: "参数修改", sort: 3, menuType: "F", perms: "system:config:edit"},
	{id: 1034, parentID: 106, name: "参数删除", sort: 4, menuType: "F", perms: "system:config:remove"},
	{id: 1035, parentID: 106, name: "参数导出", sort: 5, menuType: "F", perms: "system:config:export"},

	// ---- 通知公告按钮 ----
	{id: 1036, parentID: 107, name: "公告查询", sort: 1, menuType: "F", perms: "system:notice:query"},
	{id: 1037, parentID: 107, name: "公告新增", sort: 2, menuType: "F", perms: "system:notice:add"},
	{id: 1038, parentID: 107, name: "公告修改", sort: 3, menuType: "F", perms: "system:notice:edit"},
	{id: 1039, parentID: 107, name: "公告删除", sort: 4, menuType: "F", perms: "system:notice:remove"},

	// ---- 操作日志按钮 ----
	{id: 1040, parentID: 500, name: "操作查询", sort: 1, menuType: "F", perms: "monitor:operlog:query"},
	{id: 1041, parentID: 500, name: "操作删除", sort: 2, menuType: "F", perms: "monitor:operlog:remove"},
	{id: 1042, parentID: 500, name: "日志导出", sort: 3, menuType: "F", perms: "monitor:operlog:export"},

	// ---- 登录日志按钮 ----
	{id: 1043, parentID: 501, name: "登录查询", sort: 1, menuType: "F", perms: "monitor:logininfor:query"},
	{id: 1044, parentID: 501, name: "登录删除", sort: 2, menuType: "F", perms: "monitor:logininfor:remove"},
	{id: 1045, parentID: 501, name: "日志导出", sort: 3, menuType: "F", perms: "monitor:logininfor:export"},

	// ---- 在线用户按钮 ----
	{id: 1046, parentID: 109, name: "在线查询", sort: 1, menuType: "F", perms: "monitor:online:query"},
	{id: 1047, parentID: 109, name: "强退", sort: 2, menuType: "F", perms: "monitor:online:forceLogout"},

	// ---- 定时任务按钮 ----
	{id: 1049, parentID: 110, name: "任务查询", sort: 1, menuType: "F", perms: "monitor:job:query"},
	{id: 1050, parentID: 110, name: "任务新增", sort: 2, menuType: "F", perms: "monitor:job:add"},
	{id: 1051, parentID: 110, name: "任务修改", sort: 3, menuType: "F", perms: "monitor:job:edit"},
	{id: 1052, parentID: 110, name: "任务删除", sort: 4, menuType: "F", perms: "monitor:job:remove"},
	{id: 1053, parentID: 110, name: "状态修改", sort: 5, menuType: "F", perms: "monitor:job:changeStatus"},
	{id: 1054, parentID: 110, name: "任务导出", sort: 6, menuType: "F", perms: "monitor:job:export"},

	// ---- 代码生成按钮 ----
	{id: 1055, parentID: 115, name: "生成查询", sort: 1, menuType: "F", perms: "tool:gen:query"},
	{id: 1056, parentID: 115, name: "生成修改", sort: 2, menuType: "F", perms: "tool:gen:edit"},
	{id: 1057, parentID: 115, name: "生成删除", sort: 3, menuType: "F", perms: "tool:gen:remove"},
	{id: 1058, parentID: 115, name: "导入代码", sort: 4, menuType: "F", perms: "tool:gen:import"},
	{id: 1059, parentID: 115, name: "预览代码", sort: 5, menuType: "F", perms: "tool:gen:preview"},
	{id: 1060, parentID: 115, name: "生成代码", sort: 6, menuType: "F", perms: "tool:gen:code"},
}

// commonRoleMenuIds 普通角色（role_id=2）默认授权的菜单
var commonRoleMenuIds = []uint{
	1, 2, 100, 101, 102, 103, 104, 105, 106, 107, 108, 500, 501, 109, 110, 111, 112, 113,
	1001, 1008, 1013, 1017, 1021, 1026, 1031, 1036, 1040, 1043, 1046, 1049,
}

// dictDataSeed 字典数据种子
type dictDataSeed struct {
	dictType  string
	sort      uint
	label     string
	value     string
	listClass string
	isDefault string
}

// dictDataSeeds 常用字典数据
var dictDataSeeds = []dictDataSeed{
	{dictType: "sys_user_sex", sort: 1, label: "男", value: "0", listClass: "", isDefault: "Y"},
	{dictType: "sys_user_sex", sort: 2, label: "女", value: "1", listClass: "", isDefault: "N"},
	{dictType: "sys_user_sex", sort: 3, label: "未知", value: "2", listClass: "", isDefault: "N"},
	{dictType: "sys_show_hide", sort: 1, label: "显示", value: "0", listClass: "primary", isDefault: "Y"},
	{dictType: "sys_show_hide", sort: 2, label: "隐藏", value: "1", listClass: "danger", isDefault: "N"},
	{dictType: "sys_normal_disable", sort: 1, label: "正常", value: "0", listClass: "primary", isDefault: "Y"},
	{dictType: "sys_normal_disable", sort: 2, label: "停用", value: "1", listClass: "danger", isDefault: "N"},
	{dictType: "sys_job_status", sort: 1, label: "正常", value: "0", listClass: "primary", isDefault: "Y"},
	{dictType: "sys_job_status", sort: 2, label: "暂停", value: "1", listClass: "danger", isDefault: "N"},
	{dictType: "sys_yes_no", sort: 1, label: "是", value: "Y", listClass: "primary", isDefault: "Y"},
	{dictType: "sys_yes_no", sort: 2, label: "否", value: "N", listClass: "danger", isDefault: "N"},
	{dictType: "sys_notice_type", sort: 1, label: "通知", value: "1", listClass: "warning", isDefault: "Y"},
	{dictType: "sys_notice_type", sort: 2, label: "公告", value: "2", listClass: "success", isDefault: "N"},
	{dictType: "sys_notice_status", sort: 1, label: "正常", value: "0", listClass: "primary", isDefault: "Y"},
	{dictType: "sys_notice_status", sort: 2, label: "关闭", value: "1", listClass: "danger", isDefault: "N"},
	{dictType: "sys_common_status", sort: 1, label: "成功", value: "0", listClass: "primary", isDefault: "N"},
	{dictType: "sys_common_status", sort: 2, label: "失败", value: "1", listClass: "danger", isDefault: "N"},
}

// dictTypeSeeds 字典类型种子
var dictTypeSeeds = []struct {
	name string
	typ  string
}{
	{name: "用户性别", typ: "sys_user_sex"},
	{name: "菜单状态", typ: "sys_show_hide"},
	{name: "系统开关", typ: "sys_normal_disable"},
	{name: "任务状态", typ: "sys_job_status"},
	{name: "系统是否", typ: "sys_yes_no"},
	{name: "通知类型", typ: "sys_notice_type"},
	{name: "通知状态", typ: "sys_notice_status"},
	{name: "操作状态", typ: "sys_common_status"},
}

// Ensure 幂等初始化基础数据
// @author: [bluefrog](https://github.com/freewu)
// @function: Ensure
// @description: 按表判断并补齐部门/岗位/角色/用户/菜单/参数/字典等基础数据，已存在的数据不会重复写入
// @return: err error
func (s Seed) Ensure() error {
	log.Info("开始初始化基础数据……")
	err := domain.DB.Transaction(func(tx *gorm.DB) error {
		for _, step := range []func(*gorm.DB) error{
			s.seedDepartments,
			s.seedPosts,
			s.seedRoles,
			s.seedUsers,
			s.seedUserRelations,
			s.seedMenus,
			s.seedRoleMenus,
			s.seedConfigs,
			s.seedDictTypes,
			s.seedDictData,
			s.seedJobs,
		} {
			if err := step(tx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Errorf("基础数据初始化失败：%v", err)
		return err
	}
	log.Info("基础数据初始化完成")
	return nil
}

// empty 判断表是否为空（为空才写入种子数据，保证幂等）
func (s Seed) empty(tx *gorm.DB, model interface{}) (bool, error) {
	var count int64
	if err := tx.Model(model).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

// seedDepartments 初始化部门数据
func (s Seed) seedDepartments(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.Department{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	seeds := []struct {
		id        uint
		parentID  uint
		ancestors string
		name      string
		sort      uint
		leader    string
		phone     string
		email     string
	}{
		{id: 100, parentID: 0, ancestors: "0", name: "若依科技", sort: 0, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 101, parentID: 100, ancestors: "0,100", name: "深圳总公司", sort: 1, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 102, parentID: 100, ancestors: "0,100", name: "长沙分公司", sort: 2, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 103, parentID: 101, ancestors: "0,100,101", name: "研发部门", sort: 1, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 104, parentID: 101, ancestors: "0,100,101", name: "市场部门", sort: 2, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 105, parentID: 101, ancestors: "0,100,101", name: "测试部门", sort: 3, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 106, parentID: 101, ancestors: "0,100,101", name: "财务部门", sort: 4, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 107, parentID: 101, ancestors: "0,100,101", name: "运维部门", sort: 5, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 108, parentID: 102, ancestors: "0,100,102", name: "市场部门", sort: 1, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
		{id: 109, parentID: 102, ancestors: "0,100,102", name: "财务部门", sort: 2, leader: "若依", phone: "15888888888", email: "ry@qq.com"},
	}
	rows := make([]domain.Department, 0, len(seeds))
	for _, item := range seeds {
		rows = append(rows, domain.Department{
			ID: item.id,
			DepartmentBase: domain.DepartmentBase{
				ParentID:  item.parentID,
				Ancestors: item.ancestors,
				Name:      item.name,
				Sort:      item.sort,
				Status:    0,
				Leader:    item.leader,
				Phone:     item.phone,
				EMail:     item.email,
				RuoyiModel: core.RuoyiModel{
					CreateTime: now,
					UpdateTime: now,
					CreateBy:   "system",
					UpdateBy:   "system",
				},
			},
		})
	}
	return tx.Create(&rows).Error
}

// seedPosts 初始化岗位数据
func (s Seed) seedPosts(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.Post{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	seeds := []struct {
		id   uint
		code string
		name string
		sort uint
	}{
		{id: 1, code: "ceo", name: "董事长", sort: 1},
		{id: 2, code: "se", name: "项目经理", sort: 2},
		{id: 3, code: "hr", name: "人力资源", sort: 3},
		{id: 4, code: "user", name: "普通员工", sort: 4},
	}
	rows := make([]domain.Post, 0, len(seeds))
	for _, item := range seeds {
		rows = append(rows, domain.Post{
			ID: item.id,
			PostBase: domain.PostBase{
				Code:   item.code,
				Name:   item.name,
				Sort:   item.sort,
				Status: 0,
				RuoyiModel: core.RuoyiModel{
					CreateTime: now,
					UpdateTime: now,
					CreateBy:   "system",
					UpdateBy:   "system",
				},
			},
		})
	}
	return tx.Create(&rows).Error
}

// seedRoles 初始化角色数据
func (s Seed) seedRoles(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.Role{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	rows := []domain.Role{
		{
			ID: 1,
			RoleBase: domain.RoleBase{
				Key:                     "admin",
				Name:                    "超级管理员",
				Sort:                    1,
				Status:                  0,
				DataScope:               1,
				MenuCheckStrictly:       1,
				DepartmentCheckStrictly: 1,
				DeleteFlag:              0,
				RuoyiModel: core.RuoyiModel{
					CreateTime: now, UpdateTime: now, CreateBy: "system", UpdateBy: "system",
					Remark: "超级管理员",
				},
			},
		},
		{
			ID: 2,
			RoleBase: domain.RoleBase{
				Key:                     "common",
				Name:                    "普通角色",
				Sort:                    2,
				Status:                  0,
				DataScope:               2,
				MenuCheckStrictly:       1,
				DepartmentCheckStrictly: 1,
				DeleteFlag:              0,
				RuoyiModel: core.RuoyiModel{
					CreateTime: now, UpdateTime: now, CreateBy: "system", UpdateBy: "system",
					Remark: "普通角色",
				},
			},
		},
	}
	return tx.Create(&rows).Error
}

// seedUsers 初始化内置账号（密码为 bcrypt 加密后的 admin123）
func (s Seed) seedUsers(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.User{})
	if err != nil || !empty {
		return err
	}
	password, err := lib.HashPassword(seedInitPassword)
	if err != nil {
		return err
	}
	rows := []domain.User{
		{
			Model: core.Model{ID: 1},
			UserBase: domain.UserBase{
				Username:     "admin",
				Mobile:       "15888888888",
				Nickname:     "超级管理员",
				Status:       1,
				Email:        "admin@ruoyi.com",
				Gender:       0,
				DepartmentID: 103,
				Remark:       "超级管理员账号",
				IsAdmin:      true,
			},
			Password: password,
		},
		{
			Model: core.Model{ID: 2},
			UserBase: domain.UserBase{
				Username:     "ry",
				Mobile:       "15666666666",
				Nickname:     "若依",
				Status:       1,
				Email:        "ry@ruoyi.com",
				Gender:       0,
				DepartmentID: 105,
				Remark:       "普通角色账号",
				IsAdmin:      false,
			},
			Password: password,
		},
	}
	return tx.Create(&rows).Error
}

// seedUserRelations 初始化用户角色/岗位关联
func (s Seed) seedUserRelations(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.UserRole{})
	if err != nil || !empty {
		return err
	}
	if err = tx.Create(&[]domain.UserRole{{UserID: 1, RoleID: 1}, {UserID: 2, RoleID: 2}}).Error; err != nil {
		return err
	}
	empty, err = s.empty(tx, &domain.UserPost{})
	if err != nil || !empty {
		return err
	}
	return tx.Create(&[]domain.UserPost{{UserID: 1, PostID: 1}, {UserID: 2, PostID: 4}}).Error
}

// seedMenus 初始化菜单数据
func (s Seed) seedMenus(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.Menu{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	for _, item := range menuSeeds {
		visible := item.visible
		if visible == "" {
			visible = "0"
		}
		isFrame := uint(1) // 1否（非外链）
		if item.external {
			isFrame = 0 // 0是（外链）
		}
		// 用 map 写入：is_frame 列默认 1，结构体方式 Create 会把显式传入的 0 交给数据库默认值
		record := map[string]interface{}{
			"ID":         item.id,
			"Name":       item.name,
			"ParentID":   item.parentID,
			"Sort":       item.sort,
			"Path":       item.path,
			"Component":  item.component,
			"MenuType":   item.menuType,
			"Perms":      item.perms,
			"Icon":       item.icon,
			"IsFrame":    isFrame,
			"IsCache":    0,
			"Visible":    visible,
			"Status":     0,
			"CreateBy":   "system",
			"CreateTime": now,
			"UpdateBy":   "system",
			"UpdateTime": now,
		}
		if err := tx.Model(&domain.Menu{}).Create(record).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedRoleMenus 初始化角色菜单关联（超级管理员全部菜单、普通角色部分菜单）
func (s Seed) seedRoleMenus(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.RoleMenu{})
	if err != nil || !empty {
		return err
	}
	rows := make([]domain.RoleMenu, 0, len(menuSeeds)+len(commonRoleMenuIds))
	for _, menu := range menuSeeds {
		rows = append(rows, domain.RoleMenu{RoleID: 1, MenuID: menu.id})
	}
	for _, menuID := range commonRoleMenuIds {
		rows = append(rows, domain.RoleMenu{RoleID: 2, MenuID: menuID})
	}
	return tx.Create(&rows).Error
}

// seedConfigs 初始化系统参数
func (s Seed) seedConfigs(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.Config{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	seeds := []struct {
		name  string
		key   string
		value string
	}{
		{name: "主框架页-默认皮肤样式名称", key: "sys.index.skinName", value: "skin-blue"},
		{name: "用户管理-账号初始密码", key: "sys.user.initPassword", value: seedInitPassword},
		{name: "主框架页-侧边栏主题", key: "sys.index.sideTheme", value: "theme-dark"},
		{name: "账号自助-是否开启用户注册功能", key: "sys.account.registerUser", value: "false"},
		{name: "用户登录-验证码开关", key: "sys.account.captchaEnabled", value: "false"},
		{name: "用户登录-黑名单列表", key: "sys.login.blackIPList", value: ""},
	}
	rows := make([]domain.Config, 0, len(seeds))
	for _, item := range seeds {
		rows = append(rows, domain.Config{
			ConfigBase: domain.ConfigBase{
				Name:  item.name,
				Key:   item.key,
				Value: item.value,
				Type:  "Y",
				RuoyiModel: core.RuoyiModel{
					CreateTime: now, UpdateTime: now, CreateBy: "system", UpdateBy: "system",
					Remark: "系统内置参数",
				},
			},
		})
	}
	return tx.Create(&rows).Error
}

// seedJobs 初始化定时任务示例数据
func (s Seed) seedJobs(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.Job{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	for _, item := range jobSeeds {
		// 使用 map 写入：status=0/concurrent="0" 等零值会被 GORM 的 default 标签忽略
		record := map[string]interface{}{
			"Name":           item.name,
			"Group":          item.group,
			"Executor":       item.executor,
			"InvokeTarget":   item.invokeTarget,
			"JobArgs":        item.jobArgs,
			"JobKwargs":      "",
			"CronExpression": item.cronExpression,
			"MisfirePolicy":  item.misfirePolicy,
			"Concurrent":     item.concurrent,
			"Status":         item.status,
			"Remark":         "系统内置任务",
			"CreateBy":       "system",
			"CreateTime":     now,
			"UpdateBy":       "system",
			"UpdateTime":     now,
		}
		if err := tx.Model(&domain.Job{}).Create(record).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedDictTypes 初始化字典类型
func (s Seed) seedDictTypes(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.DictType{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	rows := make([]domain.DictType, 0, len(dictTypeSeeds))
	for _, item := range dictTypeSeeds {
		rows = append(rows, domain.DictType{
			DictTypeBase: domain.DictTypeBase{
				Name:   item.name,
				Type:   item.typ,
				Status: 0,
				RuoyiModel: core.RuoyiModel{
					CreateTime: now, UpdateTime: now, CreateBy: "system", UpdateBy: "system",
					Remark: "系统内置字典",
				},
			},
		})
	}
	return tx.Create(&rows).Error
}

// seedDictData 初始化字典数据
func (s Seed) seedDictData(tx *gorm.DB) error {
	empty, err := s.empty(tx, &domain.DictData{})
	if err != nil || !empty {
		return err
	}
	now := time.Now()
	rows := make([]domain.DictData, 0, len(dictDataSeeds))
	for _, item := range dictDataSeeds {
		rows = append(rows, domain.DictData{
			DictDataBase: domain.DictDataBase{
				Sort:      item.sort,
				Label:     item.label,
				Value:     item.value,
				Type:      item.dictType,
				ListClass: item.listClass,
				IsDefault: item.isDefault,
				Status:    0,
				RuoyiModel: core.RuoyiModel{
					CreateTime: now, UpdateTime: now, CreateBy: "system", UpdateBy: "system",
					Remark: "系统内置字典数据",
				},
			},
		})
	}
	return tx.Create(&rows).Error
}
