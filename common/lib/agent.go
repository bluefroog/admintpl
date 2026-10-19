package lib

import "strings"

// ParseUserAgent 解析 User-Agent，返回浏览器与操作系统名称
// 说明：为避免引入第三方依赖，仅做常见浏览器/系统的关键字匹配；
// 匹配顺序从特殊到通用（如 Edge 的 UA 同时包含 Chrome 与 Safari）
// @param: userAgent string 请求头 User-Agent
// @return: browser string 浏览器名称, os string 操作系统名称
func ParseUserAgent(userAgent string) (browser string, os string) {
	browser = parseBrowser(userAgent)
	os = parseOS(userAgent)
	return browser, os
}

// parseBrowser 解析浏览器
func parseBrowser(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case ua == "":
		return "未知"
	case strings.Contains(ua, "micromessenger"):
		return "微信"
	case strings.Contains(ua, "edg/"):
		return "Edge"
	case strings.Contains(ua, "edgios") || strings.Contains(ua, "edga/"):
		return "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		return "Opera"
	case strings.Contains(ua, "qqbrowser"):
		return "QQ浏览器"
	case strings.Contains(ua, "ucbrowser"):
		return "UC浏览器"
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "crios"):
		return "Chrome"
	case strings.Contains(ua, "safari"):
		return "Safari"
	case strings.Contains(ua, "msie") || strings.Contains(ua, "trident"):
		return "Internet Explorer"
	case strings.Contains(ua, "postman"):
		return "Postman"
	case strings.Contains(ua, "curl"):
		return "curl"
	case strings.Contains(ua, "go-http-client"):
		return "Go-http-client"
	default:
		return "未知"
	}
}

// parseOS 解析操作系统
func parseOS(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case ua == "":
		return "未知"
	case strings.Contains(ua, "windows nt 10.0"):
		return "Windows 10"
	case strings.Contains(ua, "windows nt 6.3"):
		return "Windows 8.1"
	case strings.Contains(ua, "windows nt 6.2"):
		return "Windows 8"
	case strings.Contains(ua, "windows nt 6.1"):
		return "Windows 7"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ios"):
		return "iOS"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		return "Mac OS"
	case strings.Contains(ua, "ubuntu"):
		return "Ubuntu"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "未知"
	}
}
