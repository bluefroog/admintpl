package lib

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"strings"

	"github.com/bluefroog/admintpl/common/config"
	log "github.com/sirupsen/logrus"
	"github.com/tjfoc/gmsm/sm3"
)

// 密码 hash 算法（前端与后端可分别配置，见 system.password.*）
const (
	// HashMD5 MD5（128 位，仅兼容老前端，安全性最弱）
	HashMD5 = "md5"
	// HashSHA1 SHA-1（160 位）
	HashSHA1 = "sha1"
	// HashSHA256 SHA-256（256 位，默认）
	HashSHA256 = "sha256"
	// HashSM3 国密 SM3（256 位）
	HashSM3 = "sm3"

	// DefaultFrontendHash 前端默认 hash 算法
	DefaultFrontendHash = HashSHA256
	// DefaultBackendHash 后端默认 hash 算法
	DefaultBackendHash = HashSHA256

	// SaltLength 密码盐长度（随机字符个数）
	SaltLength = 4
)

// 密码相关配置项（config/config.yaml -> system.password.*）
const (
	// KeyFrontendHash 前端 hash 算法（system.password.frontend-hash / PASSWORD_FRONTEND_HASH）
	KeyFrontendHash = "system.password.frontend-hash"
	// KeyBackendHash 后端 hash 算法（system.password.backend-hash / PASSWORD_BACKEND_HASH）
	KeyBackendHash = "system.password.backend-hash"
)

// ErrUnsupportedHashAlgorithm 不支持的 hash 算法
var ErrUnsupportedHashAlgorithm = errors.New("不支持的 hash 算法")

// NormalizeHashAlgorithm 归一化并校验 hash 算法（大小写与首尾空白不敏感）
// @param: algorithm string 算法名称（md5 / sha1 / sha256 / sm3）
// @return: string 归一化后的算法名称
// @return: error 算法不支持时返回 ErrUnsupportedHashAlgorithm
func NormalizeHashAlgorithm(algorithm string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(algorithm)) {
	case HashMD5:
		return HashMD5, nil
	case HashSHA1:
		return HashSHA1, nil
	case HashSHA256, "":
		return HashSHA256, nil
	case HashSM3:
		return HashSM3, nil
	default:
		return "", fmt.Errorf("%w: %q（仅支持 md5 / sha1 / sha256 / sm3）", ErrUnsupportedHashAlgorithm, algorithm)
	}
}

// HashValue 按指定算法对字符串做 hash，返回小写十六进制摘要
// @param: algorithm string hash 算法（md5 / sha1 / sha256 / sm3）
// @param: value string 待 hash 的字符串
// @return: string 十六进制摘要
// @return: error 算法不支持时返回错误
func HashValue(algorithm, value string) (string, error) {
	algo, err := NormalizeHashAlgorithm(algorithm)
	if err != nil {
		return "", err
	}
	var h hash.Hash
	switch algo {
	case HashMD5:
		h = md5.New()
	case HashSHA1:
		h = sha1.New()
	case HashSM3:
		h = sm3.New()
	default: // sha256
		h = sha256.New()
	}
	if _, err = h.Write([]byte(value)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FrontendHashAlgorithm 前端密码 hash 算法（读取 system.password.frontend-hash，非法值回退默认算法）
// @return: string 算法名称
func FrontendHashAlgorithm() string {
	return configuredHashAlgorithm(KeyFrontendHash, DefaultFrontendHash)
}

// BackendHashAlgorithm 后端密码 hash 算法（读取 system.password.backend-hash，非法值回退默认算法）
// @return: string 算法名称
func BackendHashAlgorithm() string {
	return configuredHashAlgorithm(KeyBackendHash, DefaultBackendHash)
}

// configuredHashAlgorithm 读取并校验配置中的 hash 算法
// @param: key string 配置键
// @param: def string 未配置或配置非法时的默认算法
// @return: string 归一化后的算法名称
func configuredHashAlgorithm(key, def string) string {
	value := strings.TrimSpace(config.String(key))
	if value == "" {
		return def
	}
	algo, err := NormalizeHashAlgorithm(value)
	if err != nil {
		log.Warnf("配置项 %s 取值非法（%v），已回退默认算法 %s", key, err, def)
		return def
	}
	return algo
}

// HashPassword 后端加盐 hash：把前端 hash 值与随机盐一起 hash 后入库
// @author: [bluefrog](https://github.com/freewu)
// @function: HashPassword
// @description: 生成密码哈希与随机盐（算法取 system.password.backend-hash）
// @param: value string 前端 hash 后的密码值
// @return: hashed string 入库密码（后端 hash）
// @return: salt string 随机盐（4 位）
// @return: err error
func HashPassword(value string) (hashed, salt string, err error) {
	salt = RandomSalt(SaltLength)
	hashed, err = HashValue(BackendHashAlgorithm(), value+salt)
	if err != nil {
		return "", "", err
	}
	return hashed, salt, nil
}

// HashPlainPassword 把明文密码转换为可入库的「前端 hash + 后端加盐 hash」结果（仅服务端已知明文的场景使用，如种子数据/初始密码）
// @param: plain string 明文密码
// @return: hashed string 入库密码（后端 hash）
// @return: salt string 随机盐（4 位）
// @return: err error
func HashPlainPassword(plain string) (hashed, salt string, err error) {
	frontHash, err := HashValue(FrontendHashAlgorithm(), plain)
	if err != nil {
		return "", "", err
	}
	return HashPassword(frontHash)
}

// CheckPassword 校验前端传来的 hash 值是否与库中密码一致（使用后端算法 + 库中盐重新计算）
// @author: [bluefrog](https://github.com/freewu)
// @function: CheckPassword
// @description: 校验密码是否正确
// @param: hashed string 库中密码（后端 hash）
// @param: salt string 库中盐
// @param: value string 前端 hash 后的密码值
// @return: bool
func CheckPassword(hashed, salt, value string) bool {
	if hashed == "" || value == "" {
		return false
	}
	computed, err := HashValue(BackendHashAlgorithm(), value+salt)
	if err != nil {
		return false
	}
	// 常量时间比较，避免按字符短路带来的时序差异
	return subtle.ConstantTimeCompare([]byte(computed), []byte(strings.ToLower(hashed))) == 1
}

// RandomSalt 生成指定长度的随机盐（大小写字母 + 数字，基于 crypto/rand）
// @param: n int 长度
// @return: string 随机盐
func RandomSalt(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if n <= 0 {
		n = SaltLength
	}
	buf := make([]byte, n)
	for i := range buf {
		idx, err := randIndex(len(charset))
		if err != nil {
			// crypto/rand 在受支持平台不会失败，兜底保证不返回空值
			buf[i] = charset[i%len(charset)]
			continue
		}
		buf[i] = charset[idx]
	}
	return string(buf)
}
