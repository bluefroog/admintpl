// Package task 定时任务「调用目标」注册表。
//
// 调用目标字符串格式：beanName.methodName(args)，例如：
//
//	demoTask.noParams()                            // 无参
//	demoTask.params('demo')                          // 单个字符串参数
//	demoTask.multipleParams('demo', true, 2000, 1.0) // 多类型参数
//
// 这里只在「显式注册」的方法集合内查找并调用
// （见 Register），未注册的调用目标直接报错，天然避免任意方法调用。
package task

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// RunFunc 任务执行函数，args 为调用目标解析出的参数
type RunFunc func(args []interface{}) error

// Entry 一个已注册的调用目标
type Entry struct {
	Bean   string  // 目标对象名
	Method string  // 方法名
	Remark string  // 说明
	Run    RunFunc // 执行函数
}

// Key 返回 bean.method
func (e *Entry) Key() string {
	return e.Bean + "." + e.Method
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]*Entry)
)

// Register 注册调用目标（同 key 重复注册会覆盖，便于测试替换）
func Register(bean, method, remark string, run RunFunc) {
	if bean == "" || method == "" || run == nil {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	entry := &Entry{Bean: bean, Method: method, Remark: remark, Run: run}
	registry[entry.Key()] = entry
}

// Lookup 按 bean.method 查找调用目标
func Lookup(bean, method string) (*Entry, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	entry, ok := registry[bean+"."+method]
	return entry, ok
}

// Keys 返回全部已注册的调用目标（字典序），用于错误提示
func Keys() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	keys := make([]string, 0, len(registry))
	for key := range registry {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ParseTarget 解析调用目标字符串
//
// 返回对象名、方法名与参数列表；参数支持字符串（单/双引号）、整数、浮点数、布尔值。
func ParseTarget(target string) (bean, method string, args []interface{}, err error) {
	value := strings.TrimSpace(target)
	if value == "" {
		return "", "", nil, fmt.Errorf("调用目标字符串不能为空")
	}
	open := strings.Index(value, "(")
	closeIndex := strings.LastIndex(value, ")")
	if open <= 0 || closeIndex != len(value)-1 || closeIndex < open {
		return "", "", nil, fmt.Errorf("调用目标字符串格式错误（应为 bean.method(args)）：%s", target)
	}
	call := strings.TrimSpace(value[:open])
	dot := strings.LastIndex(call, ".")
	if dot <= 0 || dot == len(call)-1 {
		return "", "", nil, fmt.Errorf("调用目标字符串格式错误（缺少 bean.method）：%s", target)
	}
	bean, method = strings.TrimSpace(call[:dot]), strings.TrimSpace(call[dot+1:])
	if bean == "" || method == "" {
		return "", "", nil, fmt.Errorf("调用目标字符串格式错误（bean/method 为空）：%s", target)
	}
	body := strings.TrimSpace(value[open+1 : closeIndex])
	if body == "" {
		return bean, method, nil, nil
	}
	for _, item := range splitArgs(body) {
		arg, err := parseArg(item)
		if err != nil {
			return "", "", nil, err
		}
		args = append(args, arg)
	}
	return bean, method, args, nil
}

// splitArgs 按逗号切分参数（忽略引号内的逗号）
func splitArgs(body string) []string {
	var (
		items   []string
		current strings.Builder
		quote   rune
	)
	for _, char := range body {
		switch {
		case quote == 0 && (char == '\'' || char == '"'):
			quote = char
			current.WriteRune(char)
		case quote != 0 && char == quote:
			quote = 0
			current.WriteRune(char)
		case char == ',' && quote == 0:
			items = append(items, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}
	items = append(items, current.String())
	return items
}

// parseArg 解析单个参数（字符串 / 整数 / 浮点数 / 布尔值）
func parseArg(item string) (interface{}, error) {
	value := strings.TrimSpace(item)
	if value == "" {
		return nil, fmt.Errorf("调用目标参数不能为空")
	}
	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			return strings.ReplaceAll(value[1:len(value)-1], "\\'", "'"), nil
		}
	}
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null", "nil":
		return nil, nil
	}
	if integer, err := strconv.ParseInt(value, 10, 64); err == nil {
		return integer, nil
	}
	if float, err := strconv.ParseFloat(value, 64); err == nil {
		return float, nil
	}
	return nil, fmt.Errorf("调用目标参数类型不支持（支持字符串/整数/浮点数/布尔值）：%s", item)
}

// LookupTarget 解析并查找调用目标，未注册时返回可读错误
func LookupTarget(target string) (*Entry, []interface{}, error) {
	bean, method, args, err := ParseTarget(target)
	if err != nil {
		return nil, nil, err
	}
	entry, ok := Lookup(bean, method)
	if !ok {
		return nil, nil, fmt.Errorf("调用目标不存在（未注册）：%s.%s，当前可用目标：%s", bean, method, strings.Join(Keys(), ", "))
	}
	return entry, args, nil
}

// Invoke 执行调用目标
//
// extraArgs 为任务表中 job_args（以 | 分隔）解析出的位置参数，仅在调用目标字符串本身不带参数时使用，
// 用于兼容「调用目标本身不带参数」的任务写法。
func Invoke(target string, extraArgs ...interface{}) error {
	entry, args, err := LookupTarget(target)
	if err != nil {
		return err
	}
	if len(args) == 0 && len(extraArgs) > 0 {
		args = extraArgs
	}
	return entry.Run(args)
}

// SplitJobArgs 把任务表 job_args 字段（以 | 分隔）解析成参数列表
func SplitJobArgs(jobArgs string) []interface{} {
	if strings.TrimSpace(jobArgs) == "" {
		return nil
	}
	parts := strings.Split(jobArgs, "|")
	args := make([]interface{}, 0, len(parts))
	for _, part := range parts {
		args = append(args, strings.TrimSpace(part))
	}
	return args
}

func init() {
	// 内置示例任务
	Register("demoTask", "noParams", "无参示例任务", func(args []interface{}) error {
		log.Infof("[定时任务] demoTask.noParams 执行，无参数")
		return nil
	})
	Register("demoTask", "params", "一个参数的示例任务", func(args []interface{}) error {
		log.Infof("[定时任务] demoTask.params 执行，参数：%v", args)
		return nil
	})
	Register("demoTask", "multipleParams", "多个参数的示例任务", func(args []interface{}) error {
		log.Infof("[定时任务] demoTask.multipleParams 执行，参数：%v", args)
		return nil
	})
	// job_args 传参示例任务
	Register("testTask", "test1", "无参测试任务", func(args []interface{}) error {
		log.Infof("[定时任务] testTask.test1 执行，无参数")
		return nil
	})
	Register("testTask", "test2", "传参测试任务", func(args []interface{}) error {
		log.Infof("[定时任务] testTask.test2 执行，参数：%v", args)
		return nil
	})
}
