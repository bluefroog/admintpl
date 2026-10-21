// Command hash 计算用户口令的「两段式 hash」（前端 hash + 后端加盐 hash），用于手工生成入库密码或排查登录问题。
//
// 计算规则与登录接口完全一致（实现在 common/lib/password.go）：
//
//	前端 hash = H_front(明文)              # 算法取 system.password.frontend-hash，经 GET /frontend-config 下发给前端
//	入库密码 = H_back(前端 hash + 盐)      # 算法取 system.password.backend-hash，盐即 user.salt
//
// 通常通过 just 调用：
//
//	just hash 'admin123'                # 自动生成 4 位随机盐
//	just hash 'admin123' 'aB3d'         # 指定盐（复现库中密码 / 校验某条记录）
//	just hash 'admin123' '' sha256 sm3  # 临时指定前端 / 后端算法（覆盖配置）
//	just hash 'admin123' 'aB3d' '' '' admin   # 附带一条可直接执行的 UPDATE SQL（按用户名）
//
// 算法来源：-front / -back 参数 > 环境变量 > .env > config/config.yaml 默认值（sha256）。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bluefroog/admintpl/common/lib"
)

// result 两段式 hash 计算结果
type result struct {
	Plain          string // 明文密码
	FrontAlgorithm string // 前端 hash 算法
	FrontHash      string // 前端 hash 结果（登录请求的 password 字段）
	Salt           string // 密码盐（user.salt）
	BackAlgorithm  string // 后端 hash 算法
	Password       string // 入库密码（user.password）
}

func main() {
	var (
		saltFlag   = flag.String("s", "", "密码盐（不填则生成 4 位随机盐；也可用第 2 个位置参数传入）")
		frontFlag  = flag.String("front", "", "前端 hash 算法：md5 / sha1 / sha256 / sm3（默认取配置）")
		backFlag   = flag.String("back", "", "后端 hash 算法：md5 / sha1 / sha256 / sm3（默认取配置）")
		userFlag   = flag.String("u", "", "用户名（填写后额外输出可直接执行的 UPDATE SQL）")
		showConfig = flag.Bool("c", false, "同时打印配置文件中的前后端算法与来源")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法：\n"+
			"  hash '明文'                    自动生成 4 位随机盐\n"+
			"  hash '明文' '盐'               指定盐（复现库中记录）\n"+
			"  hash '明文' '盐' sha256 sm3    临时指定前端 / 后端算法\n"+
			"  hash -u admin '明文' '盐'      额外输出 UPDATE SQL\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	plain := flag.Arg(0)
	if plain == "" {
		flag.Usage()
		os.Exit(2)
	}

	// 盐：-s 参数 > 第 2 个位置参数 > 随机生成
	salt := *saltFlag
	if salt == "" {
		salt = flag.Arg(1)
	}
	frontAlgo := *frontFlag
	if frontAlgo == "" {
		frontAlgo = flag.Arg(2)
	}
	backAlgo := *backFlag
	if backAlgo == "" {
		backAlgo = flag.Arg(3)
	}
	if salt == "" {
		salt = lib.RandomSalt(lib.SaltLength)
	}

	res, err := compute(plain, salt, frontAlgo, backAlgo)
	if err != nil {
		fail(err)
	}

	if *showConfig {
		fmt.Printf("配置前端算法：%s（%s）\n", lib.FrontendHashAlgorithm(), lib.KeyFrontendHash)
		fmt.Printf("配置后端算法：%s（%s）\n", lib.BackendHashAlgorithm(), lib.KeyBackendHash)
		fmt.Println()
	}
	if len(salt) != lib.SaltLength {
		fmt.Fprintf(os.Stderr, "提示：盐长度 %d 与默认的 %d 位不同（不影响计算，仅提醒与库中实际值保持一致）\n",
			len(salt), lib.SaltLength)
	}

	fmt.Printf("明文密码　：%s\n", res.Plain)
	fmt.Printf("前端算法　：%s\n", res.FrontAlgorithm)
	fmt.Printf("前端 hash ：%s\n", res.FrontHash)
	fmt.Printf("密码盐　　：%s\n", res.Salt)
	fmt.Printf("后端算法　：%s\n", res.BackAlgorithm)
	fmt.Printf("入库密码　：%s\n", res.Password)
	fmt.Println()
	fmt.Println("# 登录请求（POST /login）的 password 字段用「前端 hash」")
	fmt.Printf("#   {\"username\":\"%s\",\"password\":\"%s\"}\n", valueOrDefault(*userFlag, "<账号>"), res.FrontHash)
	if *userFlag != "" {
		fmt.Println()
		fmt.Println("# 直接修正库中密码（password 与 salt 必须成对更新）")
		fmt.Printf("UPDATE `user` SET `password`='%s', `salt`='%s' WHERE `username`='%s';\n",
			res.Password, res.Salt, *userFlag)
	}
}

// compute 按指定算法计算两段式 hash
// @param: plain string 明文密码
// @param: salt string 密码盐
// @param: frontAlgo string 前端算法（空则取配置）
// @param: backAlgo string 后端算法（空则取配置）
// @return: *result, error
func compute(plain, salt, frontAlgo, backAlgo string) (*result, error) {
	if frontAlgo == "" {
		frontAlgo = lib.FrontendHashAlgorithm()
	}
	if backAlgo == "" {
		backAlgo = lib.BackendHashAlgorithm()
	}
	front, err := lib.HashValue(frontAlgo, plain)
	if err != nil {
		return nil, err
	}
	// 与 lib.HashPassword 相同的拼接方式：前端 hash 与盐直接相连后再做后端 hash
	password, err := lib.HashValue(backAlgo, front+salt)
	if err != nil {
		return nil, err
	}
	return &result{
		Plain:          plain,
		FrontAlgorithm: strings.ToLower(strings.TrimSpace(frontAlgo)),
		FrontHash:      front,
		Salt:           salt,
		BackAlgorithm:  strings.ToLower(strings.TrimSpace(backAlgo)),
		Password:       password,
	}, nil
}

// valueOrDefault 空字符串时返回默认值
func valueOrDefault(value, def string) string {
	if value == "" {
		return def
	}
	return value
}

// fail 输出错误并退出
func fail(err error) {
	fmt.Fprintf(os.Stderr, "密码 hash 计算失败: %s\n", err)
	os.Exit(1)
}
