package metrics

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"go.uber.org/zap"
)

// Collector gathers system metrics.
type Collector struct{}

// NewCollector creates a new Collector.
func NewCollector() *Collector {
	return &Collector{}
}

// Sample returns CPU, memory, and disk usage percentages.
func (c *Collector) Sample() (float64, float64, float64) {
	cpuPercent, err := cpuUsagePercent()
	if err != nil {
		logger.Log.Warn("metrics: cpu percent error", zap.Error(err))
	}
	memPercent, err := memUsagePercent()
	if err != nil {
		logger.Log.Warn("metrics: mem percent error", zap.Error(err))
	}
	diskPercent, err := diskUsagePercent("/")
	if err != nil {
		logger.Log.Warn("metrics: disk percent error", zap.Error(err))
	}
	return cpuPercent, memPercent, diskPercent
}

func cpuUsagePercent() (float64, error) {
	values, err := cpu.Percent(0, false)
	if err != nil || len(values) == 0 {
		return 0, err
	}
	return values[0], nil
}

func memUsagePercent() (float64, error) {
	info, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return info.UsedPercent, nil
}

func diskUsagePercent(path string) (float64, error) {
	info, err := disk.Usage(path)
	if err != nil {
		return 0, err
	}
	return info.UsedPercent, nil
}
