package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bluefroog/admintpl/common/secret"
	"github.com/spf13/viper"
)

var (
	v *viper.Viper
)

// envPlaceholder 匹配配置文件中的环境变量占位符，支持默认值语法：
//
//	${KEY}         取环境变量 KEY
//	${KEY:-value}  环境变量 KEY 不存在时使用 value
var envPlaceholder = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

func init() {
	// 读取 .env 文件（可选），把变量注入进程环境，供配置文件的 ${KEY} 占位符引用。
	// 已存在的系统环境变量优先，不会被 .env 覆盖。
	loadDotEnv(envFilePath())

	// todo 后面可以支持 consul 或 ectd
	// 读取yaml文件
	v = viper.New()
	//v.Debug()

	configName := os.Getenv("CONFIG_NAME")
	if configName == "" { // 如果全局变量没有配置,使用默认的配置文件
		configName = "config" // 默认配置文件名称
	}

	// 设置读取的配置文件名
	v.SetConfigName(configName)

	v.AddConfigPath("./config") // windows环境下为%GOPATH，linux环境下为$GOPATH
	// 设置配置文件类型
	v.SetConfigType("yaml")

	// 加载配置文件
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("config read err: %s\n", err)
		return
	}

	// 展开配置文件中的 ${ENV} 占位符，实现配置与环境的解耦
	expandAll()

	// 解密配置中的敏感值（AES-256-GCM / SM4-CBC，由 secret.algorithm 选择）。
	// 所有中间件（数据库等）读取到的都是明文，因此连接参数无需各自处理解密。
	if err := DecryptAll(v, Bool(KeySecretEnabled), String(KeySecretKey), String(KeySecretAlgorithm)); err != nil {
		fmt.Fprintf(os.Stderr, "config decrypt err: %s\n", err)
		os.Exit(1) // 安全类配置错误必须快速失败，避免用密文/错误参数继续启动
	}
}

// envFilePath 返回 .env 文件路径，可通过 ENV_FILE 指定
func envFilePath() string {
	if file := os.Getenv("ENV_FILE"); file != "" {
		return file
	}
	return ".env"
}

// loadDotEnv 解析 .env 文件（KEY=VALUE），支持 # 注释、空行、export 前缀与引号包裹
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // .env 为可选项，缺失时只使用系统环境变量与配置默认值
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

// expandAll 用环境变量替换配置中的占位符
func expandAll() {
	for _, key := range v.AllKeys() {
		value, ok := v.Get(key).(string)
		if !ok || !strings.Contains(value, "${") {
			continue
		}
		v.Set(key, expandEnv(value))
	}
}

// expandEnv 展开字符串中的 ${KEY} / ${KEY:-默认值} 占位符
func expandEnv(s string) string {
	return envPlaceholder.ReplaceAllStringFunc(s, func(match string) string {
		groups := envPlaceholder.FindStringSubmatch(match)
		if value, exists := os.LookupEnv(groups[1]); exists {
			return value
		}
		return groups[2] // 未设置时使用默认值（未写默认值则为空字符串）
	})
}

// 配置加密相关的配置项（值为空时表示不加密，兼容历史明文配置）
const (
	// KeySecretEnabled 是否启用配置敏感值加密（secret.enabled / CONFIG_ENCRYPT_ENABLED）
	KeySecretEnabled = "secret.enabled"
	// KeySecretKey 加密密钥（secret.key / CONFIG_SECRET_KEY）
	KeySecretKey = "secret.key"
	// KeySecretAlgorithm 加密算法（secret.algorithm / CONFIG_SECRET_ALGORITHM）：AES / SM4
	KeySecretAlgorithm = "secret.algorithm"

	// EnvSecretEnabled / EnvSecretKey / EnvSecretAlgorithm 环境变量名（.env / 容器环境变量注入）
	EnvSecretEnabled   = "CONFIG_ENCRYPT_ENABLED"
	EnvSecretKey       = "CONFIG_SECRET_KEY"
	EnvSecretAlgorithm = "CONFIG_SECRET_ALGORITHM"
)

// DecryptAll 解密配置中所有含 ENC(...) 片段的值，就地写回配置对象：
//   - enabled=false 且未出现密文：全部按明文处理（向后兼容）；
//   - enabled=false 但出现密文：报错，避免把密文当作明文去连数据库；
//   - enabled=true：密钥缺失或解密失败即报错（快速失败）。
//
// 支持整体加密（ENC(xxx)）与局部加密（root:ENC(xxx)@tcp(127.0.0.1:3306)/db）。
func DecryptAll(conf *viper.Viper, enabled bool, key, algorithm string) error {
	// 算法先做归一化校验：非法值即使没用到密文也应尽早失败
	algo, err := secret.NormalizeAlgorithm(algorithm)
	if err != nil {
		return fmt.Errorf("配置项 %s 非法: %w", KeySecretAlgorithm, err)
	}
	_ = algo // 解密时算法取自密文标签，这里仅做校验（保证切换算法后历史密文仍可解）

	for _, name := range conf.AllKeys() {
		if name == KeySecretKey {
			continue // 密钥本身不参与解密
		}

		value, ok := conf.Get(name).(string)
		if !ok || !secret.IsEncrypted(value) {
			continue
		}

		if !enabled {
			return fmt.Errorf("配置项 %s 使用了加密值，但未开启配置加密：请设置 %s=true（secret.enabled）并配置 %s",
				name, EnvSecretEnabled, EnvSecretKey)
		}

		plain, err := secret.Resolve(value, key)
		if err != nil {
			return fmt.Errorf("配置项 %s 解密失败: %w", name, err)
		}
		conf.Set(name, plain)
	}

	return nil
}

// 返回字符类型配置
func String(key string) string {
	return v.GetString(key)
}

// 返回Int类型配置
func Int(key string) int {
	return v.GetInt(key)
}

// 返回布尔类型配置
func Bool(key string) bool {
	return v.GetBool(key)
}

// 判断配置项是否存在（用于区分「未配置」与「配置为空/零值」）
func IsSet(key string) bool {
	return v.IsSet(key)
}

// 返回 map[string]interface{} 配置
func Map(key string) map[string]interface{} {
	return v.GetStringMap(key)
}

// 返回 字符型数组 配置
func StringArray(key string) []string {
	return v.GetStringSlice(key)
}

// 返回 整型数组 配置
func IntArray(key string) []int {
	return v.GetIntSlice(key)
}

// 返回 浮点型 配置
func Float64(key string) float64 {
	return v.GetFloat64(key)
}
