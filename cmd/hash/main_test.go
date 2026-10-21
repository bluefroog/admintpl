package main

import (
	"testing"

	"github.com/bluefroog/admintpl/common/lib"
)

// TestCompute 校验两段式 hash 计算：前端 hash → 与盐拼接 → 后端 hash
func TestCompute(t *testing.T) {
	cases := []struct {
		name      string
		plain     string
		salt      string
		frontAlgo string
		backAlgo  string
		front     string
		password  string
	}{
		{
			name:      "sha256 + sha256",
			plain:     "admin123",
			salt:      "aB3d",
			frontAlgo: "sha256",
			backAlgo:  "sha256",
			front:     "240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9",
			password:  "de7f8ee9941a448d8fc2c356946e850603cb22b25943b63b29036f6344048b22",
		},
		{
			name:      "md5 + sm3（算法可混用，大小写不敏感）",
			plain:     "admin123",
			salt:      "aB3d",
			frontAlgo: "MD5",
			backAlgo:  " SM3 ",
			front:     "0192023a7bbd73250516f069df18b500",
			password:  "614931d36547566573663419baae29890620dbde317fe2925f1f312e2be4da41",
		},
		{
			name:      "sha1 + sha1",
			plain:     "admin123",
			salt:      "aB3d",
			frontAlgo: "sha1",
			backAlgo:  "sha1",
			front:     "f865b53623b121fd34ee5426c792e5c33af8c227",
			password:  "71797ad5d23eecdb3fd9dfd61a65685483fa5869",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := compute(c.plain, c.salt, c.frontAlgo, c.backAlgo)
			if err != nil {
				t.Fatalf("计算失败：%v", err)
			}
			if res.FrontHash != c.front {
				t.Errorf("前端 hash 期望 %s，实际 %s", c.front, res.FrontHash)
			}
			if res.Password != c.password {
				t.Errorf("入库密码期望 %s，实际 %s", c.password, res.Password)
			}
			if res.Salt != c.salt {
				t.Errorf("盐期望 %s，实际 %s", c.salt, res.Salt)
			}
			// 与库内登录校验保持一致（仅当后端算法与配置一致时 CheckPassword 才适用）
			if res.BackAlgorithm == lib.BackendHashAlgorithm() && res.FrontAlgorithm == lib.FrontendHashAlgorithm() {
				if !lib.CheckPassword(res.Password, res.Salt, res.FrontHash) {
					t.Errorf("CheckPassword 校验失败，密码 %s 盐 %s", res.Password, res.Salt)
				}
			}
		})
	}
}

// TestComputeDefaultAlgorithm 算法留空时取配置（默认 sha256）
func TestComputeDefaultAlgorithm(t *testing.T) {
	res, err := compute("admin123", "aB3d", "", "")
	if err != nil {
		t.Fatalf("计算失败：%v", err)
	}
	if res.FrontAlgorithm != lib.FrontendHashAlgorithm() || res.BackAlgorithm != lib.BackendHashAlgorithm() {
		t.Errorf("算法应回退配置值，实际 %s / %s", res.FrontAlgorithm, res.BackAlgorithm)
	}
	if res.Password == "" || res.FrontHash == res.Password {
		t.Errorf("两段式 hash 结果异常：front=%s password=%s", res.FrontHash, res.Password)
	}
}

// TestComputeInvalidAlgorithm 非法算法必须报错（不静默回退）
func TestComputeInvalidAlgorithm(t *testing.T) {
	if _, err := compute("admin123", "aB3d", "xxx", ""); err == nil {
		t.Error("非法前端算法应返回错误")
	}
	if _, err := compute("admin123", "aB3d", "", "xxx"); err == nil {
		t.Error("非法后端算法应返回错误")
	}
}

// TestValueOrDefault 空值返回默认值
func TestValueOrDefault(t *testing.T) {
	if got := valueOrDefault("", "admin"); got != "admin" {
		t.Errorf("空值应返回默认值，实际 %s", got)
	}
	if got := valueOrDefault("test", "admin"); got != "test" {
		t.Errorf("非空值应原样返回，实际 %s", got)
	}
}
