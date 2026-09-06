package main

import (
	"context"
	"fmt"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
	portscanruntime "github.com/yyhuni/lunafox/engines/port_scan/runtime"
)

func main() {
	Run(runPortScan)
}

func runPortScan(ctx context.Context, execution *enginecontract.Execution) error {
	if ctx == nil {
		return fmt.Errorf("execution context is required")
	}
	if execution.Config.NaabuActive.Enabled && execution.Config.NaabuActive.PortMode == "custom" && strings.TrimSpace(execution.Config.NaabuActive.Ports) == "" {
		return fmt.Errorf("ports is required when port-mode is custom")
	}
	if execution.Input.Subdomains == nil {
		return fmt.Errorf("subdomains input handle is required")
	}
	subdomainsPath, err := execution.Input.Subdomains.Path(ctx)
	if err != nil {
		return fmt.Errorf("materialize subdomains input: %w", err)
	}
	runtimeInstance := portscanruntime.New()
	artifacts, err := runtimeInstance.Execute(
		ctx,
		execution.Config,
		execution.Target,
		subdomainsPath,
		execution.Workspace,
		execution.Progress,
	)
	if err != nil {
		return err
	}
	return runtimeInstance.ReportResults(ctx, execution.Progress, execution.Results.HostPorts, artifacts)
}
