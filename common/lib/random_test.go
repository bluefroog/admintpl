package lib

import "testing"

// TestParseUserAgent 校验 User-Agent 解析（浏览器与操作系统）
func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		name        string
		userAgent   string
		wantBrowser string
		wantOS      string
	}{
		{
			name:        "Chrome on Windows",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantBrowser: "Chrome",
			wantOS:      "Windows 10",
		},
		{
			name:        "Edge 包含 Chrome 关键字时优先识别为 Edge",
			userAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			wantBrowser: "Edge",
			wantOS:      "Windows 10",
		},
		{
			name:        "Firefox on Ubuntu",
			userAgent:   "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0",
			wantBrowser: "Firefox",
			wantOS:      "Ubuntu",
		},
		{
			name:        "微信内置浏览器",
			userAgent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 MicroMessenger/8.0.30",
			wantBrowser: "微信",
			wantOS:      "iOS",
		},
		{
			name:        "curl",
			userAgent:   "curl/8.5.0",
			wantBrowser: "curl",
			wantOS:      "未知",
		},
		{
			name:        "空 User-Agent",
			userAgent:   "",
			wantBrowser: "未知",
			wantOS:      "未知",
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			browser, osName := ParseUserAgent(item.userAgent)
			if browser != item.wantBrowser {
				t.Errorf("浏览器解析错误：期望 %s，实际 %s", item.wantBrowser, browser)
			}
			if osName != item.wantOS {
				t.Errorf("操作系统解析错误：期望 %s，实际 %s", item.wantOS, osName)
			}
		})
	}
}

// TestIPLocation 校验 IP 归属地判定（仅本机/内网/未知）
func TestIPLocation(t *testing.T) {
	cases := map[string]string{
		"":                "未知",
		"127.0.0.1":       "本机",
		"::1":             "本机",
		"192.168.122.129": "内网IP",
		"10.0.0.8":        "内网IP",
		"172.16.1.1":      "内网IP",
		"8.8.8.8":         "未知",
		"非法IP":            "未知",
	}
	for ip, want := range cases {
		if got := IPLocation(ip); got != want {
			t.Errorf("IPLocation(%q) = %s，期望 %s", ip, got, want)
		}
	}
}

// TestRandomHelpers 校验随机串生成的长度与字符范围
func TestRandomHelpers(t *testing.T) {
	if got := len(Token()); got != 64 {
		t.Errorf("Token 长度应为 64，实际 %d", got)
	}
	if got := len(UUID()); got != 32 {
		t.Errorf("UUID 长度应为 32，实际 %d", got)
	}
	if got := len(RandomDigits(6)); got != 6 {
		t.Errorf("RandomDigits 长度应为 6，实际 %d", got)
	}
	for i, ch := range RandomDigits(64) {
		if ch < '0' || ch > '9' {
			t.Errorf("RandomDigits 第 %d 位出现非数字字符：%c", i, ch)
		}
	}
	if Token() == Token() {
		t.Error("连续两次生成的令牌不应相同")
	}
}
