package models

type ServerStatistics struct {
	Cpu         CpuData
	Memory      MemoryData
	LoadAverage LoadAverage
	Network     NetworkData
	Disk        DiskData
}
