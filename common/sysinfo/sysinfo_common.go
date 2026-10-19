package sysinfo

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// mountInfo /proc/mounts 中的一条挂载信息
type mountInfo struct {
	device     string // 设备名
	mountpoint string // 挂载点
	fsType     string // 文件系统类型
}

// cpuInfo 读取 CPU 核心数与使用率/负载
func cpuInfo() CPU {
	info := CPU{Num: runtime.NumCPU()}
	info.Used = cpuUsage()
	info.Avg5, info.Avg15 = loadAvg()
	return info
}

// cpuUsage 采样 200ms 计算 CPU 使用率（%）
func cpuUsage() float64 {
	firstTotal, firstIdle, ok := readStatCPU()
	if !ok {
		return 0
	}
	time.Sleep(cpuSampleInterval)
	secondTotal, secondIdle, ok := readStatCPU()
	if !ok {
		return 0
	}
	totalDelta := secondTotal - firstTotal
	idleDelta := secondIdle - firstIdle
	if totalDelta == 0 || totalDelta < idleDelta {
		return 0
	}
	return round2(float64(totalDelta-idleDelta) / float64(totalDelta) * 100)
}

// memInfo 读取系统内存信息
func memInfo() Mem {
	content, err := readProcFile("/proc/meminfo")
	if err != nil {
		return Mem{}
	}
	values := parseKeyValues(content)
	totalKB := values["MemTotal"]
	freeKB := values["MemAvailable"]
	if freeKB == 0 {
		freeKB = values["MemFree"]
	}
	return buildMem(totalKB, freeKB)
}

// loadAvg 读取系统平均负载
func loadAvg() (avg5 float64, avg15 float64) {
	content, err := readProcFile("/proc/loadavg")
	if err != nil {
		return 0, 0
	}
	return parseLoadAvg(content)
}

// diskInfo 读取磁盘使用情况
func diskInfo() []Disk {
	content, err := readProcFile("/proc/mounts")
	if err != nil {
		return nil
	}
	disks := make([]Disk, 0, 4)
	for _, mount := range parseMounts(content) {
		// 只统计目录（容器内 /etc/resolv.conf 等单文件挂载不展示）
		if info, err := os.Stat(mount.mountpoint); err != nil || !info.IsDir() {
			continue
		}
		total, free, ok := mountUsage(mount.mountpoint)
		if !ok || total == 0 {
			continue
		}
		var used uint64
		if total > free {
			used = total - free
		}
		typeName := "本地磁盘"
		if strings.HasPrefix(mount.device, "/dev/") {
			typeName = "本地固定磁盘"
		}
		disks = append(disks, Disk{
			DirName:     mount.mountpoint,
			SysTypeName: mount.fsType,
			TypeName:    typeName,
			Total:       total,
			Free:        free,
			Used:        used,
			Usage:       percent(used, total),
		})
	}
	return disks
}

// osName 读取操作系统名称
func osName() string {
	content, err := readProcFile("/etc/os-release")
	if err != nil {
		return runtime.GOOS
	}
	return parseOSRelease(content)
}

// parseStatCPU 解析 /proc/stat 中的总体 CPU 行，返回总时间与空闲时间
// @param: content string /proc/stat 内容
// @return: total uint64 总时间, idle uint64 空闲时间（含 iowait）, ok bool 是否解析成功
func parseStatCPU(content string) (total uint64, idle uint64, ok bool) {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		for index, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				continue
			}
			total += value
			if index == 3 || index == 4 { // idle + iowait
				idle += value
			}
		}
		return total, idle, total > 0
	}
	return 0, 0, false
}

// parseKeyValues 解析 `key: value` 形式的文件内容（值只取第一个数字，单位忽略）
// @param: content string 文件内容
// @return: map[string]uint64
func parseKeyValues(content string) map[string]uint64 {
	values := make(map[string]uint64)
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		values[strings.TrimSuffix(fields[0], ":")] = value
	}
	return values
}

// buildMem 由 KB 单位的总内存/可用内存构造内存信息（返回 MB）
// @param: totalKB uint64 总内存（KB）
// @param: freeKB uint64 可用内存（KB）
// @return: Mem
func buildMem(totalKB, freeKB uint64) Mem {
	if totalKB == 0 {
		return Mem{}
	}
	total := totalKB / 1024
	free := freeKB / 1024
	if free > total {
		free = total
	}
	used := total - free
	return Mem{Total: total, Used: used, Free: free, Usage: percent(used, total)}
}

// parseLoadAvg 解析 /proc/loadavg 内容，返回 5 分钟与 15 分钟平均负载
// @param: content string /proc/loadavg 内容，例如 "0.00 0.01 0.05 1/234 5678"
// @return: avg5 float64, avg15 float64
func parseLoadAvg(content string) (avg5 float64, avg15 float64) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return 0, 0
	}
	avg5, _ = strconv.ParseFloat(fields[1], 64)
	avg15, _ = strconv.ParseFloat(fields[2], 64)
	return round2(avg5), round2(avg15)
}

// parseOSRelease 解析 /etc/os-release 内容，返回 PRETTY_NAME
// @param: content string /etc/os-release 内容
// @return: string 操作系统的可读名称
func parseOSRelease(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "PRETTY_NAME=") {
			continue
		}
		return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
	}
	return ""
}

// parseMounts 解析 /proc/mounts 内容，返回真实磁盘挂载点
// 只保留设备以 /dev/ 开头或挂载点为 / 的条目（容器内的 overlay 根目录属于后者）
// @param: content string /proc/mounts 内容
// @return: []mountInfo
func parseMounts(content string) []mountInfo {
	seen := make(map[string]bool)
	mounts := make([]mountInfo, 0, 4)
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		device := fields[0]
		mountpoint := strings.ReplaceAll(fields[1], "\\040", " ")
		if !strings.HasPrefix(device, "/dev/") && mountpoint != "/" {
			continue
		}
		if seen[device] {
			continue
		}
		seen[device] = true
		mounts = append(mounts, mountInfo{device: device, mountpoint: mountpoint, fsType: fields[2]})
	}
	return mounts
}
