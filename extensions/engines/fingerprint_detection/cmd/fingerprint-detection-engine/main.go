package main

import (
	"context"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
	fingerprintdetectionruntime "github.com/yyhuni/lunafox/engines/fingerprint_detection/runtime"
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
	return fingerprintdetectionruntime.New().Execute(ctx, execution)
}
