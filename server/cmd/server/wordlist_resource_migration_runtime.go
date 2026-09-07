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

func runWordlistResourceMigrationFromRuntime() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	databaseConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wordlist resource migration failed: load database configuration")
		return 1
	}
	report, err := app.RunWordlistResourceIdentityMigration(ctx, databaseConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wordlist resource migration failed: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		return 1
	}
	return 0
}
