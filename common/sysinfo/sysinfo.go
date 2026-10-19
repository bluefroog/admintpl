// Package sysinfo 采集服务器运行指标。
//
// 与源项目（使用 github.com/shirou/gopsutil）不同，这里只用 Go 标准库：
//   - Linux：读取 /proc/stat、/proc/meminfo、/proc/loadavg、/proc/mounts + syscall.Statfs
//   - 其它平台：无法采集的系统指标返回零值，仅返回 Go 运行时与主机基础信息
package sysinfo

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/bluefroog/admintpl/common/lib"
)

// CPU CPU 信息
type CPU struct {
	Num   int     // 核心数
	Used  float64 // 使用率（%）
	Avg5  float64 // 5 分钟平均负载
	Avg15 float64 // 15 分钟平均负载
}

// Mem 内存信息（单位 MB）
type Mem struct {
	Total uint64  // 总内存
	Used  uint64  // 已用内存
	Free  uint64  // 剩余内存
	Usage float64 // 使用率（%）
}

// Go Go 运行时信息（单位 MB）
type Go struct {
	Name      string  // 语言环境
	Version   string  // 版本
	StartTime string  // 启动时间
	RunTime   string  // 运行时长
	Home      string  // 安装路径
	UserDir   string  // 项目路径
	Total     uint64  // 分配的总内存
	Used      uint64  // 已使用的内存
	Free      uint64  // 剩余的内存
	Usage     float64 // 使用率（%）
}

// Sys 服务器信息
type Sys struct {
	ComputerName string // 服务器名称
	ComputerIP   string // 服务器 IP
	OSName       string // 操作系统
	OSArch       string // 系统架构
}

// Disk 磁盘信息（单位 MB）
type Disk struct {
	DirName     string  // 盘符路径
	SysTypeName string  // 文件系统
	TypeName    string  // 盘符类型
	Total       uint64  // 总大小
	Free        uint64  // 剩余大小
	Used        uint64  // 已用大小
	Usage       float64 // 使用率（%）
}

// Info 服务器监控汇总
type Info struct {
	CPU   CPU
	Mem   Mem
	Go    Go
	Sys   Sys
	Disks []Disk
}

// startTime 进程启动时间
var startTime = time.Now()

// cpuSampleInterval CPU 使用率采样间隔
const cpuSampleInterval = 200 * time.Millisecond

// Collect 采集服务器运行指标
// @return: Info
func Collect() Info {
	return Info{
		CPU: cpuInfo(),
		Mem: memInfo(),
		Go:  goInfo(),
		Sys: Sys{
			ComputerName: hostname(),
			ComputerIP:   lib.LocalIP(),
			OSName:       osName(),
			OSArch:       runtime.GOARCH,
		},
		Disks: diskInfo(),
	}
}

// RunTime 格式化运行时长，例如：1天2小时3分钟
// @param: d time.Duration 运行时长
// @return: string
func RunTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// goInfo Go 运行时内存与版本信息
func goInfo() Go {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	userDir, _ := os.Getwd()
	total := memStats.Sys / 1024 / 1024
	free := memStats.HeapIdle / 1024 / 1024
	var used uint64
	if total > free {
		used = total - free
	}
	return Go{
		Name:      "GoLang",
		Version:   runtime.Version(),
		StartTime: startTime.Format("2006-01-02 15:04:05"),
		RunTime:   RunTime(time.Since(startTime)),
		Home:      runtime.GOROOT(),
		UserDir:   userDir,
		Total:     total,
		Used:      used,
		Free:      free,
		Usage:     percent(used, total),
	}
}

// hostname 服务器名称
func hostname() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

// percent 计算百分比并保留两位小数
func percent(part, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return round2(float64(part) / float64(total) * 100)
}

// round2 四舍五入保留两位小数
func round2(value float64) float64 {
	result, err := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	if err != nil {
		return value
	}
	return result
}
