package portscanruntime

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

type naabuCommand struct {
	label   string
	args    []string
	timeout time.Duration
}

func buildNaabuActiveCommand(hostsFilePath, outputFilePath string, hostCount uint64, config portscancontract.NaabuActiveConfig) (naabuCommand, error) {
	args := []string{"-exclude-cdn", "-warm-up-time", "5", "-verify", "-list", hostsFilePath, "-json", "-silent", "-o", outputFilePath}
	if config.Threads > 0 {
		args = append(args, "-c", strconv.FormatInt(config.Threads, 10))
	}
	ports := strings.TrimSpace(config.Ports)
	topPorts := strings.TrimSpace(config.TopPorts)
	switch strings.TrimSpace(config.PortMode) {
	case "top":
		if topPorts != "" {
			args = append(args, "-top-ports", topPorts)
		}
	case "custom":
		if ports == "" {
			return naabuCommand{}, fmt.Errorf("ports is required when port-mode is custom")
		}
		args = append(args, "-p", ports)
	case "top-and-custom":
		if ports != "" {
			args = append(args, "-p", ports)
		}
		if topPorts != "" {
			args = append(args, "-top-ports", topPorts)
		}
	default:
		return naabuCommand{}, fmt.Errorf("unsupported port-mode %q", config.PortMode)
	}
	if config.Rate > 0 {
		args = append(args, "-rate", strconv.FormatInt(config.Rate, 10))
	}
	return newNaabuCommand(portscancontract.SectionNaabuActive, args, config.Timeout, hostCount)
}

func buildNaabuPassiveCommand(hostsFilePath, outputFilePath string, hostCount uint64, config portscancontract.NaabuPassiveConfig) (naabuCommand, error) {
	args := []string{"-list", hostsFilePath, "-passive", "-json", "-silent", "-o", outputFilePath}
	return newNaabuCommand(portscancontract.SectionNaabuPassive, args, config.Timeout, hostCount)
}

func newNaabuCommand(label string, args []string, perHostTimeoutSeconds int64, hostCount uint64) (naabuCommand, error) {
	timeout, err := perHostProcessTimeout(label, perHostTimeoutSeconds, hostCount)
	if err != nil {
		return naabuCommand{}, err
	}
	return naabuCommand{
		label:   label,
		args:    append([]string(nil), args...),
		timeout: timeout,
	}, nil
}

// perHostProcessTimeout turns the configured per-candidate budget into the
// single naabu process deadline. The process scans the whole candidate file,
// so the deadline is the candidate count times that budget. A product that
// cannot be represented fails before naabu starts instead of wrapping.
func perHostProcessTimeout(label string, perHostSeconds int64, hostCount uint64) (time.Duration, error) {
	if perHostSeconds <= 0 {
		return 0, fmt.Errorf("%s timeout must be positive", label)
	}
	if hostCount == 0 {
		return 0, fmt.Errorf("%s candidate host count must be positive", label)
	}
	if hostCount > uint64(math.MaxInt64) || uint64(perHostSeconds) > uint64(math.MaxInt64)/hostCount {
		return 0, fmt.Errorf("%s timeout budget exceeds process deadline range", label)
	}
	totalSeconds := perHostSeconds * int64(hostCount)
	maxSeconds := int64(math.MaxInt64 / int64(time.Second))
	if totalSeconds > maxSeconds {
		return 0, fmt.Errorf("%s timeout budget exceeds process deadline range", label)
	}
	return time.Duration(totalSeconds) * time.Second, nil
}
