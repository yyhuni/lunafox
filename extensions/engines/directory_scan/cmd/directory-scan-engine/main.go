package main

import (
	"context"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
	directoryscanruntime "github.com/yyhuni/lunafox/engines/directory_scan/runtime"
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
	return directoryscanruntime.New().Execute(ctx, execution)
}
