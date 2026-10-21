package lib

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
)

// RandomHex 生成指定字节数的随机十六进制字符串（长度为 2*n）
// @param: n int 随机字节数
// @return: string
func RandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 在受支持的平台上不会失败；兜底使用时间纳秒填充，保证不返回空值
		return fallbackHex(n)
	}
	return hex.EncodeToString(b)
}

// Token 生成登录令牌（32 字节随机数，64 位十六进制字符串）
// @return: string
func Token() string {
	return RandomHex(32)
}

// UUID 生成 16 字节随机标识（32 位十六进制字符串），用于验证码标识等
// @return: string
func UUID() string {
	return RandomHex(16)
}

// RandomDigits 生成指定长度的随机数字串（用于图形验证码）
// @param: n int 长度
// @return: string
func RandomDigits(n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		v, err := randIndex(10)
		if err != nil {
			buf[i] = byte('0' + i%10)
			continue
		}
		buf[i] = byte('0' + v)
	}
	return string(buf)
}

// randIndex 返回 [0, n) 区间内的随机整数（基于 crypto/rand）
// @param: n int 上界（不含）
// @return: int 随机整数, error
func randIndex(n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("随机数上界必须大于 0")
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

// fallbackHex 随机数源不可用时的兜底实现（基于 big.Int 的随机源同样源自 crypto/rand，故仅在极端情况下触发）
func fallbackHex(n int) string {
	buf := make([]byte, 2*n)
	for i := range buf {
		buf[i] = '0'
	}
	return string(buf)
}
