package portscanruntime

import (
	"fmt"
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

func buildNaabuActiveCommand(hostsFilePath, outputFilePath string, config portscancontract.NaabuActiveConfig) (naabuCommand, error) {
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
	return newNaabuCommand(portscancontract.SectionNaabuActive, args, config.Timeout)
}

func buildNaabuPassiveCommand(hostsFilePath, outputFilePath string, config portscancontract.NaabuPassiveConfig) (naabuCommand, error) {
	args := []string{"-list", hostsFilePath, "-passive", "-json", "-silent", "-o", outputFilePath}
	return newNaabuCommand(portscancontract.SectionNaabuPassive, args, config.Timeout)
}

func newNaabuCommand(label string, args []string, timeoutSeconds int64) (naabuCommand, error) {
	if timeoutSeconds <= 0 {
		return naabuCommand{}, fmt.Errorf("%s timeout must be positive", label)
	}
	return naabuCommand{
		label:   label,
		args:    append([]string(nil), args...),
		timeout: time.Duration(timeoutSeconds) * time.Second,
	}, nil
}
