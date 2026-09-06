package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"

	identitywiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/identity"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// RunAdminPasswordReset initializes only the database path needed for one
// administrator reset. It deliberately does not use initInfra, which starts
// the normal Server's external clients and long-lived runtime dependencies.
func RunAdminPasswordReset(ctx context.Context, databaseConfig *config.DatabaseConfig, migrationsFS embed.FS) (string, error) {
	if ctx == nil {
		return "", errors.New("admin password reset context is required")
	}
	if databaseConfig == nil {
		return "", errors.New("admin password reset database configuration is required")
	}

	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		return "", fmt.Errorf("connect database for admin password reset: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return "", fmt.Errorf("access database connection for admin password reset: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	database.MigrationsFS = migrationsFS
	database.MigrationsPath = "migrations"
	if err := database.RunMigrations(sqlDB); err != nil {
		return "", fmt.Errorf("run migrations for admin password reset: %w", err)
	}

	// The command renderer owns every operator-visible message. A database error
	// must not cause GORM to append unreviewed SQL or values to the terminal.
	return resetAdminPassword(ctx, db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}))
}

func resetAdminPassword(ctx context.Context, db *gorm.DB) (string, error) {
	if db == nil {
		return "", errors.New("admin password reset database is required")
	}
	store := identitywiring.NewIdentityAdminPasswordResetStoreAdapter(identityrepo.NewUserRepository(db))
	service := identityapp.NewAdminPasswordResetService(
		store,
		identityapp.NewAuthPasswordHasher(),
		identityapp.NewCryptoPasswordGenerator(),
	)
	result, err := service.Reset(ctx)
	if err != nil {
		return "", err
	}
	return result.Password, nil
}
