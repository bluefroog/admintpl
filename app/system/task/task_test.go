package task

import (
	"errors"
	"strings"
	"testing"
)

// TestParseTarget 调用目标字符串解析
func TestParseTarget(t *testing.T) {
	cases := []struct {
		target string
		bean   string
		method string
		args   []interface{}
	}{
		{"ryTask.ryNoParams()", "ryTask", "ryNoParams", nil},
		{"ryTask.ryParams('ry')", "ryTask", "ryParams", []interface{}{"ry"}},
		{"ryTask.ryParams(\"ry\")", "ryTask", "ryParams", []interface{}{"ry"}},
		{" ryTask.ryMultipleParams('ry', true, 2000, 1.0) ", "ryTask", "ryMultipleParams",
			[]interface{}{"ry", true, int64(2000), 1.0}},
		{"testTask.test2('hello, world')", "testTask", "test2", []interface{}{"hello, world"}},
		{"testTask.test2('')", "testTask", "test2", []interface{}{""}},
	}
	for _, item := range cases {
		bean, method, args, err := ParseTarget(item.target)
		if err != nil {
			t.Errorf("%q 解析失败：%v", item.target, err)
			continue
		}
		if bean != item.bean || method != item.method {
			t.Errorf("%q 解析结果应为 %s.%s，实际 %s.%s", item.target, item.bean, item.method, bean, method)
		}
		if len(args) != len(item.args) {
			t.Errorf("%q 参数个数应为 %d，实际 %d（%v）", item.target, len(item.args), len(args), args)
			continue
		}
		for i := range args {
			if args[i] != item.args[i] {
				t.Errorf("%q 第 %d 个参数应为 %v，实际 %v", item.target, i+1, item.args[i], args[i])
			}
		}
	}
}

// TestParseTargetInvalid 非法调用目标
func TestParseTargetInvalid(t *testing.T) {
	targets := []string{
		"",
		"ryTask",
		"ryTask.ryNoParams",
		"ryTask.ryNoParams(",
		".ryNoParams()",
		"ryTask.()",
		"ryTask.ryParams('ry'",
		"ryTask.ryParams(a)",
		"ryTask.ryParams('ry',)",
		"ryTask.ryParams('ry') extra",
	}
	for _, target := range targets {
		if _, _, _, err := ParseTarget(target); err == nil {
			t.Errorf("%q 应当解析失败", target)
		}
	}
}

// TestLookupAndInvoke 注册、查找与执行
func TestLookupAndInvoke(t *testing.T) {
	if _, ok := Lookup("ryTask", "ryNoParams"); !ok {
		t.Fatal("内置示例任务 ryTask.ryNoParams 应当已注册")
	}
	if _, ok := Lookup("notExists", "nothing"); ok {
		t.Fatal("未注册的目标不应被找到")
	}

	var received []interface{}
	Register("utTask", "capture", "单元测试用任务", func(args []interface{}) error {
		received = args
		return nil
	})
	if err := Invoke("utTask.capture('a', 2, false)"); err != nil {
		t.Fatalf("执行失败：%v", err)
	}
	if len(received) != 3 || received[0] != "a" || received[1] != int64(2) || received[2] != false {
		t.Errorf("任务收到的参数不正确：%v", received)
	}

	// 目标带参数时忽略 job_args
	received = nil
	if err := Invoke("utTask.capture('x')", "job-arg"); err != nil {
		t.Fatalf("执行失败：%v", err)
	}
	if len(received) != 1 || received[0] != "x" {
		t.Errorf("目标自带参数时不应使用 job_args，实际：%v", received)
	}

	// 目标不带参数时使用 job_args
	received = nil
	if err := Invoke("utTask.capture()", "job-arg", int64(9)); err != nil {
		t.Fatalf("执行失败：%v", err)
	}
	if len(received) != 2 || received[0] != "job-arg" || received[1] != int64(9) {
		t.Errorf("目标不带参数时应使用 job_args，实际：%v", received)
	}

	// 未注册目标报错，并给出可用目标提示
	err := Invoke("notExists.nothing()")
	if err == nil {
		t.Fatal("未注册目标应当报错")
	}
	if !strings.Contains(err.Error(), "调用目标不存在") || !strings.Contains(err.Error(), "utTask.capture") {
		t.Errorf("错误信息应包含可用目标提示，实际：%v", err)
	}
}

// TestRunError 任务返回错误
func TestRunError(t *testing.T) {
	Register("utTask", "fail", "执行失败的任务", func(args []interface{}) error {
		return errors.New("boom")
	})
	if err := Invoke("utTask.fail()"); err == nil || err.Error() != "boom" {
		t.Errorf("任务错误应向上返回，实际：%v", err)
	}
}

// TestSplitJobArgs job_args 解析（兼容 yjgo 的 | 分隔写法）
func TestSplitJobArgs(t *testing.T) {
	if got := SplitJobArgs(""); got != nil {
		t.Errorf("空 job_args 应返回 nil，实际 %v", got)
	}
	got := SplitJobArgs("a|2|true")
	if len(got) != 3 || got[0] != "a" || got[1] != "2" || got[2] != "true" {
		t.Errorf("job_args 解析结果不正确：%v", got)
	}
}

// TestKeys 已注册目标列表
func TestKeys(t *testing.T) {
	keys := Keys()
	if len(keys) == 0 {
		t.Fatal("应当有已注册的调用目标")
	}
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			t.Errorf("Keys 应当按字典序返回：%v", keys)
		}
	}
}
