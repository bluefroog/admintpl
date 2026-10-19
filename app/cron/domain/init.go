package domain

// Models 当前模块（定时任务）需要自动迁移的表模型
func Models() []interface{} {
	return []interface{}{
		Job{},    // 定时任务表
		JobLog{}, // 定时任务日志表
	}
}
