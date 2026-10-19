package domain

// ServerCPU CPU 信息
type ServerCPU struct {
	Num   int     `json:"cpuNum"`   // 核心数
	Used  float64 `json:"cpuUsed"`  // 使用率（%）
	Avg5  float64 `json:"cpuAvg5"`  // 5 分钟平均负载
	Avg15 float64 `json:"cpuAvg15"` // 15 分钟平均负载
}

// ServerMem 内存信息（单位 MB）
type ServerMem struct {
	Total uint64  `json:"memTotal"` // 总内存
	Used  uint64  `json:"memUsed"`  // 已用内存
	Free  uint64  `json:"memFree"`  // 剩余内存
	Usage float64 `json:"memUsage"` // 使用率（%）
}

// ServerGo Go 运行时信息（单位 MB）
type ServerGo struct {
	Name      string  `json:"goName"`      // 语言环境
	Version   string  `json:"goVersion"`   // 版本
	StartTime string  `json:"goStartTime"` // 启动时间
	RunTime   string  `json:"goRunTime"`   // 运行时长
	Home      string  `json:"goHome"`      // 安装路径
	UserDir   string  `json:"goUserDir"`   // 项目路径
	Total     uint64  `json:"goTotal"`     // 分配的总内存
	Used      uint64  `json:"goUsed"`      // 已使用的内存
	Free      uint64  `json:"goFree"`      // 剩余的内存
	Usage     float64 `json:"goUsage"`     // 使用率（%）
}

// ServerSys 服务器信息
type ServerSys struct {
	ComputerName string `json:"sysComputerName"` // 服务器名称
	ComputerIP   string `json:"sysComputerIp"`   // 服务器 IP
	OSName       string `json:"sysOsName"`       // 操作系统
	OSArch       string `json:"sysOsArch"`       // 系统架构
}

// ServerDisk 磁盘信息（单位 MB）
type ServerDisk struct {
	DirName     string  `json:"dirName"`     // 盘符路径
	SysTypeName string  `json:"sysTypeName"` // 文件系统
	TypeName    string  `json:"typeName"`    // 盘符类型
	Total       uint64  `json:"total"`       // 总大小
	Free        uint64  `json:"free"`        // 剩余大小
	Used        uint64  `json:"used"`        // 已用大小
	Usage       float64 `json:"usage"`       // 使用率（%）
}

// ServerInfo 服务监控信息
type ServerInfo struct {
	CPU   ServerCPU    `json:"cpu"`      // CPU 信息
	Mem   ServerMem    `json:"mem"`      // 内存信息
	Go    ServerGo     `json:"go"`       // Go 运行时信息
	Sys   ServerSys    `json:"sys"`      // 服务器信息
	Disks []ServerDisk `json:"sysFiles"` // 磁盘信息
}
