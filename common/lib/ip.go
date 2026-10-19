package lib

import (
	"net"
	"strings"
)

// IPLocation 解析 IP 归属地
// 说明：与源项目（调用外部 IP 库/接口）不同，后台服务默认不发起外部请求，
// 因此这里只做本机/内网判定，其余返回“未知”，需要精细归属地时再接入离线 IP 库
// @param: ip string IP 地址
// @return: string 归属地描述
func IPLocation(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "未知"
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "未知"
	}
	if parsed.IsLoopback() {
		return "本机"
	}
	if parsed.IsPrivate() || parsed.IsLinkLocalUnicast() {
		return "内网IP"
	}
	return "未知"
}
