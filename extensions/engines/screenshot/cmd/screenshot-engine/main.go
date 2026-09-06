package main

import (
	"context"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
	screenshotruntime "github.com/yyhuni/lunafox/engines/screenshot/runtime"
)

func main() {
	Run(runEngine)
}

func runEngine(ctx context.Context, execution *enginecontract.Execution) error {
	if ctx == nil {
		return fmt.Errorf("execution context is required")
	}
	if execution == nil {
		return fmt.Errorf("execution is required")
	}
	runtimeInstance := screenshotruntime.New()
	return runtimeInstance.Execute(ctx, execution)
}
