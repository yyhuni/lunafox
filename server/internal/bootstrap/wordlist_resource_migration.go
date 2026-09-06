package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	wordlistmigration "github.com/yyhuni/lunafox/server/internal/modules/catalog/migration"
)

// RunWordlistResourceIdentityMigration runs the explicit development data
// cutover without starting HTTP, Redis, Agent or Worker runtime dependencies.
// The database must already have the squashed baseline tables; an empty or
// uninitialized database is intentionally rejected rather than silently
// creating a snapshot that has nothing to migrate.
func RunWordlistResourceIdentityMigration(ctx context.Context, databaseConfig *config.DatabaseConfig) (*wordlistmigration.Report, error) {
	if ctx == nil {
		return nil, errors.New("wordlist resource migration context is required")
	}
	if databaseConfig == nil {
		return nil, errors.New("wordlist resource migration database configuration is required")
	}
	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		return nil, fmt.Errorf("connect database for wordlist resource migration: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("access database connection for wordlist resource migration: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	runner := wordlistmigration.NewWordlistResourceIdentityMigration(db)
	return runner.Apply(ctx)
}
