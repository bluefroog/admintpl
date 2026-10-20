package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bluefroog/admintpl/common/secret"
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
	if conf.GetString("database.connections.admin.dsn") == "" {
		t.Fatal("database.connections.admin.dsn 为空")
	}
	if conf.GetInt("database.connections.admin.max-idle") <= 0 {
		t.Fatalf("database.connections.admin.max-idle 解析失败: %d", conf.GetInt("database.connections.admin.max-idle"))
	}

	// 代码读取的配置项必须全部存在于配置文件中：
	// 防止「代码读 system.auth.enabled、配置写在 auth.enabled」这类键名不一致导致开关静默失效
	required := []string{
		"secret.enabled", "secret.key", "secret.algorithm",
		"service.port", "service.debug",
		"database.default",
		"database.connections.admin.type", "database.connections.admin.dsn", "database.connections.admin.prefix",
		"database.connections.admin.singular",
		"database.connections.admin.max-idle", "database.connections.admin.max-open", "database.connections.admin.max-lifetime",
		"database.connections.admin.debug",
		"system.auth.enabled", "system.auth.token-expire", "system.auth.max-retry",
		"system.auth.lock-time", "system.auth.permission-enabled",
		"system.captcha.enabled", "system.captcha.length", "system.captcha.expire",
		"system.seed.enabled",
		"system.job.enabled", "system.job.log-retain-days",
		"system.data-scope.enabled",
		"s3.enabled", "s3.driver", "s3.public-base-url",
		"s3.local.dir", "s3.local.url-prefix",
		"s3.minio.endpoint", "s3.minio.access-key", "s3.minio.secret-key", "s3.minio.bucket",
		"s3.minio.region", "s3.minio.secure", "s3.minio.path-style",
		"s3.rustfs.endpoint", "s3.rustfs.access-key", "s3.rustfs.secret-key", "s3.rustfs.bucket",
		"s3.rustfs.region", "s3.rustfs.secure", "s3.rustfs.path-style",
		"s3.cos.bucket-url", "s3.cos.bucket", "s3.cos.region", "s3.cos.secret-id",
		"s3.cos.secret-key", "s3.cos.domain",
		"s3.oss.endpoint", "s3.oss.access-key-id", "s3.oss.access-key-secret",
		"s3.oss.bucket", "s3.oss.domain",
		"cache.enabled", "cache.driver", "cache.prefix",
		"cache.file.dir", "cache.file.flush-interval",
		"cache.memcache.servers", "cache.memcache.timeout",
		"cache.redis.mode", "cache.redis.addr", "cache.redis.addrs",
		"cache.redis.cluster-addrs", "cache.redis.master-name",
		"cache.redis.sentinel-addrs", "cache.redis.sentinel-password",
		"cache.redis.username", "cache.redis.password",
		"cache.redis.db", "cache.redis.timeout", "cache.redis.pool-size",
		"system.gen.package-name", "system.gen.module-name", "system.gen.author",
		"system.gen.auto-remove-pre", "system.gen.table-prefix",
	}
	for _, key := range required {
		if !conf.IsSet(key) {
			t.Errorf("配置文件缺少代码使用的配置项: %s", key)
		}
	}

	// 配置项存在性判断（IsSet 用于区分「未配置」与「显式配置为 false」）
	if !conf.IsSet("system.data-scope.enabled") {
		t.Error("已配置的配置项 IsSet 应为 true")
	}
	if conf.IsSet("system.not-exist.key") {
		t.Error("未配置的配置项 IsSet 应为 false")
	}

	// 开关型配置必须能解析为布尔值（值写错时 viper 会静默返回 false）
	bools := []string{
		"secret.enabled",
		"service.debug", "database.connections.admin.debug", "system.auth.enabled",
		"system.auth.permission-enabled", "system.captcha.enabled", "system.seed.enabled",
		"system.job.enabled", "system.data-scope.enabled", "cache.enabled",
		"system.gen.auto-remove-pre",
		"s3.enabled", "s3.minio.secure", "s3.minio.path-style",
		"s3.rustfs.secure", "s3.rustfs.path-style",
	}
	for _, key := range bools {
		value := strings.ToLower(conf.GetString(key))
		if value != "true" && value != "false" {
			t.Errorf("开关型配置 %s 取值非法（应为 true/false）: %q", key, conf.GetString(key))
		}
	}
}

// TestDecryptAll 校验配置敏感值的解密行为（开关、密钥、整体/局部加密、错误场景）
func TestDecryptAll(t *testing.T) {
	const key = "unit-test-key"
	encryptedDSN, err := secret.Encrypt("root:p@ss@tcp(127.0.0.1:3306)/admintpl", key, secret.AlgorithmAES)
	if err != nil {
		t.Fatal(err)
	}
	encryptedPassword, err := secret.Encrypt("p@ss", key, secret.AlgorithmAES)
	if err != nil {
		t.Fatal(err)
	}
	sm4DSN, err := secret.Encrypt("root:p@ss@tcp(127.0.0.1:3306)/admintpl", key, secret.AlgorithmSM4)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		settings  map[string]interface{}
		enabled   bool
		key       string
		algorithm string
		want      map[string]string
		wantErr   string
	}{
		{
			name:     "未开启且无密文：按明文处理",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": "root:123456@tcp(db:3306)/admintpl"}}},
			enabled:  false,
			key:      "",
			want:     map[string]string{"database.admin.dsn": "root:123456@tcp(db:3306)/admintpl"},
		},
		{
			name:     "开启后整体解密",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": encryptedDSN}}},
			enabled:  true,
			key:      key,
			want:     map[string]string{"database.admin.dsn": "root:p@ss@tcp(127.0.0.1:3306)/admintpl"},
		},
		{
			name:     "开启后局部解密（只加密密码）",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": "root:" + encryptedPassword + "@tcp(127.0.0.1:3306)/admintpl?parseTime=True"}}},
			enabled:  true,
			key:      key,
			want:     map[string]string{"database.admin.dsn": "root:p@ss@tcp(127.0.0.1:3306)/admintpl?parseTime=True"},
		},
		{
			name:     "明文与密文可混用",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": encryptedDSN, "type": "mysql"}}},
			enabled:  true,
			key:      key,
			want:     map[string]string{"database.admin.dsn": "root:p@ss@tcp(127.0.0.1:3306)/admintpl", "database.admin.type": "mysql"},
		},
		{
			name:     "有密文但未开启：报错提示开启开关",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": encryptedDSN}}},
			enabled:  false,
			key:      key,
			wantErr:  "未开启配置加密",
		},
		{
			name:     "开启但密钥为空：报错",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": encryptedDSN}}},
			enabled:  true,
			key:      "",
			wantErr:  "未配置加密密钥",
		},
		{
			name:     "开启但密钥不匹配：报错",
			settings: map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": encryptedDSN}}},
			enabled:  true,
			key:      "wrong-key",
			wantErr:  "解密失败",
		},
		{
			name:      "SM4 密文可解密（算法由密文标签决定）",
			settings:  map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": sm4DSN}}},
			enabled:   true,
			key:       key,
			algorithm: secret.AlgorithmSM4,
			want:      map[string]string{"database.admin.dsn": "root:p@ss@tcp(127.0.0.1:3306)/admintpl"},
		},
		{
			name:      "切换到 SM4 后仍能解 AES 旧密文",
			settings:  map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": encryptedDSN}}},
			enabled:   true,
			key:       key,
			algorithm: secret.AlgorithmSM4,
			want:      map[string]string{"database.admin.dsn": "root:p@ss@tcp(127.0.0.1:3306)/admintpl"},
		},
		{
			name:      "算法配置非法：直接报错",
			settings:  map[string]interface{}{"database": map[string]interface{}{"admin": map[string]interface{}{"dsn": "root:123456@tcp(db:3306)/admintpl"}}},
			enabled:   true,
			key:       key,
			algorithm: "DES",
			wantErr:   "不支持的配置加密算法",
		},
		{
			name:     "密钥本身不参与解密",
			settings: map[string]interface{}{"secret": map[string]interface{}{"enabled": "true", "key": encryptedPassword}},
			enabled:  true,
			key:      key,
			want:     map[string]string{"secret.key": encryptedPassword},
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			conf := viper.New()
			for k, val := range item.settings {
				conf.Set(k, val)
			}

			err := DecryptAll(conf, item.enabled, item.key, item.algorithm)
			if item.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), item.wantErr) {
					t.Fatalf("错误信息应包含 %q，实际 %v", item.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应报错: %v", err)
			}
			for k, want := range item.want {
				if got := conf.GetString(k); got != want {
					t.Fatalf("%s = %q, want %q", k, got, want)
				}
			}
		})
	}
}

// TestKeys 校验按节点枚举直属子键名（多连接配置按名字读取）
func TestKeys(t *testing.T) {
	saved := v
	v = viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(`
database:
  default: admin
  connections:
    log:
      dsn: ""
    admin:
      dsn: "root:123456@tcp(127.0.0.1:3306)/admintpl"
      prefix: ""
cache:
  driver: file
`)); err != nil {
		t.Fatalf("解析测试配置失败: %v", err)
	}
	t.Cleanup(func() { v = saved })

	got := Keys("database.connections")
	if len(got) != 2 || got[0] != "admin" || got[1] != "log" {
		t.Fatalf("Keys(database.connections) = %v, want [admin log]", got)
	}
	if got = Keys("not-exist"); got != nil {
		t.Fatalf("不存在的节点应返回 nil，实际 %v", got)
	}
	if got = Keys("cache.driver"); got != nil {
		t.Fatalf("标量节点应返回 nil，实际 %v", got)
	}
}
