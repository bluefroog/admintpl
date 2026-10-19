package system

import (
	"github.com/bluefroog/admintpl/app/system/controller"
	"github.com/bluefroog/admintpl/app/system/service"
	"github.com/bluefroog/admintpl/common/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Initialize 系统管理模块初始化（路由注册 + 基础数据 + 在线用户清理）
func Initialize(r *gin.Engine) {
	// 注册路由
	controller.InitRouter(r)
	// 参数/字典缓存开关（未配置时默认开启）
	cacheEnabled := service.ConfigCacheEnabled()
	// 输出关键开关，便于排查“配置未随部署更新”这类问题
	log.Infof("system 初始化：登录认证=%v 权限校验=%v 图形验证码=%v 数据权限=%v 基础数据初始化=%v 参数/字典缓存=%v",
		config.Bool("system.auth.enabled"),
		config.Bool("system.auth.permission-enabled"),
		config.Bool("system.captcha.enabled"),
		config.Bool("system.data-scope.enabled"),
		config.Bool("system.seed.enabled"),
		cacheEnabled,
	)
	// 清理过期的在线用户记录
	service.Auth{}.CleanExpiredOnline()
	// 基础数据初始化（system.seed.enabled=true 时按表补齐，已存在的数据不会重复写入）
	if config.Bool("system.seed.enabled") {
		service.Seed{}.Ensure()
	}
	// 参数 / 字典缓存预热（system.cache.enabled=false 时跳过，每次查询直接回源数据库）
	if cacheEnabled {
		if err, total := (service.Config{}).LoadCache(); err != nil {
			log.Errorf("参数缓存预热失败：%v", err)
		} else {
			log.Infof("参数缓存预热完成：%d 项", total)
		}
		if err, total := (service.DictData{}).LoadCache(); err != nil {
			log.Errorf("字典缓存预热失败：%v", err)
		} else {
			log.Infof("字典缓存预热完成：%d 个字典类型", total)
		}
	}
}
