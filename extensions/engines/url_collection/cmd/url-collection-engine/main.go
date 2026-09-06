package main

import (
	"context"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
	urlcollectionruntime "github.com/yyhuni/lunafox/engines/url_collection/runtime"
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
	return urlcollectionruntime.New().Execute(ctx, execution)
}
