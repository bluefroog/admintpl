package lib

import (
	"net"
	"strings"
)

// LocalIP 获取本机（本容器）内网 IP
// @return: string IP 地址，获取失败返回空字符串
func LocalIP() string {
	// 不需要真正发包，UDP Dial 即可拿到出口网卡地址
	if conn, err := net.Dial("udp", "8.8.8.8:80"); err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil {
			return addr.IP.String()
		}
	}
	// 兜底：遍历网卡取第一个非回环 IPv4
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, item := range interfaces {
		if item.Flags&net.FlagUp == 0 || item.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := item.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			if ipNet, ok := address.(*net.IPNet); ok {
				if ipv4 := ipNet.IP.To4(); ipv4 != nil {
					return ipv4.String()
				}
			}
		}
	}
	return ""
}

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
