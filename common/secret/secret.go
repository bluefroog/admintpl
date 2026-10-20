// Package secret 提供配置敏感值（数据库密码、令牌密钥等）的对称加解密能力。
//
// 设计要点：
//   - 算法可配置：`AES`（默认，AES-256-GCM）或 `SM4`（国密 SM4-CBC），由 secret.algorithm / CONFIG_SECRET_ALGORITHM 指定；
//     加解密统一复用 github.com/freewu/crypto-go 的封装（`aes.GetInstance("GCM")` / `sm4.GetInstance("CBC")`）；
//   - 两种算法都具备完整性校验：AES-GCM 由认证标签保证；SM4-CBC 无认证能力，
//     因此在密文载荷内附一段截断的 SHA-256 校验和（16 字节），篡改 IV 或密文都会校验失败，
//     行为与 AES 保持一致（绝不静默返回脏数据）；
//   - 算法随密文走：密文形如 `ENC(AES:xxxx)` / `ENC(SM4:xxxx)`，解密时按密文自带的算法标签选择算法，
//     因此切换算法不影响历史密文；无标签的旧密文（`ENC(xxxx)`）按 AES 处理，保持向后兼容；
//   - 密钥：任意长度字符串，内部经 SHA-256 派生（AES 取 32 字节、SM4 取 16 字节），
//     既方便直接使用口令，也满足 crypto-go / gmsm 对密钥长度的硬性要求；
//   - 格式：`ENC(base64url(...))`，base64 URL 安全字母表不含括号与冒号，
//     因此密文可以整体作为配置值，也可以嵌在 DSN 等长字符串中（如 root:ENC(xxx)@tcp(...)）；
//   - 明文兼容：不含 ENC(...) 片段的值原样返回，便于「部分加密」渐进迁移。
package secret

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	cryptoaes "github.com/freewu/crypto-go/aes"
	cryptosm4 "github.com/freewu/crypto-go/sm4"
)

const (
	// Prefix 密文前缀
	Prefix = "ENC("
	// Suffix 密文后缀
	Suffix = ")"

	// AlgorithmAES AES-256-GCM（带认证，默认算法）
	AlgorithmAES = "AES"
	// AlgorithmSM4 国密 SM4-CBC（分组模式，无 AEAD）
	AlgorithmSM4 = "SM4"
	// DefaultAlgorithm 默认算法
	DefaultAlgorithm = AlgorithmAES

	// aesKeySize AES-256 密钥长度
	aesKeySize = 32
	// sm4KeySize SM4 密钥长度（gmsm 要求 16 字节）
	sm4KeySize = 16
	// nonceSize crypto-go 的 GCM 封装固定把 12 字节 nonce 拼在密文头部
	nonceSize = 12
	// sm4BlockSize SM4 分组长度，同时作为 CBC 的 IV 长度
	sm4BlockSize = 16
	// tagSize GCM 认证标签长度
	tagSize = 16
	// checksumSize SM4-CBC 载荷内附带的截断 SHA-256 校验和长度
	checksumSize = 16
)

// ErrNoKey 未提供加密密钥
var ErrNoKey = errors.New("未配置加密密钥（secret.key / CONFIG_SECRET_KEY）")

var (
	// aesCipher crypto-go 的 AES 封装（GCM 模式，无状态，可复用）
	aesCipher = cryptoaes.GetInstance("GCM")
	// sm4Cipher crypto-go 的 SM4 封装（CBC 模式）；其 IV 通过 gmsm 的包级 sm4.SetIV 设置，
	// 并发调用会互相覆盖，因此用它时必须串行
	sm4Cipher = cryptosm4.GetInstance("CBC")
	// sm4Mu 保护 SM4 加解密（crypto-go 的 SM4 实现非并发安全）
	sm4Mu sync.Mutex
)

// NormalizeAlgorithm 归一化算法名（大小写不敏感，允许常见别名），非法值返回错误
func NormalizeAlgorithm(algorithm string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "", AlgorithmAES, "AES-GCM", "AESGCM", "AES256", "AES-256":
		return AlgorithmAES, nil
	case AlgorithmSM4, "SM4-CBC", "SM4CBC":
		return AlgorithmSM4, nil
	}
	return "", fmt.Errorf("不支持的配置加密算法 %q（可选：%s / %s）", algorithm, AlgorithmAES, AlgorithmSM4)
}

// deriveKey 将任意长度的口令派生为指定长度的密钥
func deriveKey(key string, size int) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:size]
}

// IsEncrypted 判断值（或其片段）是否含密文标记
func IsEncrypted(value string) bool {
	return strings.Contains(value, Prefix)
}

// Encrypt 把明文加密为 ENC(算法:密文)；
// algorithm 为空时使用 DefaultAlgorithm（AES）。
func Encrypt(plain, key, algorithm string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", ErrNoKey
	}

	algo, err := NormalizeAlgorithm(algorithm)
	if err != nil {
		return "", err
	}

	var sealed []byte
	switch algo {
	case AlgorithmSM4:
		sealed, err = encryptSM4([]byte(plain), deriveKey(key, sm4KeySize))
	default:
		// crypto-go 的 GCM 封装内部生成随机 nonce 并拼接在密文头部：nonce || ciphertext || tag
		sealed, err = aesCipher.Encrypt([]byte(plain), deriveKey(key, aesKeySize), nil)
	}
	if err != nil {
		return "", err
	}

	return Prefix + algo + ":" + base64.RawURLEncoding.EncodeToString(sealed) + Suffix, nil
}

// Decrypt 解密单个 ENC(...) 密文；非密文（无 ENC 前缀）原样返回。
// 算法取自密文自带的标签，无标签按 AES 处理（向后兼容）。
func Decrypt(value, key string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, Prefix) || !strings.HasSuffix(trimmed, Suffix) {
		return value, nil // 明文：保持原样，支持与密文混用
	}
	if strings.TrimSpace(key) == "" {
		return "", ErrNoKey
	}

	body := strings.TrimSuffix(strings.TrimPrefix(trimmed, Prefix), Suffix)
	algo, encoded := DefaultAlgorithm, body
	// base64 URL 字母表不含冒号，出现冒号即代表算法标签
	if idx := strings.Index(body, ":"); idx >= 0 {
		normalized, err := NormalizeAlgorithm(body[:idx])
		if err != nil {
			return "", err
		}
		algo, encoded = normalized, body[idx+1:]
	}

	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("密文 base64 解码失败: %w", err)
	}

	var plain []byte
	switch algo {
	case AlgorithmSM4:
		plain, err = decryptSM4(sealed, deriveKey(key, sm4KeySize))
	default:
		// 至少要有 nonce + 认证标签，否则 crypto-go 的拆分逻辑会越界
		if len(sealed) < nonceSize+tagSize {
			return "", errors.New("密文长度不足（可能被截断）")
		}
		plain, err = aesCipher.Decrypt(sealed, deriveKey(key, aesKeySize), nil)
	}
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

// encryptSM4 SM4-CBC 加密，返回 iv || ciphertext（PKCS7 填充由 gmsm 完成）；
// 明文载荷为 校验和 || 明文，用于弥补 CBC 无完整性校验的短板。
func encryptSM4(plain, key []byte) ([]byte, error) {
	payload := append(checksum(plain), plain...)

	iv := make([]byte, sm4BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	sm4Mu.Lock()
	defer sm4Mu.Unlock()

	encrypted, err := sm4Cipher.Encrypt(payload, key, iv)
	if err != nil {
		return nil, err
	}
	return append(iv, encrypted...), nil
}

// decryptSM4 解密 iv || ciphertext，并校验载荷完整性
func decryptSM4(sealed, key []byte) ([]byte, error) {
	if len(sealed) <= sm4BlockSize || (len(sealed)-sm4BlockSize)%sm4BlockSize != 0 {
		return nil, errors.New("sm4 密文长度非法（可能被截断）")
	}
	// 去掉 IV 后至少要容纳「填充后的一个分组」，其中含校验和
	if len(sealed)-sm4BlockSize < sm4BlockSize {
		return nil, errors.New("sm4 密文长度非法（可能被截断）")
	}

	iv, ciphertext := sealed[:sm4BlockSize], sealed[sm4BlockSize:]

	sm4Mu.Lock()
	defer sm4Mu.Unlock()

	// gmsm 的 CBC 实现会吞掉去填充错误并返回 nil，这里用 nil 判定解密失败
	plain, err := sm4Cipher.Decrypt(ciphertext, key, iv)
	if err != nil {
		return nil, err
	}
	if plain == nil {
		return nil, errors.New("sm4 去填充失败（密钥不匹配或内容被篡改）")
	}
	if len(plain) < checksumSize {
		return nil, errors.New("sm4 载荷损坏（密钥不匹配或内容被篡改）")
	}

	payload, sum := plain[checksumSize:], plain[:checksumSize]
	if !hmac.Equal(sum, checksum(payload)) {
		return nil, errors.New("sm4 完整性校验失败（密钥不匹配或内容被篡改）")
	}
	return payload, nil
}

// checksum 返回明文的截断 SHA-256 校验和
func checksum(plain []byte) []byte {
	sum := sha256.Sum256(plain)
	return sum[:checksumSize]
}

// mask 日志/报错用：只保留密文前若干字符，避免完整密文落到日志
func mask(value string) string {
	const keep = 12
	if len(value) <= keep {
		return value
	}
	return value[:keep] + "..."
}
