package main

import (
	"context"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
	websitediscoveryruntime "github.com/yyhuni/lunafox/engines/website_discovery/runtime"
)

func main() {
	Run(runWebsiteDiscovery)
}

func runWebsiteDiscovery(ctx context.Context, execution *enginecontract.Execution) error {
	if ctx == nil {
		return fmt.Errorf("execution context is required")
	}
	if execution == nil {
		return fmt.Errorf("execution is required")
	}
	if execution.Input.HostPorts == nil {
		return fmt.Errorf("hostPorts input handle is required")
	}
	hostPortsPath, err := execution.Input.HostPorts.Path(ctx)
	if err != nil {
		return fmt.Errorf("materialize hostPorts input: %w", err)
	}
	runtimeInstance := websitediscoveryruntime.New()
	artifacts, err := runtimeInstance.Execute(
		ctx,
		execution.Config,
		execution.Target,
		hostPortsPath,
		execution.Workspace,
		execution.Progress,
	)
	if err != nil {
		return err
	}
	return runtimeInstance.ReportResults(ctx, execution.Progress, execution.Results.Websites, artifacts)
}
