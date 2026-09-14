package main

import (
	"fmt"
	"os"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
)

// runDatabaseMigrationFromRuntime is the only supported host-side migration
// entry point. The upgrader invokes `server migrate up`; it never passes a
// version, SQL string, path, or down-migration flag from the HTTP boundary.
func runDatabaseMigrationFromRuntime() int {
	databaseConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "database migration failed: load database configuration")
		return 1
	}
	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database migration failed: connect database: %v\n", err)
		return 1
	}
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "database migration failed: open database: %v\n", err)
		return 1
	}
	defer sqlDB.Close()
	database.MigrationsFS = migrationsFS
	database.MigrationsPath = "migrations"
	if err := database.RunMigrations(sqlDB); err != nil {
		fmt.Fprintf(os.Stderr, "database migration failed: %v\n", err)
		return 1
	}
	return 0
}
