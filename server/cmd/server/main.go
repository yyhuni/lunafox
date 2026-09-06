package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yyhuni/lunafox/server/internal/app"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/pkg"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
	command, err := parseServerCommand(os.Args)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}

	if command.kind == serverCommandResetAdmin {
		if code := runAdminPasswordResetFromRuntime(); code != 0 {
			os.Exit(code)
		}
		return
	}
	if command.kind == serverCommandWordlistResourceMigration {
		if code := runWordlistResourceMigrationFromRuntime(); code != 0 {
			os.Exit(code)
		}
		return
	}
	if command.kind == serverCommandFingerprintBootstrap {
		databaseConfig, err := config.LoadDatabaseConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fingerprint bootstrap failed: load database configuration")
			os.Exit(1)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := app.RunFingerprintBootstrap(ctx, databaseConfig, command.fingerprintBootstrapPath); err != nil {
			fmt.Fprintf(os.Stderr, "Fingerprint bootstrap failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if command.kind == serverCommandWordlistBootstrap {
		databaseConfig, err := config.LoadDatabaseConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Wordlist bootstrap failed: load database configuration")
			os.Exit(1)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := app.RunWordlistBootstrap(ctx, databaseConfig, os.Getenv("WORDLISTS_BASE_PATH"), command.wordlistManifestPath, command.wordlistSourcePath); err != nil {
			fmt.Fprintf(os.Stderr, "Wordlist bootstrap failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if command.kind == serverCommandAgentBootstrap {
		databaseConfig, err := config.LoadDatabaseConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Agent bootstrap failed: load database configuration")
			os.Exit(1)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := app.RunAgentBootstrap(ctx, databaseConfig, command.agentCredentialsPath, command.agentHostname, command.agentVersion); err != nil {
			fmt.Fprintf(os.Stderr, "Agent bootstrap failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Load configuration for the long-lived Server paths only.
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := pkg.InitLogger(&pkg.LogConfig{
		Level: cfg.Log.Level,
	}); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer pkg.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if command.kind == serverCommandEngineBootstrap {
		if err := app.RunEngineBootstrap(ctx, cfg, migrationsFS); err != nil {
			fmt.Printf("Engine bootstrap failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
	app.Run(ctx, cfg, migrationsFS)
}
