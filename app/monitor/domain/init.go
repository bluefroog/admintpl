package domain

// Models 当前模块（系统监控）需要自动迁移的表模型
func Models() []interface{} {
	return []interface{}{
		UserOnline{},   // 用户在线状态表
		LoginLog{},     // 登录日志表
		OperationLog{}, // 操作日志表
	}
}
