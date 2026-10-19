package domain

// Models 当前模块（系统管理）需要自动迁移的表模型
func Models() []interface{} {
	return []interface{}{
		User{},       // 后台用户表
		Department{}, // 部门表
		Post{},       // 职位表
		Notice{},     // 公告表
		Config{},     // 参数表
		Role{},       // 角色表
		Menu{},       // 菜单表
		DictType{},   // 字典类型表
		DictData{},   // 字典数据表
		RoleDept{},   // 角色部门关联表
		RoleMenu{},   // 角色菜单关联表
		UserPost{},   // 用户岗位关联表
		UserRole{},   // 用户角色关联表
	}
}
