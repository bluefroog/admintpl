package controller

import "github.com/gin-gonic/gin"

func InitRouter(r *gin.Engine) {
	group := r.Group("/system")
	{
		// 用户相关接口
		user := User{}
		group.GET("/user/list", user.List)
		group.POST("/user/create", user.Create)
		group.PUT("/user/update", user.Update)
		group.DELETE("/user/delete", user.Delete)
		group.GET("/user/detail", user.Detail)

		// 部门相关接口
		department := Department{}
		group.GET("/department/list", department.List)
		group.POST("/department/create", department.Create)
		group.PUT("/department/update", department.Update)
		group.DELETE("/department/delete", department.Delete)
		group.GET("/department/detail", department.Detail)

		// 职位相关接口
		post := Post{}
		group.GET("/post/list", post.List)
		group.POST("/post/create", post.Create)
		group.PUT("/post/update", post.Update)
		group.DELETE("/post/delete", post.Delete)
		group.GET("/post/detail", post.Detail)

		// 公告相关接口
		notice := Notice{}
		group.GET("/notice/list", notice.List)
		group.POST("/notice/create", notice.Create)
		group.PUT("/notice/update", notice.Update)
		group.DELETE("/notice/delete", notice.Delete)
		group.GET("/notice/detail", notice.Detail)

		// 系统参数相关接口
		config := Config{}
		group.GET("/config/list", config.List)
		group.POST("/config/create", config.Create)
		group.PUT("/config/update", config.Update)
		group.DELETE("/config/delete", config.Delete)
		group.GET("/config/detail", config.Detail)

		// 访问日志相关接口
		loginLog := LoginLog{}
		group.GET("/login-log/list", loginLog.List)
		group.DELETE("/login-log/delete", loginLog.Delete)
		group.GET("/login-log/detail", loginLog.Detail)

		// 角色相关接口
		role := Role{}
		group.GET("/role/list", role.List)
		group.POST("/role/create", role.Create)
		group.PUT("/role/update", role.Update)
		group.DELETE("/role/delete", role.Delete)
		group.GET("/role/detail", role.Detail)

		// 菜单相关接口
		menu := Menu{}
		group.GET("/menu/list", menu.List)
		group.POST("/menu/create", menu.Create)
		group.PUT("/menu/update", menu.Update)
		group.DELETE("/menu/delete", menu.Delete)
		group.GET("/menu/detail", menu.Detail)

		// 字典类型相关接口
		dictType := DictType{}
		group.GET("/dict-type/list", dictType.List)
		group.POST("/dict-type/create", dictType.Create)
		group.PUT("/dict-type/update", dictType.Update)
		group.DELETE("/dict-type/delete", dictType.Delete)
		group.GET("/dict-type/detail", dictType.Detail)

		// 字典数据相关接口
		dictData := DictData{}
		group.GET("/dict-data/list", dictData.List)
		group.POST("/dict-data/create", dictData.Create)
		group.PUT("/dict-data/update", dictData.Update)
		group.DELETE("/dict-data/delete", dictData.Delete)
		group.GET("/dict-data/detail", dictData.Detail)

		// 定时任务相关接口
		job := Job{}
		group.GET("/job/list", job.List)
		group.POST("/job/create", job.Create)
		group.PUT("/job/update", job.Update)
		group.DELETE("/job/delete", job.Delete)
		group.GET("/job/detail", job.Detail)

		// 定时任务日志相关接口
		jobLog := JobLog{}
		group.GET("/job-log/list", jobLog.List)
		group.DELETE("/job-log/delete", jobLog.Delete)
		group.GET("/job-log/detail", jobLog.Detail)

		// 操作日志相关接口
		operationLog := OperationLog{}
		group.GET("/operation-log/list", operationLog.List)
		group.DELETE("/operation-log/delete", operationLog.Delete)
		group.GET("/operation-log/detail", operationLog.Detail)

		// 角色部门关联相关接口
		roleDept := RoleDept{}
		group.GET("/role-dept/list", roleDept.List)
		group.POST("/role-dept/create", roleDept.Create)
		group.DELETE("/role-dept/delete", roleDept.Delete)

		// 角色菜单关联相关接口
		roleMenu := RoleMenu{}
		group.GET("/role-menu/list", roleMenu.List)
		group.POST("/role-menu/create", roleMenu.Create)
		group.DELETE("/role-menu/delete", roleMenu.Delete)

		// 用户岗位关联相关接口
		userPost := UserPost{}
		group.GET("/user-post/list", userPost.List)
		group.POST("/user-post/create", userPost.Create)
		group.DELETE("/user-post/delete", userPost.Delete)

		// 用户角色关联相关接口
		userRole := UserRole{}
		group.GET("/user-role/list", userRole.List)
		group.POST("/user-role/create", userRole.Create)
		group.DELETE("/user-role/delete", userRole.Delete)

	}
	//r.GET("/ping", func(c *gin.Context) {
	//	c.JSON(200, gin.H{
	//		"message": "pong",
	//	})
	//})
}
