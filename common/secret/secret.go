// Package secret 提供配置敏感值（数据库密码、令牌密钥等）的对称加解密能力。
//
// 设计要点：
//   - 算法：AES-256-GCM（带认证的加密，密文被篡改会解密失败，不会静默返回错误明文）；
//     加解密由 github.com/freewu/crypto-go 的 AES 封装完成（`aes.GetInstance("GCM")`），
//     该封装会自动在密文头部拼接 12 字节随机 nonce，无需自行管理 IV；
//   - 密钥：任意长度字符串，内部经 SHA-256 派生为 32 字节密钥（crypto-go 要求密钥长度为 16/24/32 字节，
//     派生后固定 32 字节，既方便直接使用口令，也避免「密钥长度不对」的运行时错误）；
//   - 格式：ENC(base64url(nonce || ciphertext || tag))，base64 URL 安全字母表不含括号，
//     因此密文可以整体作为配置值，也可以嵌在 DSN 等长字符串中（如 root:ENC(xxx)@tcp(...)）；
//   - 明文兼容：不含 ENC(...) 片段的值原样返回，便于「部分加密」渐进迁移。
package secret

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	cryptoaes "github.com/freewu/crypto-go/aes"
)

const (
	// Prefix 密文前缀
	Prefix = "ENC("
	// Suffix 密文后缀
	Suffix = ")"
	// mode 加密模式：GCM（AEAD，带完整性校验）
	mode = "GCM"
	// nonceSize GCM 随机数长度（crypto-go 的 GCM 封装固定把 12 字节 nonce 拼在密文头部）
	nonceSize = 12
)

// ErrNoKey 未提供加密密钥
var ErrNoKey = errors.New("未配置加密密钥（secret.key / CONFIG_SECRET_KEY）")

// cipher crypto-go 的 AES 封装实例（GCM 模式，无状态，可复用）
var cipher = cryptoaes.GetInstance(mode)

// deriveKey 将任意长度的口令派生为 32 字节（AES-256）密钥
func deriveKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// IsEncrypted 判断值（或其片段）是否含密文标记
func IsEncrypted(value string) bool {
	return strings.Contains(value, Prefix)
}

// Encrypt 把明文加密为 ENC(...) 密文
func Encrypt(plain, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", ErrNoKey
	}

	// crypto-go 的 GCM 封装内部生成随机 nonce 并拼接在密文头部：nonce || ciphertext || tag
	sealed, err := cipher.Encrypt([]byte(plain), deriveKey(key), nil)
	if err != nil {
		return "", err
	}

	return Prefix + base64.RawURLEncoding.EncodeToString(sealed) + Suffix, nil
}

// Decrypt 解密单个 ENC(...) 密文；非密文（无 ENC 前缀）原样返回
func Decrypt(value, key string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, Prefix) || !strings.HasSuffix(trimmed, Suffix) {
		return value, nil // 明文：保持原样，支持与密文混用
	}
	if strings.TrimSpace(key) == "" {
		return "", ErrNoKey
	}

	encoded := strings.TrimSuffix(strings.TrimPrefix(trimmed, Prefix), Suffix)
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("密文 base64 解码失败: %w", err)
	}
	// 至少要有 nonce + 认证标签，否则 crypto-go 的拆分逻辑会越界
	if len(sealed) < nonceSize+16 {
		return "", errors.New("密文长度不足（可能被截断）")
	}

	plain, err := cipher.Decrypt(sealed, deriveKey(key), nil)
	if err != nil {
		return "", errors.New("密文解密失败（密钥不匹配或内容被篡改）")
	}

	return string(plain), nil
}

// Resolve 解密字符串中的全部 ENC(...) 片段：
//   - 整值加密：ENC(xxx) -> 明文
//   - 局部加密：root:ENC(xxx)@tcp(127.0.0.1:3306)/db?... -> root:明文@tcp(...)
//
// 不含密文的值原样返回。
func Resolve(value, key string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}

	var out strings.Builder
	rest := value
	for {
		start := strings.Index(rest, Prefix)
		if start < 0 {
			out.WriteString(rest)
			return out.String(), nil
		}

		out.WriteString(rest[:start])
		rest = rest[start:]

		end := strings.Index(rest, Suffix)
		if end < 0 {
			return "", fmt.Errorf("加密值格式非法（缺少 %q）: %s", Suffix, mask(rest))
		}

		token := rest[:end+len(Suffix)]
		plain, err := Decrypt(token, key)
		if err != nil {
			return "", err
		}
		out.WriteString(plain)
		rest = rest[end+len(Suffix):]
	}
}

// mask 日志/报错用：只保留密文前若干字符，避免完整密文落到日志
func mask(value string) string {
	const keep = 12
	if len(value) <= keep {
		return value
	}
	return value[:keep] + "..."
}
