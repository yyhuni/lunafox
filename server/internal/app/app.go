package app

import (
	"context"
	"embed"

	"github.com/yyhuni/lunafox/server/internal/bootstrap"
	"github.com/yyhuni/lunafox/server/internal/config"
	wordlistmigration "github.com/yyhuni/lunafox/server/internal/modules/catalog/migration"
)

// Run delegates to bootstrap package for dependency wiring and server lifecycle.
func Run(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) {
	bootstrap.Run(ctx, cfg, migrationsFS)
}

func RunEngineBootstrap(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) error {
	return bootstrap.RunEngineBootstrap(ctx, cfg, migrationsFS)
}

func RunFingerprintBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, corpusPath string) error {
	return bootstrap.RunFingerprintBootstrap(ctx, databaseConfig, corpusPath)
}

// RunWordlistBootstrap imports the immutable default wordlist resources without
// starting the long-lived Server runtime.
func RunWordlistBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, basePath, manifestPath, sourcePath string) error {
	return bootstrap.RunWordlistBootstrap(ctx, databaseConfig, basePath, manifestPath, sourcePath)
}

// RunAgentBootstrap creates the first internal Agent registration and writes
// its process credentials to the bootstrap-owned temporary file.
func RunAgentBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, credentialsPath, hostname, version string) error {
	return bootstrap.RunAgentBootstrap(ctx, databaseConfig, credentialsPath, hostname, version)
}

// RunAdminPasswordReset performs the bounded database-only administrator reset path.
func RunAdminPasswordReset(ctx context.Context, databaseConfig *config.DatabaseConfig, migrationsFS embed.FS) (string, error) {
	return bootstrap.RunAdminPasswordReset(ctx, databaseConfig, migrationsFS)
}

// RunWordlistResourceIdentityMigration performs the explicit development
// Catalog/resource hard cut without starting the long-lived Server.
func RunWordlistResourceIdentityMigration(ctx context.Context, databaseConfig *config.DatabaseConfig) (*wordlistmigration.Report, error) {
	return bootstrap.RunWordlistResourceIdentityMigration(ctx, databaseConfig)
}
