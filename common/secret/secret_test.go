package secret

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testKey = "test-secret-key-123"

// algorithms 覆盖所有受支持的算法
var algorithms = []string{AlgorithmAES, AlgorithmSM4}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plain := "root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local"

	for _, algorithm := range algorithms {
		t.Run(algorithm, func(t *testing.T) {
			token, err := Encrypt(plain, testKey, algorithm)
			if err != nil {
				t.Fatalf("加密失败: %v", err)
			}
			if !strings.HasPrefix(token, Prefix+algorithm+":") || !strings.HasSuffix(token, Suffix) {
				t.Fatalf("密文应带算法标签 %s%s:...%s，实际 %s", Prefix, algorithm, Suffix, token)
			}
			if strings.Contains(token, plain) {
				t.Fatal("密文中不应包含明文")
			}
			// base64 URL 安全字母表：密文不能出现 + / =，避免与 DSN、URL 冲突
			body := strings.TrimSuffix(strings.TrimPrefix(token, Prefix+algorithm+":"), Suffix)
			if strings.ContainsAny(body, "+/=") {
				t.Fatalf("密文应使用 URL 安全 base64: %s", token)
			}

			got, err := Decrypt(token, testKey)
			if err != nil {
				t.Fatalf("解密失败: %v", err)
			}
			if got != plain {
				t.Fatalf("解密结果 = %q, want %q", got, plain)
			}
		})
	}
}

func TestEmptyPlainRoundTrip(t *testing.T) {
	for _, algorithm := range algorithms {
		t.Run(algorithm, func(t *testing.T) {
			token, err := Encrypt("", testKey, algorithm)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Decrypt(token, testKey)
			if err != nil {
				t.Fatalf("空明文应可解密: %v", err)
			}
			if got != "" {
				t.Fatalf("解密结果应为空，实际 %q", got)
			}
		})
	}
}

func TestEncryptRandomness(t *testing.T) {
	for _, algorithm := range algorithms {
		t.Run(algorithm, func(t *testing.T) {
			first, err := Encrypt("same-plain", testKey, algorithm)
			if err != nil {
				t.Fatal(err)
			}
			second, err := Encrypt("same-plain", testKey, algorithm)
			if err != nil {
				t.Fatal(err)
			}
			if first == second {
				t.Fatal("相同明文两次加密的结果应不同（随机 nonce / IV）")
			}
		})
	}
}

func TestDecryptPlaintextPassthrough(t *testing.T) {
	got, err := Decrypt("root:123456@tcp(127.0.0.1:3306)/admintpl", testKey)
	if err != nil {
		t.Fatalf("明文不应报错: %v", err)
	}
	if got != "root:123456@tcp(127.0.0.1:3306)/admintpl" {
		t.Fatalf("明文应原样返回，实际 %q", got)
	}
}

// TestDecryptLegacyToken 无算法标签的历史密文按 AES 处理（向后兼容）
func TestDecryptLegacyToken(t *testing.T) {
	plain := "root:p@ss@tcp(127.0.0.1:3306)/admintpl"
	sealed, err := aesCipher.Encrypt([]byte(plain), deriveKey(testKey, aesKeySize), nil)
	if err != nil {
		t.Fatal(err)
	}
	legacy := Prefix + base64.RawURLEncoding.EncodeToString(sealed) + Suffix

	got, err := Decrypt(legacy, testKey)
	if err != nil {
		t.Fatalf("旧格式（无算法标签）密文应可解密: %v", err)
	}
	if got != plain {
		t.Fatalf("解密结果 = %q, want %q", got, plain)
	}
}

func TestDecryptErrors(t *testing.T) {
	for _, algorithm := range algorithms {
		t.Run(algorithm, func(t *testing.T) {
			token, err := Encrypt("plain", testKey, algorithm)
			if err != nil {
				t.Fatal(err)
			}

			cases := []struct {
				name  string
				value string
				key   string
				want  string
			}{
				{"密钥不匹配", token, "other-key", "解密失败"},
				{"密钥为空", token, "", "未配置加密密钥"},
				{"base64 非法", Prefix + algorithm + ":not*base64" + Suffix, testKey, "base64 解码失败"},
				{"密文被截断", token[:len(token)/2] + Suffix, testKey, "密文"},
				{"算法标签非法", Prefix + "DES:abcd" + Suffix, testKey, "不支持的配置加密算法"},
			}
			for _, item := range cases {
				t.Run(item.name, func(t *testing.T) {
					if _, err := Decrypt(item.value, item.key); err == nil {
						t.Fatal("应当返回错误")
					} else if !strings.Contains(err.Error(), item.want) {
						t.Fatalf("错误信息应包含 %q，实际 %q", item.want, err.Error())
					}
				})
			}
		})
	}
}

func TestEncryptRejectsEmptyKey(t *testing.T) {
	for _, algorithm := range algorithms {
		if _, err := Encrypt("plain", "  ", algorithm); err == nil {
			t.Fatalf("%s：密钥为空应返回错误", algorithm)
		}
	}
}

func TestEncryptRejectsUnknownAlgorithm(t *testing.T) {
	if _, err := Encrypt("plain", testKey, "DES"); err == nil {
		t.Fatal("不支持的算法应返回错误")
	}
}

func TestNormalizeAlgorithm(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		isError bool
	}{
		{"", AlgorithmAES, false},
		{"aes", AlgorithmAES, false},
		{"AES", AlgorithmAES, false},
		{" AES-GCM ", AlgorithmAES, false},
		{"AES-256", AlgorithmAES, false},
		{"sm4", AlgorithmSM4, false},
		{"SM4", AlgorithmSM4, false},
		{"sm4-cbc", AlgorithmSM4, false},
		{"DES", "", true},
		{"RC4", "", true},
	}
	for _, item := range cases {
		got, err := NormalizeAlgorithm(item.in)
		if item.isError {
			if err == nil {
				t.Fatalf("%q 应返回错误", item.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q 不应报错: %v", item.in, err)
		}
		if got != item.want {
			t.Fatalf("NormalizeAlgorithm(%q) = %q, want %q", item.in, got, item.want)
		}
	}
}

// TestDecryptIgnoresConfiguredAlgorithm 算法由密文自带标签决定，与当前配置无关
func TestDecryptIgnoresConfiguredAlgorithm(t *testing.T) {
	aesToken, err := Encrypt("aes-value", testKey, AlgorithmAES)
	if err != nil {
		t.Fatal(err)
	}
	sm4Token, err := Encrypt("sm4-value", testKey, AlgorithmSM4)
	if err != nil {
		t.Fatal(err)
	}

	for _, token := range []string{aesToken, sm4Token} {
		if _, err := Decrypt(token, testKey); err != nil {
			t.Fatalf("不分算法都应能解密 %s: %v", token, err)
		}
	}
}

func TestResolve(t *testing.T) {
	aesToken, err := Encrypt("p@ssw0rd", testKey, AlgorithmAES)
	if err != nil {
		t.Fatal(err)
	}
	sm4Token, err := Encrypt("p@ssw0rd", testKey, AlgorithmSM4)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		in    string
		want  string
		isErr bool
	}{
		{"无密文原样返回", "root:123456@tcp(127.0.0.1:3306)/admintpl", "root:123456@tcp(127.0.0.1:3306)/admintpl", false},
		{"AES 整体加密", aesToken, "p@ssw0rd", false},
		{"SM4 整体加密", sm4Token, "p@ssw0rd", false},
		{"DSN 局部加密（AES）", "root:" + aesToken + "@tcp(127.0.0.1:3306)/admintpl", "root:p@ssw0rd@tcp(127.0.0.1:3306)/admintpl", false},
		{"DSN 局部加密（SM4）", "root:" + sm4Token + "@tcp(127.0.0.1:3306)/admintpl", "root:p@ssw0rd@tcp(127.0.0.1:3306)/admintpl", false},
		{"多个密文", aesToken + "-" + sm4Token, "p@ssw0rd-p@ssw0rd", false},
		{"缺少右括号", "root:ENC(abcdef", "", true},
		{"明文与密文混合", "prefix-" + sm4Token + "-suffix", "prefix-p@ssw0rd-suffix", false},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got, err := Resolve(item.in, testKey)
			if item.isErr {
				if err == nil {
					t.Fatal("应当返回错误")
				}
				return
			}
			if err != nil {
				t.Fatalf("不应报错: %v", err)
			}
			if got != item.want {
				t.Fatalf("Resolve = %q, want %q", got, item.want)
			}
		})
	}
}

func TestIsEncrypted(t *testing.T) {
	if IsEncrypted("root:123456@tcp(127.0.0.1:3306)/admintpl") {
		t.Fatal("明文不应被识别为密文")
	}
	token, err := Encrypt("plain", testKey, AlgorithmSM4)
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(token) {
		t.Fatal("密文应被识别")
	}
}

// TestDecryptDetectsTampering 任意字节被篡改（含 SM4 的 IV 段、AES 的 nonce/tag 段）都必须报错，
// 不允许静默返回脏数据。
func TestDecryptDetectsTampering(t *testing.T) {
	for _, algorithm := range algorithms {
		t.Run(algorithm, func(t *testing.T) {
			token, err := Encrypt("root:123456@tcp(127.0.0.1:3306)/admintpl", testKey, algorithm)
			if err != nil {
				t.Fatal(err)
			}

			head := Prefix + algorithm + ":"
			sealed, err := base64.RawURLEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(token, head), Suffix))
			if err != nil {
				t.Fatal(err)
			}

			positions := []int{0, 3, len(sealed) / 2, len(sealed) - 1}
			for _, idx := range positions {
				tampered := make([]byte, len(sealed))
				copy(tampered, sealed)
				tampered[idx] ^= 0x01

				value := head + base64.RawURLEncoding.EncodeToString(tampered) + Suffix
				plain, err := Decrypt(value, testKey)
				if err == nil {
					t.Fatalf("第 %d 字节被篡改后仍解密成功（返回 %q），必须报错", idx, plain)
				}
			}
		})
	}
}
