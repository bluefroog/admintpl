package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

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
