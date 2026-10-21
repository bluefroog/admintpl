package lib

import (
	"errors"
	"strings"
	"testing"
)

// TestNormalizeHashAlgorithm 校验 hash 算法归一化与非法算法报错
func TestNormalizeHashAlgorithm(t *testing.T) {
	cases := []struct {
		name      string
		algorithm string
		want      string
		wantErr   bool
	}{
		{name: "md5", algorithm: "md5", want: HashMD5},
		{name: "大写与空白", algorithm: "  SHA1 ", want: HashSHA1},
		{name: "sha256", algorithm: "sha256", want: HashSHA256},
		{name: "sm3", algorithm: "SM3", want: HashSM3},
		{name: "空值按默认", algorithm: "", want: HashSHA256},
		{name: "不支持", algorithm: "sha512", wantErr: true},
		{name: "非法值", algorithm: "crc32", wantErr: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got, err := NormalizeHashAlgorithm(item.algorithm)
			if item.wantErr {
				if err == nil {
					t.Fatalf("期望返回错误，实际 %q", got)
				}
				if !errors.Is(err, ErrUnsupportedHashAlgorithm) {
					t.Errorf("错误应可被 errors.Is(ErrUnsupportedHashAlgorithm) 识别，实际 %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("不应返回错误，实际 %v", err)
			}
			if got != item.want {
				t.Errorf("NormalizeHashAlgorithm(%q) = %q，期望 %q", item.algorithm, got, item.want)
			}
		})
	}
}

// TestHashValue 校验各算法对 "abc" 的标准摘要（十六进制小写）
func TestHashValue(t *testing.T) {
	cases := []struct {
		algorithm string
		want      string
	}{
		{algorithm: HashMD5, want: "900150983cd24fb0d6963f7d28e17f72"},
		{algorithm: HashSHA1, want: "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{algorithm: HashSHA256, want: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{algorithm: HashSM3, want: "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"},
	}
	for _, item := range cases {
		t.Run(item.algorithm, func(t *testing.T) {
			got, err := HashValue(item.algorithm, "abc")
			if err != nil {
				t.Fatalf("hash 失败：%v", err)
			}
			if got != item.want {
				t.Errorf("HashValue(%q, abc) = %s，期望 %s", item.algorithm, got, item.want)
			}
		})
	}

	if _, err := HashValue("sha512", "abc"); err == nil {
		t.Error("不支持的算法应返回错误")
	}
}

// TestHashPasswordAndCheck 校验两段式 hash：随机盐、可校验、错误值/空值拒绝
func TestHashPasswordAndCheck(t *testing.T) {
	// 前端 hash（模拟前端按 hashAlgorithm 计算的结果）
	front, err := HashValue(FrontendHashAlgorithm(), "admin123")
	if err != nil {
		t.Fatalf("前端 hash 失败：%v", err)
	}

	hashed, salt, err := HashPassword(front)
	if err != nil {
		t.Fatalf("HashPassword 失败：%v", err)
	}
	if len(salt) != SaltLength {
		t.Errorf("盐长度应为 %d，实际 %d（%q）", SaltLength, len(salt), salt)
	}
	if hashed == "" || hashed == front {
		t.Errorf("入库密码应为后端加盐 hash，实际 %q", hashed)
	}
	if !CheckPassword(hashed, salt, front) {
		t.Error("正确密码应校验通过")
	}
	if CheckPassword(hashed, salt, front+"x") {
		t.Error("错误密码不应校验通过")
	}
	if CheckPassword(hashed, salt+"x", front) {
		t.Error("盐不一致时不应校验通过")
	}
	if CheckPassword("", salt, front) || CheckPassword(hashed, salt, "") {
		t.Error("空密码/空 hash 不应校验通过")
	}

	// 每次生成的盐不同，同一个前端 hash 得到不同的入库密码
	hashed2, salt2, err := HashPassword(front)
	if err != nil {
		t.Fatalf("HashPassword 失败：%v", err)
	}
	if salt2 == salt && hashed2 == hashed {
		t.Error("多次加盐 hash 结果不应完全相同（随机盐失效）")
	}
	if !CheckPassword(hashed2, salt2, front) {
		t.Error("新盐生成的密码应校验通过")
	}
}

// TestHashPlainPassword 校验服务端已知明文时的两段式 hash（种子数据 / 初始密码场景）
func TestHashPlainPassword(t *testing.T) {
	hashed, salt, err := HashPlainPassword("admin123")
	if err != nil {
		t.Fatalf("HashPlainPassword 失败：%v", err)
	}
	front, err := HashValue(FrontendHashAlgorithm(), "admin123")
	if err != nil {
		t.Fatalf("前端 hash 失败：%v", err)
	}
	if !CheckPassword(hashed, salt, front) {
		t.Error("前端 hash 后应能通过校验（两段式算法不一致）")
	}
	if CheckPassword(hashed, salt, "admin124") {
		t.Error("错误明文不应校验通过")
	}
}

// TestRandomSalt 校验随机盐长度与字符集
func TestRandomSalt(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := 0; i < 100; i++ {
		salt := RandomSalt(SaltLength)
		if len(salt) != SaltLength {
			t.Fatalf("盐长度应为 %d，实际 %d（%q）", SaltLength, len(salt), salt)
		}
		for _, ch := range salt {
			if !strings.ContainsRune(charset, ch) {
				t.Fatalf("盐包含非法字符 %q（%q）", ch, salt)
			}
		}
	}
	if RandomSalt(0) == "" {
		t.Error("长度为 0 时应回退默认长度，不应返回空串")
	}
	if RandomSalt(-1) == "" {
		t.Error("负长度时应回退默认长度，不应返回空串")
	}
}
