package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("TEST_DSN", "root:123456@tcp(127.0.0.1:3306)/demo")
	t.Setenv("TEST_EMPTY", "")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"无占位符", ":8080", ":8080"},
		{"取环境变量", "${TEST_DSN}", "root:123456@tcp(127.0.0.1:3306)/demo"},
		{"混合文本", "prefix-${TEST_DSN}-suffix", "prefix-root:123456@tcp(127.0.0.1:3306)/demo-suffix"},
		{"未设置取默认值", "${TEST_NOT_EXIST:-10}", "10"},
		{"未设置无默认值为空", "${TEST_NOT_EXIST}", ""},
		{"已设置为空不取默认值", "${TEST_EMPTY:-10}", ""},
		{"默认值含冒号", "${TEST_NOT_EXIST:-:8080}", ":8080"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := expandEnv(c.in); got != c.want {
				t.Fatalf("expandEnv(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestLoadDotEnv(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), ".env")
	if err != nil {
		t.Fatal(err)
	}
	content := "# comment\n" +
		"\n" +
		"TEST_DOTENV_A=1\n" +
		"export TEST_DOTENV_B=hello\n" +
		"TEST_DOTENV_C=\"quoted value\"\n" +
		"TEST_DOTENV_D=\n"
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
	file.Close()

	loadDotEnv(file.Name())

	want := map[string]string{
		"TEST_DOTENV_A": "1",
		"TEST_DOTENV_B": "hello",
		"TEST_DOTENV_C": "quoted value",
		"TEST_DOTENV_D": "",
	}
	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Fatalf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestLoadDotEnvNotOverride(t *testing.T) {
	t.Setenv("TEST_DOTENV_KEEP", "system")

	file, err := os.CreateTemp(t.TempDir(), ".env")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("TEST_DOTENV_KEEP=dotenv\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	loadDotEnv(file.Name())

	if got := os.Getenv("TEST_DOTENV_KEEP"); got != "system" {
		t.Fatalf("系统环境变量被 .env 覆盖: got %q, want %q", got, "system")
	}
}

// TestConfigFileExpansion 校验根目录 config/config.yaml 的占位符可以正常展开与解析
func TestConfigFileExpansion(t *testing.T) {
	envFile := filepath.Join("..", "..", ".env")
	yamlFile := filepath.Join("..", "..", "config", "config.yaml")
	if _, err := os.Stat(yamlFile); err != nil {
		t.Skip("config/config.yaml 不存在，跳过")
	}
	if _, err := os.Stat(envFile); err == nil {
		loadDotEnv(envFile)
	}

	data, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatal(err)
	}
	expanded := expandEnv(string(data))
	if strings.Contains(expanded, "${") {
		t.Fatalf("配置中仍有未展开的占位符:\n%s", expanded)
	}

	conf := viper.New()
	conf.SetConfigType("yaml")
	if err := conf.ReadConfig(strings.NewReader(expanded)); err != nil {
		t.Fatalf("解析展开后的配置失败: %v", err)
	}
	if conf.GetString("service.port") == "" {
		t.Fatal("service.port 为空")
	}
	if conf.GetString("database.admin.dsn") == "" {
		t.Fatal("database.admin.dsn 为空")
	}
	if conf.GetInt("database.admin.max-idle") <= 0 {
		t.Fatalf("database.admin.max-idle 解析失败: %d", conf.GetInt("database.admin.max-idle"))
	}
}
