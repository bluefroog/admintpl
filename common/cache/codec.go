package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"sync"
)

// envelope 缓存值序列化信封：具体类型名 + gob 载荷
//
// 缓存对业务暴露的是 interface{}，落地到文件 / memcache / redis 时必须序列化；
// 这里用 gob 并按「类型名」记录具体类型，读取时再按类型名还原，
// 因此业务结构体只需在进程内注册一次（见 RegisterType），无需 gob.Register 全局注册。
type envelope struct {
	Type string // 具体类型名（reflect.Type.String()，如 []domain.DictData）
	Data []byte // gob 编码后的具体值
}

var (
	typeMu sync.RWMutex
	// types 已注册的可持久化缓存值类型（类型名 -> 类型）
	types = map[string]reflect.Type{}
)

// RegisterType 注册可持久化的缓存值类型
//
// 基础类型（string / int / bool / []string 等）已由本包注册；
// 业务结构体（如 []domain.DictData）需在 init 中注册，否则进程重启后无法从
// 本地文件 / memcache / redis 还原（会退化为缓存未命中，不影响正确性）。
// @param: samples ...interface{} 该类型的任一实例（如 []domain.DictData{}）
func RegisterType(samples ...interface{}) {
	typeMu.Lock()
	defer typeMu.Unlock()
	for _, sample := range samples {
		if sample == nil {
			continue
		}
		t := reflect.TypeOf(sample)
		types[t.String()] = t
	}
}

// encodeValue 序列化缓存值（类型名 + gob 载荷）
// @param: value interface{} 缓存值
// @return: []byte 序列化结果, error 错误信息
func encodeValue(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("缓存值不能为空")
	}
	t := reflect.TypeOf(value)
	var data bytes.Buffer
	if err := gob.NewEncoder(&data).Encode(value); err != nil {
		return nil, fmt.Errorf("缓存值序列化失败（%s）: %w", t.String(), err)
	}
	// 同进程写入时自动登记类型，保证回读可用（跨进程依赖 RegisterType）
	typeMu.Lock()
	types[t.String()] = t
	typeMu.Unlock()

	var out bytes.Buffer
	if err := gob.NewEncoder(&out).Encode(envelope{Type: t.String(), Data: data.Bytes()}); err != nil {
		return nil, fmt.Errorf("缓存值封装失败: %w", err)
	}
	return out.Bytes(), nil
}

// decodeValue 反序列化缓存值
// @param: payload []byte 序列化载荷
// @return: interface{} 缓存值, error 错误信息
func decodeValue(payload []byte) (interface{}, error) {
	var env envelope
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&env); err != nil {
		return nil, fmt.Errorf("缓存值反序列化失败: %w", err)
	}
	typeMu.RLock()
	t, ok := types[env.Type]
	typeMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("缓存值类型未注册: %s（请在 init 中调用 cache.RegisterType）", env.Type)
	}
	ptr := reflect.New(t)
	if err := gob.NewDecoder(bytes.NewReader(env.Data)).Decode(ptr.Interface()); err != nil {
		return nil, fmt.Errorf("缓存值反序列化失败（%s）: %w", env.Type, err)
	}
	return ptr.Elem().Interface(), nil
}

func init() {
	// 基础类型：多数业务缓存值都是这些（验证码字符串、参数值、计数等）
	RegisterType("", []byte{}, 0, int8(0), int16(0), int32(0), int64(0),
		uint(0), uint8(0), uint16(0), uint32(0), uint64(0),
		float32(0), float64(0), false,
		[]string{}, []int{}, []int64{}, []float64{}, []bool{},
		map[string]string{}, map[string]interface{}{}, map[string]int64{},
		[]interface{}{})
}
