package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/yyhuni/lunafox/server/internal/app"
	"github.com/yyhuni/lunafox/server/internal/bootstrap"
	"github.com/yyhuni/lunafox/server/internal/config"
)

type residentAgentReadinessConfigLoader func() (*config.DatabaseConfig, error)

type residentAgentReadinessRunner func(context.Context, *config.DatabaseConfig) (bootstrap.ResidentAgentReadiness, error)

func runResidentAgentReadinessFromRuntime() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runResidentAgentReadinessRuntime(
		ctx,
		os.Stdout,
		os.Stderr,
		config.LoadDatabaseConfig,
		func(ctx context.Context, databaseConfig *config.DatabaseConfig) (bootstrap.ResidentAgentReadiness, error) {
			return app.RunResidentAgentReadiness(ctx, databaseConfig)
		},
	)
}

// runResidentAgentReadinessRuntime stays on the bounded database path: the
// lifecycle probe must not need HTTP, JWT, Redis, or administrator credentials.
func runResidentAgentReadinessRuntime(
	ctx context.Context,
	stdout io.Writer,
	stderr io.Writer,
	loadDatabaseConfig residentAgentReadinessConfigLoader,
	probe residentAgentReadinessRunner,
) int {
	if ctx == nil || stdout == nil || stderr == nil || loadDatabaseConfig == nil || probe == nil {
		if stderr != nil {
			_, _ = io.WriteString(stderr, "resident Agent readiness check failed\n")
		}
		return 1
	}

	databaseConfig, err := loadDatabaseConfig()
	if err != nil {
		_, _ = io.WriteString(stderr, "resident Agent readiness check failed to load the database configuration\n")
		return 1
	}

	return runResidentAgentReadinessCommand(ctx, stdout, stderr, func(ctx context.Context) (bootstrap.ResidentAgentReadiness, error) {
		return probe(ctx, databaseConfig)
	})
}
