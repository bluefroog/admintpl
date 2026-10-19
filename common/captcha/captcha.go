package captcha

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"strings"
	"time"

	"github.com/bluefroog/admintpl/common/cache"
	"github.com/bluefroog/admintpl/common/lib"
)

// keyPrefix 验证码在缓存中的键前缀
const keyPrefix = "captcha:"

// store 验证码内存存储（进程内有效，多实例部署时不共享；已注册到缓存监控）
var store = cache.New(2 * time.Minute)

func init() {
	cache.Register("captcha", store)
}

// font5x7 0-9 的 5x7 点阵字模（仅数字，用于标准库绘制验证码）
var font5x7 = [10][7]string{
	{"01110", "10001", "10011", "10101", "11001", "10001", "01110"}, // 0
	{"00100", "01100", "00100", "00100", "00100", "00100", "01110"}, // 1
	{"01110", "10001", "00001", "00010", "00100", "01000", "11111"}, // 2
	{"11111", "00010", "00100", "00010", "00001", "10001", "01110"}, // 3
	{"00010", "00110", "01010", "10010", "11111", "00010", "00010"}, // 4
	{"11111", "10000", "11110", "00001", "00001", "10001", "01110"}, // 5
	{"00110", "01000", "10000", "11110", "10001", "10001", "01110"}, // 6
	{"11111", "00001", "00010", "00100", "01000", "01000", "01000"}, // 7
	{"01110", "10001", "10001", "01110", "10001", "10001", "01110"}, // 8
	{"01110", "10001", "10001", "01111", "00001", "00010", "01100"}, // 9
}

// Generate 生成数字图形验证码
// @function: Generate
// @description: 生成验证码图片（base64 编码，前端需拼接 data:image/png;base64, 前缀），并把验证码文本写入内存缓存
// @param: length int 验证码位数
// @param: expire time.Duration 有效期
// @return: id string 验证码标识, image string 图片 base64, code string 验证码文本, err error
func Generate(length int, expire time.Duration) (id string, image string, code string, err error) {
	if length <= 0 {
		length = 4
	}
	if expire <= 0 {
		expire = 2 * time.Minute
	}
	code = lib.RandomDigits(length)
	id = lib.UUID()
	img, err := drawImage(code)
	if err != nil {
		return "", "", "", err
	}
	store.Set(keyPrefix+id, code, expire)
	return id, img, code, nil
}

// Verify 校验验证码（一次性，无论成功与否都会失效）
// @param: id string 验证码标识
// @param: code string 用户输入的验证码
// @return: bool 是否校验通过
func Verify(id string, code string) bool {
	if id == "" || code == "" {
		return false
	}
	key := keyPrefix + id
	value, ok := store.Get(key)
	if !ok {
		return false
	}
	store.Delete(key)
	return strings.EqualFold(strings.TrimSpace(code), value.(string))
}

// Remove 移除指定验证码（主动失效）
// @param: id string 验证码标识
func Remove(id string) {
	if id != "" {
		store.Delete(keyPrefix + id)
	}
}

// drawImage 使用标准库绘制验证码图片并返回 base64 字符串
func drawImage(code string) (string, error) {
	const (
		scale    = 6 // 字模放大倍数
		charW    = 5 * scale
		charH    = 7 * scale
		paddingX = 12
		paddingY = 10
		gap      = 8
	)
	width := paddingX*2 + len(code)*charW + (len(code)-1)*gap
	height := charH + paddingY*2
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 随机浅色背景
	draw.Draw(img, img.Bounds(), &image.Uniform{C: lightColor()}, image.Point{}, draw.Src)

	// 逐位绘制数字，位置与颜色随机，模拟轻微抖动
	for i, ch := range code {
		if ch < '0' || ch > '9' {
			continue
		}
		fg := darkColor()
		x0 := paddingX + i*(charW+gap) + randomInt(4) - 2
		y0 := paddingY + randomInt(6) - 3
		for row, line := range font5x7[ch-'0'] {
			for col, bit := range line {
				if bit != '1' {
					continue
				}
				rect := image.Rect(
					x0+col*scale,
					y0+row*scale,
					x0+(col+1)*scale,
					y0+(row+1)*scale,
				)
				draw.Draw(img, rect, &image.Uniform{C: fg}, image.Point{}, draw.Src)
			}
		}
	}

	// 干扰线
	for i := 0; i < 4; i++ {
		c := lightColor()
		x1, y1, x2, y2 := randomInt(width), randomInt(height), randomInt(width), randomInt(height)
		drawLine(img, x1, y1, x2, y2, c)
	}
	// 干扰点
	for i := 0; i < 80; i++ {
		img.Set(randomInt(width), randomInt(height), lightColor())
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		return "", errors.New("验证码生成失败")
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// drawLine 绘制直线（Bresenham 算法）
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx, dy := x2-x1, y2-y1
	steps := dx
	if steps < 0 {
		steps = -steps
	}
	if dy > steps {
		steps = dy
	} else if -dy > steps {
		steps = -dy
	}
	if steps == 0 {
		img.Set(x1, y1, c)
		return
	}
	for i := 0; i <= steps; i++ {
		img.Set(x1+dx*i/steps, y1+dy*i/steps, c)
	}
}

// lightColor 随机浅色（背景/干扰元素）
func lightColor() color.RGBA {
	return color.RGBA{R: uint8(180 + randomInt(70)), G: uint8(180 + randomInt(70)), B: uint8(180 + randomInt(70)), A: 255}
}

// darkColor 随机深色（数字）
func darkColor() color.RGBA {
	return color.RGBA{R: uint8(randomInt(100)), G: uint8(randomInt(100)), B: uint8(randomInt(120)), A: 255}
}

// randomInt 返回 [0,n) 之间的随机整数
func randomInt(n int) int {
	if n <= 0 {
		return 0
	}
	v, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}
