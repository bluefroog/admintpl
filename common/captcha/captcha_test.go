package captcha

import (
	"strings"
	"testing"
	"time"
)

// TestGenerate 校验验证码生成（图片为 base64 PNG、标识唯一）
func TestGenerate(t *testing.T) {
	id, image, code, err := Generate(4, time.Minute)
	if err != nil {
		t.Fatalf("生成验证码失败：%v", err)
	}
	if id == "" {
		t.Error("验证码标识为空")
	}
	if len(code) != 4 {
		t.Errorf("验证码长度应为 4，实际 %d", len(code))
	}
	if image == "" {
		t.Error("验证码图片为空")
	}
	// PNG 文件头 base64 前缀为 iVBORw0KGgo
	if !strings.HasPrefix(image, "iVBORw0KGgo") {
		t.Errorf("验证码图片不是 PNG，前 16 位为 %s", image[:16])
	}
	// 位数小于 1 时按默认 4 位处理
	if _, _, code, err = Generate(0, time.Minute); err != nil || len(code) != 4 {
		t.Errorf("长度缺省时应为 4 位，实际 %d（err=%v）", len(code), err)
	}
}

// TestVerify 校验验证码一次性与错误码校验
func TestVerify(t *testing.T) {
	id, _, code, err := Generate(4, time.Minute)
	if err != nil {
		t.Fatalf("生成验证码失败：%v", err)
	}
	if !Verify(id, code) {
		t.Error("正确验证码应校验通过")
	}
	if Verify(id, code) {
		t.Error("验证码应一次性有效，二次校验必须失败")
	}

	id, _, code, _ = Generate(4, time.Minute)
	if Verify(id, "!!!") {
		t.Error("错误验证码不应通过")
	}
	if Verify(id, code) {
		t.Error("校验失败后验证码也应失效")
	}

	if Verify("", code) || Verify(id, "") {
		t.Error("空参数不应通过校验")
	}
}

// TestVerifyExpire 校验验证码过期
func TestVerifyExpire(t *testing.T) {
	id, _, code, err := Generate(4, time.Millisecond)
	if err != nil {
		t.Fatalf("生成验证码失败：%v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if Verify(id, code) {
		t.Error("验证码过期后不应通过校验")
	}
}

// TestRemove 校验主动失效
func TestRemove(t *testing.T) {
	id, _, code, _ := Generate(4, time.Minute)
	Remove(id)
	if Verify(id, code) {
		t.Error("验证码被移除后不应通过校验")
	}
	Remove("") // 空标识不应 panic
}
