package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	serverdatabase "github.com/yyhuni/lunafox/server/internal/database"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	globalAssetSearchPostgresDSNEnv = "LUNAFOX_GLOBAL_ASSET_SEARCH_POSTGRES_DSN"
	globalAssetSearchPostgresDBName = "lunafox_global_asset_search"
)

type globalAssetSearchPostgresFixture struct {
	websiteIDs  []int
	endpointIDs []int
}

// TestGlobalAssetSearchPostgresIntegration exercises the PostgreSQL-only
// transaction timeout and actual current-state tables. It is opt-in because it
// rebuilds the named disposable database from the baseline migration.
func TestGlobalAssetSearchPostgresIntegration(t *testing.T) {
	db, sqlDB := openGlobalAssetSearchPostgres(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	fixture := seedGlobalAssetSearchPostgresFixture(t, ctx, sqlDB)

	websiteRepo := assetrepo.NewWebsiteRepository(db)
	endpointRepo := assetrepo.NewEndpointRepository(db)
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`title="Global search"`)
	if err != nil {
		t.Fatalf("parse global search probe: %v", err)
	}
	query := assetapp.GlobalAssetSearchStoreQuery{AST: ast, PageSize: 10}

	beforeTimeout := globalAssetSearchPostgresStatementTimeout(t, ctx, sqlDB)
	websites, err := websiteRepo.SearchGlobalWebsites(ctx, query)
	if err != nil {
		t.Fatalf("search current Websites: %v", err)
	}
	if got := globalAssetSearchWebsiteIDs(websites); strings.Join(intStrings(got), ",") != strings.Join(intStrings(fixture.websiteIDs), ",") {
		t.Fatalf("Website search must return only active current rows: got=%v want=%v", got, fixture.websiteIDs)
	}
	if afterTimeout := globalAssetSearchPostgresStatementTimeout(t, ctx, sqlDB); afterTimeout != beforeTimeout {
		t.Fatalf("successful search leaked statement_timeout: before=%q after=%q", beforeTimeout, afterTimeout)
	}

	endpoints, err := endpointRepo.SearchGlobalEndpoints(ctx, query)
	if err != nil {
		t.Fatalf("search current Endpoints: %v", err)
	}
	if got := globalAssetSearchEndpointIDs(endpoints); strings.Join(intStrings(got), ",") != strings.Join(intStrings(fixture.endpointIDs), ",") {
		t.Fatalf("Endpoint search must use only the Endpoint current table: got=%v want=%v", got, fixture.endpointIDs)
	}

	globalAssetSearchPostgresInstallSlowWebsiteView(t, ctx, sqlDB)
	timeoutItems, err := websiteRepo.SearchGlobalWebsites(ctx, query)
	if !errors.Is(err, assetapp.ErrGlobalAssetSearchTimeout) {
		t.Fatalf("slow PostgreSQL statement error = %v, want %v", err, assetapp.ErrGlobalAssetSearchTimeout)
	}
	if timeoutItems != nil {
		t.Fatalf("timed-out search must not return partial rows: %+v", timeoutItems)
	}
	if afterTimeout := globalAssetSearchPostgresStatementTimeout(t, ctx, sqlDB); afterTimeout != beforeTimeout {
		t.Fatalf("failed search leaked statement_timeout: before=%q after=%q", beforeTimeout, afterTimeout)
	}
}

func openGlobalAssetSearchPostgres(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(globalAssetSearchPostgresDSNEnv))
	if dsn == "" {
		t.Skip("set " + globalAssetSearchPostgresDSNEnv + " to run the PostgreSQL 18 global asset search integration gate")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open global asset search PostgreSQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open global asset search SQL handle: %v", err)
	}
	// Keep a small pool so the migration driver's retained connection cannot
	// block fixture setup; SET LOCAL leakage is checked after every search.
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var databaseName string
	if err := sqlDB.QueryRowContext(ctx, "SELECT current_database()").Scan(&databaseName); err != nil {
		t.Fatalf("read global asset search database name: %v", err)
	}
	if databaseName != globalAssetSearchPostgresDBName {
		t.Fatalf("refusing destructive global asset search database %q; want %q", databaseName, globalAssetSearchPostgresDBName)
	}
	var publicTables int
	if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'`).Scan(&publicTables); err != nil {
		t.Fatalf("inspect global asset search database: %v", err)
	}
	if publicTables != 0 {
		t.Fatalf("global asset search database must start empty, found %d public tables", publicTables)
	}

	previousFS, previousPath := serverdatabase.MigrationsFS, serverdatabase.MigrationsPath
	serverdatabase.MigrationsFS = migrationsFS
	serverdatabase.MigrationsPath = "migrations"
	t.Cleanup(func() {
		serverdatabase.MigrationsFS = previousFS
		serverdatabase.MigrationsPath = previousPath
	})
	if err := serverdatabase.RunMigrations(sqlDB); err != nil {
		t.Fatalf("apply empty baseline for global asset search: %v", err)
	}
	t.Cleanup(func() {
		if err := serverdatabase.MigrateDown(sqlDB); err != nil {
			t.Errorf("tear down global asset search baseline: %v", err)
		}
	})
	return db, sqlDB
}

func seedGlobalAssetSearchPostgresFixture(t *testing.T, ctx context.Context, db *sql.DB) globalAssetSearchPostgresFixture {
	t.Helper()
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	activeOne := globalAssetSearchPostgresCreateTarget(t, ctx, db, "global-search-active-one.example", nil)
	activeTwo := globalAssetSearchPostgresCreateTarget(t, ctx, db, "global-search-active-two.example", nil)
	deletedAt := now.Add(-time.Hour)
	deleted := globalAssetSearchPostgresCreateTarget(t, ctx, db, "global-search-deleted.example", &deletedAt)

	firstWebsite := globalAssetSearchPostgresInsertWebsite(t, ctx, db, activeOne, "https://active-one.example/global-search-probe", now)
	secondWebsite := globalAssetSearchPostgresInsertWebsite(t, ctx, db, activeTwo, "https://active-two.example/global-search-probe", now.Add(-time.Second))
	_ = globalAssetSearchPostgresInsertWebsite(t, ctx, db, deleted, "https://deleted.example/global-search-probe", now.Add(-2*time.Second))

	firstEndpoint := globalAssetSearchPostgresInsertEndpoint(t, ctx, db, activeOne, "https://active-one.example/api/global-search-probe", now)
	_ = globalAssetSearchPostgresInsertEndpoint(t, ctx, db, deleted, "https://deleted.example/api/global-search-probe", now.Add(-time.Second))

	var scanID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO scan (target_id, scan_workflow_id, input_source, trigger_type, status)
		VALUES ($1, 'global-asset-search-integration', 'scan_snapshot', 'manual', 'initiated')
		RETURNING id
	`, activeOne).Scan(&scanID); err != nil {
		t.Fatalf("create Snapshot fixture Scan: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO website_snapshot (scan_id, url, host, title, tech)
		VALUES ($1, 'https://snapshot-only.example/global-search-probe', 'snapshot-only.example', 'Snapshot only', $2::varchar(100)[])
	`, scanID, pq.Array([]string{"Nginx"})); err != nil {
		t.Fatalf("create Snapshot-only Website fixture: %v", err)
	}

	return globalAssetSearchPostgresFixture{
		websiteIDs:  []int{firstWebsite, secondWebsite},
		endpointIDs: []int{firstEndpoint},
	}
}

func globalAssetSearchPostgresCreateTarget(t *testing.T, ctx context.Context, db *sql.DB, name string, deletedAt *time.Time) int {
	t.Helper()
	var id int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO target (name, type, deleted_at)
		VALUES ($1, 'domain', $2)
		RETURNING id
	`, name, deletedAt).Scan(&id); err != nil {
		t.Fatalf("create Target %q: %v", name, err)
	}
	return id
}

func globalAssetSearchPostgresInsertWebsite(t *testing.T, ctx context.Context, db *sql.DB, targetID int, url string, createdAt time.Time) int {
	t.Helper()
	var id int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO website (target_id, url, host, title, tech, status_code, response_headers, response_body, created_at)
		VALUES ($1, $2, $3, 'Global search Website', $4::varchar(100)[], 200, 'website headers', 'website body', $5)
		RETURNING id
	`, targetID, url, fmt.Sprintf("website-%d.example", targetID), pq.Array([]string{"Nginx"}), createdAt).Scan(&id); err != nil {
		t.Fatalf("insert Website for Target %d: %v", targetID, err)
	}
	return id
}

func globalAssetSearchPostgresInsertEndpoint(t *testing.T, ctx context.Context, db *sql.DB, targetID int, url string, createdAt time.Time) int {
	t.Helper()
	var id int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO endpoint (target_id, url, host, title, tech, status_code, response_headers, response_body, created_at)
		VALUES ($1, $2, $3, 'Global search Endpoint', $4::varchar(100)[], 200, 'endpoint headers', 'endpoint body', $5)
		RETURNING id
	`, targetID, url, fmt.Sprintf("endpoint-%d.example", targetID), pq.Array([]string{"Nginx"}), createdAt).Scan(&id); err != nil {
		t.Fatalf("insert Endpoint for Target %d: %v", targetID, err)
	}
	return id
}

func globalAssetSearchPostgresStatementTimeout(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	var value string
	if err := db.QueryRowContext(ctx, "SHOW statement_timeout").Scan(&value); err != nil {
		t.Fatalf("read statement_timeout: %v", err)
	}
	return value
}

func globalAssetSearchPostgresInstallSlowWebsiteView(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	const baseTable = "website_global_asset_search_base"
	if _, err := db.ExecContext(ctx, `
		CREATE FUNCTION global_asset_search_test_sleep() RETURNS boolean
		LANGUAGE plpgsql VOLATILE AS $$
		BEGIN
			PERFORM pg_sleep(6);
			RETURN TRUE;
		END;
		$$
	`); err != nil {
		t.Fatalf("create slow search function: %v", err)
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE website RENAME TO "+baseTable); err != nil {
		t.Fatalf("rename Website table for timeout fixture: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DROP VIEW IF EXISTS website"); err != nil {
			t.Errorf("drop slow Website view: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), "ALTER TABLE IF EXISTS "+baseTable+" RENAME TO website"); err != nil {
			t.Errorf("restore Website table after timeout fixture: %v", err)
		}
		if _, err := db.ExecContext(context.Background(), "DROP FUNCTION IF EXISTS global_asset_search_test_sleep()"); err != nil {
			t.Errorf("drop slow search function: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "CREATE VIEW website AS SELECT * FROM "+baseTable+" WHERE global_asset_search_test_sleep()"); err != nil {
		t.Fatalf("create slow Website view: %v", err)
	}
}

func globalAssetSearchWebsiteIDs(items []assetdomain.Website) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func globalAssetSearchEndpointIDs(items []assetdomain.Endpoint) []int {
	ids := make([]int, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func intStrings(values []int) []string {
	out := make([]string, len(values))
	for index, value := range values {
		out[index] = fmt.Sprintf("%d", value)
	}
	return out
}
