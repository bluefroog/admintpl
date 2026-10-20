package bootstrap

import (
	"errors"
	"log"
	"strings"

	"github.com/bluefroog/admintpl/common/config"
	"github.com/bluefroog/admintpl/common/s3"
	"github.com/gin-gonic/gin"
)

// MountLocalStatic 本地目录后端：把上传目录挂成静态路由，使 s3.URL() 返回的地址可直接访问
// 仅当 s3.enabled=true 且驱动为 local 时生效；云存储后端由各服务的访问域名/CDN 对外提供，无需本地路由
// @param: r *gin.Engine gin 引擎
func MountLocalStatic(r *gin.Engine) {
	st, err := s3.New()
	if err != nil {
		if !errors.Is(err, s3.ErrDisabled) {
			log.Printf("文件存储初始化失败，跳过本地上传目录静态路由: %v", err)
		}
		return
	}
	defer func() { _ = st.Close() }()

	dir, ok := s3.LocalDirectory(st)
	if !ok {
		return // 非本地目录后端
	}
	prefix := strings.TrimSpace(config.String("s3.local.url-prefix"))
	if prefix == "" {
		prefix = s3.DefaultLocalURLPrefix
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	r.Static(prefix, dir)
	log.Printf("本地上传目录已挂载静态路由: %s -> %s", prefix, dir)
}
