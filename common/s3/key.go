package s3

import (
	"crypto/rand"
	"encoding/hex"
	"mime"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

// 文件键扩展名白名单（不符合的扩展名一律丢弃，避免生成奇怪的对象键）
var keyExtensionPattern = regexp.MustCompile(`^\.[a-z0-9]{1,10}$`)

// NormalizeKey 归一化文件键：统一使用 / 分隔、去掉首尾 /、拒绝绝对路径与 ..
// @param: key string 原始文件键
// @return: string 归一化后的文件键, error 非法键返回 ErrInvalidKey
func NormalizeKey(key string) (string, error) {
	key = strings.ReplaceAll(strings.TrimSpace(key), "\\", "/")
	key = strings.TrimLeft(key, "/")
	if key == "" {
		return "", ErrInvalidKey
	}
	clean := path.Clean(key)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", ErrInvalidKey
	}
	return clean, nil
}

// GenerateKey 生成文件键：yyyy/mm/dd/<32 位随机十六进制><扩展名>
// @param: filename string 原始文件名（仅用于取扩展名，可传空）
// @return: string 文件键（如 2024/06/01/9f8d...c1.png）
func GenerateKey(filename string) string {
	ext := strings.ToLower(path.Ext(strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/")))
	if !keyExtensionPattern.MatchString(ext) {
		ext = ""
	}
	buf := make([]byte, objectKeyRandomByteCount)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 不可用时退化为时间戳，保证仍能生成唯一键
		return time.Now().Format("2006/01/02") + "/" + time.Now().Format("150405.000000000") + ext
	}
	return time.Now().Format("2006/01/02") + "/" + hex.EncodeToString(buf) + ext
}

// contentTypeByExtension 按扩展名推断内容类型
func contentTypeByExtension(key string) string {
	if ext := path.Ext(strings.ToLower(key)); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return DefaultContentType
}

// joinPublicURL 拼接对外访问地址（base 可为域名、CDN 前缀或网关路径；空则返回 /key）
func joinPublicURL(base, key string) string {
	clean, err := NormalizeKey(key)
	if err != nil {
		return ""
	}
	segments := strings.Split(clean, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	escaped := strings.Join(segments, "/")
	base = strings.TrimSpace(base)
	if base == "" {
		return "/" + escaped
	}
	return strings.TrimSuffix(base, "/") + "/" + escaped
}
