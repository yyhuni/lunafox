package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// MigrationsFS holds the embedded migration files
// This will be set from main.go where the embed directive is valid
var MigrationsFS embed.FS

// MigrationsPath is the path within the embedded filesystem
var MigrationsPath = "migrations"

// RunMigrations runs all pending database migrations
func RunMigrations(db *sql.DB) error {
	pkg.Info("Running database migrations...")

	// Create source driver from embedded files
	source, err := iofs.New(MigrationsFS, MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		pkg.Info("No migrations applied yet")
	} else {
		pkg.Info("Database migrations completed",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	return nil
}

// MigrateDown rolls back the last migration
func MigrateDown(db *sql.DB) error {
	pkg.Info("Rolling back last migration...")

	source, err := iofs.New(MigrationsFS, MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	pkg.Info("Migration rollback completed")
	return nil
}

// MigrateToVersion migrates to a specific version
func MigrateToVersion(db *sql.DB, version uint) error {
	pkg.Info("Migrating to version", zap.Uint("version", version))

	source, err := iofs.New(MigrationsFS, MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	pkg.Info("Migration completed", zap.Uint("version", version))
	return nil
}

// GetMigrationVersion returns the current migration version
func GetMigrationVersion(db *sql.DB) (uint, bool, error) {
	source, err := iofs.New(MigrationsFS, MigrationsPath)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration source: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	return version, dirty, nil
}
