// Command encrypt 加密 / 解密配置中的敏感值（AES-256-GCM）。
//
// 通常通过 just 调用：
//
//	just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'
//	just decrypt 'ENC(xxxxx)'
//	just encrypt --env 'root:123456@tcp(...)/admintpl?...'   # 直接输出可粘贴到 .env 的片段
//
// 密钥来源：-k 参数 > 环境变量 CONFIG_SECRET_KEY > .env 中的 CONFIG_SECRET_KEY。
//
// 说明：本命令刻意不依赖 common/config（避免「配置里已有密文但密钥还没配好」时无法生成密文），
// 仅用极简的 .env 解析读取密钥。
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bluefroog/admintpl/common/secret"
)

func main() {
	var (
		encryptMode = flag.Bool("e", false, "加密模式：把明文加密为 ENC(...) 密文")
		decryptMode = flag.Bool("d", false, "解密模式：把 ENC(...) 密文还原为明文")
		envMode     = flag.Bool("env", false, "在加密结果之外输出可直接粘贴到 .env 的配置片段")
		key         = flag.String("k", "", "加密密钥（默认取环境变量 CONFIG_SECRET_KEY，其次取 .env）")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法：\n"+
			"  encrypt -e '明文'        加密（输出 ENC(...)）\n"+
			"  encrypt -d 'ENC(...)'   解密\n"+
			"  encrypt -env -e '明文'   输出 .env 配置片段\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	input := flag.Arg(0)
	if input == "" || (*encryptMode && *decryptMode) {
		flag.Usage()
		os.Exit(2)
	}

	secretKey := resolveKey(*key)
	if secretKey == "" {
		fail(fmt.Errorf("未找到加密密钥：请设置环境变量 CONFIG_SECRET_KEY（或写入 .env），或用 -k 指定"))
	}

	switch {
	case *decryptMode:
		plain, err := secret.Decrypt(input, secretKey)
		if err != nil {
			fail(err)
		}
		fmt.Println(plain)
	case *encryptMode || *envMode:
		token, err := secret.Encrypt(input, secretKey)
		if err != nil {
			fail(err)
		}
		if *envMode {
			fmt.Printf("# 配置加密（把下面三行写入 .env；密钥务必与部署环境保持一致）\n")
			fmt.Printf("CONFIG_ENCRYPT_ENABLED=true\n")
			fmt.Printf("CONFIG_SECRET_KEY=%s\n", secretKey)
			fmt.Printf("DB_DSN=%s\n", token)
			return
		}
		fmt.Println(token)
	default:
		flag.Usage()
		os.Exit(2)
	}
}

// resolveKey 取加密密钥：-k 参数 > 环境变量 > .env 文件
func resolveKey(flagKey string) string {
	if flagKey != "" {
		return flagKey
	}
	if key := os.Getenv("CONFIG_SECRET_KEY"); key != "" {
		return key
	}
	return dotEnvValue(envFilePath(), "CONFIG_SECRET_KEY")
}

// envFilePath 返回 .env 路径，可用 ENV_FILE 指定（与 common/config 保持一致）
func envFilePath() string {
	if file := os.Getenv("ENV_FILE"); file != "" {
		return file
	}
	return ".env"
}

// dotEnvValue 从 .env 中读取指定键（与 common/config.loadDotEnv 的解析规则一致）
func dotEnvValue(path, want string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		name, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(name) != want {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "加密处理失败: %s\n", err)
	os.Exit(1)
}
