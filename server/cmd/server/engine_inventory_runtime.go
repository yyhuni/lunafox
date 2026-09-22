package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yyhuni/lunafox/server/internal/app"
	"github.com/yyhuni/lunafox/server/internal/config"
)

// runEngineInventoryFromRuntime is intentionally argument-free. The host
// upgrader invokes this fixed command inside the running Server container;
// target paths, refs, and component IDs are all resolved from Server-owned DB
// and exact package-cache state.
func runEngineInventoryFromRuntime() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine inventory failed: load config: %v\n", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	inventory, err := app.RunEngineInventory(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine inventory failed: %v\n", err)
		return 1
	}
	encoded, err := json.Marshal(inventory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine inventory failed: encode output: %v\n", err)
		return 1
	}
	if _, err := os.Stdout.Write(append(encoded, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Engine inventory failed: write output: %v\n", err)
		return 1
	}
	return 0
}
