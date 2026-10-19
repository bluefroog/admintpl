//go:build !linux

package sysinfo

import (
	"errors"
)

// errNotSupported 平台不支持读取系统指标
var errNotSupported = errors.New("当前平台不支持读取系统指标：仅 Linux 提供 CPU/内存/磁盘采集，其它平台只返回 Go 运行时与主机基础信息")

// readProcFile 非 Linux 平台不读取 /proc
// @param: path string 文件路径
// @return: string, error
func readProcFile(path string) (string, error) {
	return "", errNotSupported
}

// readStatCPU 非 Linux 平台无法读取 /proc/stat
// @return: uint64, uint64, bool
func readStatCPU() (uint64, uint64, bool) {
	return 0, 0, false
}

// mountUsage 非 Linux 平台无法读取磁盘容量
// @param: mountpoint string 挂载点
// @return: uint64, uint64, bool
func mountUsage(mountpoint string) (uint64, uint64, bool) {
	return 0, 0, false
}
