package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/yyhuni/lunafox/server/internal/app"
	"github.com/yyhuni/lunafox/server/internal/config"
)

type adminPasswordResetConfigLoader func() (*config.DatabaseConfig, error)

type adminPasswordResetRunner func(context.Context, *config.DatabaseConfig) (string, error)

func runAdminPasswordResetFromRuntime() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runAdminPasswordResetRuntime(
		ctx,
		os.Stdout,
		os.Stderr,
		config.LoadDatabaseConfig,
		func(ctx context.Context, databaseConfig *config.DatabaseConfig) (string, error) {
			return app.RunAdminPasswordReset(ctx, databaseConfig, migrationsFS)
		},
	)
}

// runAdminPasswordResetRuntime keeps the operational command on the narrow
// database-only path. It intentionally never calls the long-lived Server runner.
func runAdminPasswordResetRuntime(
	ctx context.Context,
	stdout io.Writer,
	stderr io.Writer,
	loadDatabaseConfig adminPasswordResetConfigLoader,
	reset adminPasswordResetRunner,
) int {
	if ctx == nil || stdout == nil || stderr == nil || loadDatabaseConfig == nil || reset == nil {
		if stderr != nil {
			_, _ = io.WriteString(stderr, "administrator password reset failed\n")
		}
		return 1
	}

	databaseConfig, err := loadDatabaseConfig()
	if err != nil {
		_, _ = io.WriteString(stderr, "administrator password reset failed\n")
		return 1
	}

	return runAdminPasswordResetCommand(ctx, stdout, stderr, func(ctx context.Context) (string, error) {
		return reset(ctx, databaseConfig)
	})
}
