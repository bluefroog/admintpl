package service

import (
	"github.com/bluefroog/admintpl/app/monitor/domain"
	"github.com/bluefroog/admintpl/common/sysinfo"
)

// Server 服务监控服务
type Server struct{}

// Info 采集服务器运行指标
// @author: [bluefrog](https://github.com/freewu)
// @function: Info
// @description: 采集 CPU/内存/Go 运行时/主机/磁盘信息（仅使用标准库，不依赖 gopsutil）
// @return: *domain.ServerInfo 服务器监控信息
func (s Server) Info() *domain.ServerInfo {
	info := sysinfo.Collect()
	disks := make([]domain.ServerDisk, 0, len(info.Disks))
	for _, disk := range info.Disks {
		disks = append(disks, domain.ServerDisk{
			DirName:     disk.DirName,
			SysTypeName: disk.SysTypeName,
			TypeName:    disk.TypeName,
			Total:       disk.Total,
			Free:        disk.Free,
			Used:        disk.Used,
			Usage:       disk.Usage,
		})
	}
	return &domain.ServerInfo{
		CPU: domain.ServerCPU{
			Num:   info.CPU.Num,
			Used:  info.CPU.Used,
			Avg5:  info.CPU.Avg5,
			Avg15: info.CPU.Avg15,
		},
		Mem: domain.ServerMem{
			Total: info.Mem.Total,
			Used:  info.Mem.Used,
			Free:  info.Mem.Free,
			Usage: info.Mem.Usage,
		},
		Go: domain.ServerGo{
			Name:      info.Go.Name,
			Version:   info.Go.Version,
			StartTime: info.Go.StartTime,
			RunTime:   info.Go.RunTime,
			Home:      info.Go.Home,
			UserDir:   info.Go.UserDir,
			Total:     info.Go.Total,
			Used:      info.Go.Used,
			Free:      info.Go.Free,
			Usage:     info.Go.Usage,
		},
		Sys: domain.ServerSys{
			ComputerName: info.Sys.ComputerName,
			ComputerIP:   info.Sys.ComputerIP,
			OSName:       info.Sys.OSName,
			OSArch:       info.Sys.OSArch,
		},
		Disks: disks,
	}
}
