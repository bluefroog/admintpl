package main

import (
	"testing"

	"github.com/bluefroog/admintpl/common/config"
)

// TestConfigLoad 以项目根目录为工作目录，校验 config.yaml + .env 的真实加载与占位符展开
func TestConfigLoad(t *testing.T) {
	if got := config.String("service.port"); got == "" {
		t.Fatal("service.port 为空，配置未加载或占位符未展开")
	}
	if got := config.String("database.admin.type"); got == "" {
		t.Fatal("database.admin.type 为空")
	}
	if got := config.String("database.admin.dsn"); got == "" {
		t.Fatal("database.admin.dsn 为空")
	}
	if config.Int("database.admin.max-idle") <= 0 {
		t.Fatal("database.admin.max-idle 解析失败")
	}
}
