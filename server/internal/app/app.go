package app

import (
	"context"
	"embed"

	"github.com/yyhuni/lunafox/server/internal/bootstrap"
	"github.com/yyhuni/lunafox/server/internal/config"
	wordlistmigration "github.com/yyhuni/lunafox/server/internal/modules/catalog/migration"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/upgrader"
)

// Run delegates to bootstrap package for dependency wiring and server lifecycle.
func Run(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) {
	bootstrap.Run(ctx, cfg, migrationsFS)
}

func RunEngineBootstrap(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) error {
	return bootstrap.RunEngineBootstrap(ctx, cfg, migrationsFS)
}

// RunEngineInventory reads the exact installed Engine Package inventory for
// the host upgrader's argument-free live-observation command.
func RunEngineInventory(ctx context.Context, cfg *config.Config) (upgrader.EngineInventory, error) {
	return bootstrap.RunEngineInventory(ctx, cfg)
}

func RunFingerprintBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, corpusPath string) error {
	return bootstrap.RunFingerprintBootstrap(ctx, databaseConfig, corpusPath)
}

// RunWordlistBootstrap imports or validates the immutable default wordlist
// resources without starting the long-lived Server runtime.
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

// RunResidentAgentReadiness reports whether the Agent bound to this deployment
// can accept work. It uses the Server's own database trust boundary, so the
// public lifecycle scripts never need an administrator credential or a JWT.
func RunResidentAgentReadiness(ctx context.Context, databaseConfig *config.DatabaseConfig) (bootstrap.ResidentAgentReadiness, error) {
	return bootstrap.RunResidentAgentReadiness(ctx, databaseConfig)
}

// RunWordlistResourceIdentityMigration performs the explicit development
// Catalog/resource hard cut without starting the long-lived Server.
func RunWordlistResourceIdentityMigration(ctx context.Context, databaseConfig *config.DatabaseConfig) (*wordlistmigration.Report, error) {
	return bootstrap.RunWordlistResourceIdentityMigration(ctx, databaseConfig)
}
