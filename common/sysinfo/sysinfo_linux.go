//go:build linux

package sysinfo

import (
	"os"
	"syscall"
)

// readProcFile 读取 /proc 或系统配置文件
// @param: path string 文件路径
// @return: string 文件内容, error 错误
func readProcFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// readStatCPU 读取并解析 /proc/stat 的总体 CPU 行
// @return: total uint64 总时间, idle uint64 空闲时间, ok bool 是否解析成功
func readStatCPU() (total uint64, idle uint64, ok bool) {
	content, err := readProcFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	return parseStatCPU(content)
}

// mountUsage 读取挂载点的容量信息（单位 MB）
// @param: mountpoint string 挂载点
// @return: total uint64 总量, free uint64 可用量, ok bool 是否读取成功
func mountUsage(mountpoint string) (total uint64, free uint64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountpoint, &stat); err != nil {
		return 0, 0, false
	}
	if stat.Bsize <= 0 {
		return 0, 0, false
	}
	blockSize := uint64(stat.Bsize)
	return stat.Blocks * blockSize / 1024 / 1024, stat.Bavail * blockSize / 1024 / 1024, true
}
