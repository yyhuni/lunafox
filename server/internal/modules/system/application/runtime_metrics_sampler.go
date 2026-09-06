package application

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const bytesPerGiB = 1024 * 1024 * 1024

type procCPUStat struct {
	idle  uint64
	total uint64
}

type osRuntimeMetricsSampler struct {
	diskPath string

	mu      sync.Mutex
	lastCPU *procCPUStat
}

func NewOSRuntimeMetricsSampler(diskPath string) RuntimeMetricSampler {
	normalized := strings.TrimSpace(diskPath)
	if normalized == "" {
		if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
			normalized = cwd
		} else {
			normalized = "."
		}
	}
	return &osRuntimeMetricsSampler{diskPath: normalized}
}

func (sampler *osRuntimeMetricsSampler) Sample(context.Context) (RuntimeMetricSample, error) {
	now := time.Now().UTC()
	cpuPercent, cpuOK := sampler.sampleCPUPercent()
	memoryPercent, memoryOK := sampleMemoryPercent()
	diskPercent, diskOK := sampleDiskPercent(sampler.diskPath)
	if !cpuOK && !memoryOK && !diskOK {
		return RuntimeMetricSample{}, ErrRuntimeMetricsUnavailable
	}
	return RuntimeMetricSample{
		SampledAt: now,
		CPU:       cpuPercent,
		Memory:    memoryPercent,
		Disk:      diskPercent,
		Scope:     RuntimeMetricScopeRuntime,
	}, nil
}

func (sampler *osRuntimeMetricsSampler) sampleCPUPercent() (float64, bool) {
	stat, err := readProcCPUStat("/proc/stat")
	if err != nil {
		return 0, false
	}
	sampler.mu.Lock()
	defer sampler.mu.Unlock()
	if sampler.lastCPU == nil {
		sampler.lastCPU = &stat
		return 0, true
	}
	prev := *sampler.lastCPU
	sampler.lastCPU = &stat
	if stat.total <= prev.total || stat.idle < prev.idle {
		return 0, true
	}
	totalDelta := stat.total - prev.total
	idleDelta := stat.idle - prev.idle
	if totalDelta == 0 || idleDelta > totalDelta {
		return 0, true
	}
	return (float64(totalDelta-idleDelta) / float64(totalDelta)) * 100, true
}

type osRuntimeMetricsMetadata struct {
	diskPath string
}

func NewOSRuntimeMetricsMetadataProvider(diskPath string) RuntimeMetricMetadataProvider {
	normalized := strings.TrimSpace(diskPath)
	if normalized == "" {
		if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
			normalized = cwd
		} else {
			normalized = "."
		}
	}
	return osRuntimeMetricsMetadata{diskPath: normalized}
}

func (metadata osRuntimeMetricsMetadata) Capacity(context.Context) (RuntimeMetricCapacity, error) {
	memoryTotalBytes, memoryOK := readProcMemoryTotalBytes("/proc/meminfo")
	diskTotalBytes, diskOK := statfsTotalBytes(metadata.diskPath)
	if !memoryOK && !diskOK {
		return RuntimeMetricCapacity{CPUCores: runtime.NumCPU()}, nil
	}
	return RuntimeMetricCapacity{
		CPUCores:      runtime.NumCPU(),
		MemoryTotalGB: bytesToGiB(memoryTotalBytes),
		DiskTotalGB:   bytesToGiB(diskTotalBytes),
	}, nil
}

func (metadata osRuntimeMetricsMetadata) Source(context.Context) (RuntimeMetricSource, error) {
	hostname, _ := os.Hostname()
	return RuntimeMetricSource{
		Hostname: strings.TrimSpace(hostname),
		DiskPath: metadata.diskPath,
	}, nil
}

func readProcCPUStat(path string) (procCPUStat, error) {
	file, err := os.Open(path)
	if err != nil {
		return procCPUStat{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return procCPUStat{}, err
		}
		return procCPUStat{}, errors.New("missing cpu stat")
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return procCPUStat{}, fmt.Errorf("invalid cpu stat line")
	}
	values := make([]uint64, 0, len(fields)-1)
	for _, raw := range fields[1:] {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return procCPUStat{}, err
		}
		values = append(values, value)
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	return procCPUStat{idle: idle, total: total}, nil
}

func sampleMemoryPercent() (float64, bool) {
	total, available, ok := readProcMemoryBytes("/proc/meminfo")
	if !ok || total <= 0 || available > total {
		return 0, false
	}
	return (float64(total-available) / float64(total)) * 100, true
}

func readProcMemoryTotalBytes(path string) (uint64, bool) {
	total, _, ok := readProcMemoryBytes(path)
	return total, ok
}

func readProcMemoryBytes(path string) (uint64, uint64, bool) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()

	var total uint64
	var available uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = value * 1024
		case "MemAvailable:":
			available = value * 1024
		}
	}
	if total == 0 || available == 0 {
		return 0, 0, false
	}
	return total, available, true
}

func sampleDiskPercent(path string) (float64, bool) {
	total, available, ok := statfsBytes(path)
	if !ok || total <= 0 || available > total {
		return 0, false
	}
	return (float64(total-available) / float64(total)) * 100, true
}

func statfsTotalBytes(path string) (uint64, bool) {
	total, _, ok := statfsBytes(path)
	return total, ok
}

func statfsBytes(path string) (uint64, uint64, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	blockSize := uint64(stat.Bsize)
	return stat.Blocks * blockSize, stat.Bavail * blockSize, true
}

func bytesToGiB(bytes uint64) float64 {
	if bytes == 0 {
		return 0
	}
	return mathRoundOneDecimal(float64(bytes) / bytesPerGiB)
}

func mathRoundOneDecimal(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
