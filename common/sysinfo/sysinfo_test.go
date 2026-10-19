package sysinfo

import (
	"testing"
	"time"
)

func TestRunTime(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want string
	}{
		{"小于一天", 3*time.Hour + 5*time.Minute, "3小时5分钟"},
		{"超过一天", 26*time.Hour + 30*time.Minute, "1天2小时30分钟"},
		{"小于一小时", 42 * time.Minute, "42分钟"},
		{"负数", -1 * time.Minute, "0分钟"},
	}
	for _, c := range cases {
		if got := RunTime(c.in); got != c.want {
			t.Errorf("%s: RunTime(%v) = %s; want %s", c.name, c.in, got, c.want)
		}
	}
}

func TestPercent(t *testing.T) {
	if got := percent(0, 0); got != 0 {
		t.Errorf("percent(0,0) = %v; want 0", got)
	}
	if got := percent(1, 3); got != 33.33 {
		t.Errorf("percent(1,3) = %v; want 33.33", got)
	}
	if got := percent(3, 3); got != 100 {
		t.Errorf("percent(3,3) = %v; want 100", got)
	}
}

func TestParseStatCPU(t *testing.T) {
	content := "cpu  100 20 30 400 50 0 10 0 0 0\ncpu0 50 10 15 200 25 0 5 0 0 0\nintr 12345\n"
	total, idle, ok := parseStatCPU(content)
	if !ok {
		t.Fatal("parseStatCPU 解析失败")
	}
	if total != 610 {
		t.Errorf("total = %d; want 610", total)
	}
	if idle != 450 {
		t.Errorf("idle = %d; want 450", idle)
	}
	if _, _, ok := parseStatCPU("no cpu line"); ok {
		t.Error("无 CPU 行时应返回 false")
	}
}

func TestParseKeyValuesAndBuildMem(t *testing.T) {
	content := "MemTotal:       16384000 kB\nMemFree:         1000000 kB\nMemAvailable:    8192000 kB\n"
	values := parseKeyValues(content)
	if values["MemTotal"] != 16384000 || values["MemAvailable"] != 8192000 {
		t.Fatalf("parseKeyValues = %v", values)
	}
	mem := buildMem(values["MemTotal"], values["MemAvailable"])
	if mem.Total != 16000 || mem.Free != 8000 || mem.Used != 8000 {
		t.Errorf("buildMem = %+v; want total=16000 used=8000 free=8000", mem)
	}
	if mem.Usage != 50 {
		t.Errorf("Usage = %v; want 50", mem.Usage)
	}
	if empty := buildMem(0, 0); empty.Total != 0 {
		t.Errorf("总内存为 0 时应返回空结构：%+v", empty)
	}
	if overflow := buildMem(1000, 2000); overflow.Free != 0 || overflow.Used != 0 {
		t.Errorf("可用内存大于总内存时应修正为 0：%+v", overflow)
	}
}

func TestParseLoadAvg(t *testing.T) {
	avg5, avg15 := parseLoadAvg("0.11 0.25 0.50 1/234 5678\n")
	if avg5 != 0.25 || avg15 != 0.5 {
		t.Errorf("parseLoadAvg = %v, %v; want 0.25, 0.5", avg5, avg15)
	}
	if avg5, avg15 := parseLoadAvg("bad"); avg5 != 0 || avg15 != 0 {
		t.Errorf("非法内容应返回 0：%v, %v", avg5, avg15)
	}
}

func TestParseOSRelease(t *testing.T) {
	content := "NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 22.04.3 LTS\"\nID=ubuntu\n"
	if got := parseOSRelease(content); got != "Ubuntu 22.04.3 LTS" {
		t.Errorf("parseOSRelease = %s", got)
	}
	if got := parseOSRelease("ID=ubuntu"); got != "" {
		t.Errorf("无 PRETTY_NAME 时应返回空串，得到 %s", got)
	}
}

func TestParseMounts(t *testing.T) {
	content := "overlay / overlay rw,relatime 0 0\n" +
		"proc /proc proc rw,nosuid 0 0\n" +
		"/dev/sda1 /boot ext4 rw,relatime 0 0\n" +
		"/dev/sdb1 /data\\040disk xfs rw 0 0\n" +
		"/dev/sda1 /boot/dup ext4 rw 0 0\n"
	mounts := parseMounts(content)
	if len(mounts) != 3 {
		t.Fatalf("parseMounts 数量 = %d; want 3 (%v)", len(mounts), mounts)
	}
	if mounts[0].mountpoint != "/" || mounts[0].fsType != "overlay" {
		t.Errorf("容器根目录挂载解析错误：%+v", mounts[0])
	}
	if mounts[1].device != "/dev/sda1" || mounts[1].mountpoint != "/boot" {
		t.Errorf("设备挂载解析错误：%+v", mounts[1])
	}
	if mounts[2].mountpoint != "/data disk" {
		t.Errorf("转义挂载点解析错误：%+v", mounts[2])
	}
}

func TestCollect(t *testing.T) {
	info := Collect()
	if info.CPU.Num <= 0 {
		t.Errorf("CPU.Num = %d; want > 0", info.CPU.Num)
	}
	if info.Go.Name != "GoLang" || info.Go.Version == "" {
		t.Errorf("Go 运行时信息异常：%+v", info.Go)
	}
	if info.Go.StartTime == "" || info.Go.RunTime == "" {
		t.Errorf("启动时间/运行时长不应为空：%+v", info.Go)
	}
	if info.Sys.OSArch == "" {
		t.Errorf("系统架构不应为空：%+v", info.Sys)
	}
	if info.Mem.Total > 0 && info.Mem.Used+info.Mem.Free != info.Mem.Total {
		t.Errorf("内存数据不自洽：%+v", info.Mem)
	}
	for _, disk := range info.Disks {
		if disk.Total == 0 || disk.Used+disk.Free != disk.Total {
			t.Errorf("磁盘数据不自洽：%+v", disk)
		}
	}
}
