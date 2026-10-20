package secret

import (
	"strings"
	"testing"
)

const testKey = "test-secret-key-123"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plain := "root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local"

	token, err := Encrypt(plain, testKey)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if !strings.HasPrefix(token, Prefix) || !strings.HasSuffix(token, Suffix) {
		t.Fatalf("密文格式应为 %s...%s，实际 %s", Prefix, Suffix, token)
	}
	if strings.Contains(token, plain) {
		t.Fatal("密文中不应包含明文")
	}
	// base64 URL 安全字母表：密文不能出现 + / =，避免与 DSN、URL 冲突
	if body := strings.TrimSuffix(strings.TrimPrefix(token, Prefix), Suffix); strings.ContainsAny(body, "+/=") {
		t.Fatalf("密文应使用 URL 安全 base64: %s", token)
	}

	got, err := Decrypt(token, testKey)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if got != plain {
		t.Fatalf("解密结果 = %q, want %q", got, plain)
	}
}

func TestEncryptRandomNonce(t *testing.T) {
	first, err := Encrypt("same-plain", testKey)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encrypt("same-plain", testKey)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("相同明文两次加密的结果应不同（随机 nonce）")
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

func TestDecryptErrors(t *testing.T) {
	token, err := Encrypt("plain", testKey)
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
		{"base64 非法", "ENC(not*base64)", testKey, "base64 解码失败"},
		{"密文被截断", token[:len(token)/2] + ")", testKey, "密文长度不足"},
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
}

func TestEncryptRejectsEmptyKey(t *testing.T) {
	if _, err := Encrypt("plain", "  "); err == nil {
		t.Fatal("密钥为空应返回错误")
	}
}

func TestResolve(t *testing.T) {
	encrypted, err := Encrypt("p@ssw0rd", testKey)
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
		{"整体加密", encrypted, "p@ssw0rd", false},
		{"DSN 局部加密", "root:" + encrypted + "@tcp(127.0.0.1:3306)/admintpl", "root:p@ssw0rd@tcp(127.0.0.1:3306)/admintpl", false},
		{"多个密文", encrypted + "-" + encrypted, "p@ssw0rd-p@ssw0rd", false},
		{"缺少右括号", "root:ENC(abcdef", "", true},
		{"明文与密文混合", "prefix-" + encrypted + "-suffix", "prefix-p@ssw0rd-suffix", false},
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
	token, err := Encrypt("plain", testKey)
	if err != nil {
		t.Fatal(err)
	}
	if !IsEncrypted(token) {
		t.Fatal("密文应被识别")
	}
}
