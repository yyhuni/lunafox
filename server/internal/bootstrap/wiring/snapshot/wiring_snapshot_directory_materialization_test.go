package snapshotwiring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDirectoryMaterializationCoordinatorRollsBackEveryFailureStage(t *testing.T) {
	t.Run("snapshot write", func(t *testing.T) {
		db := newDirectoryMaterializationTransactionDB(t, true)
		coordinator := newSnapshotMaterializationCoordinator(db, scanrepo.NewScanRepository(db))

		err := coordinator.Materialize(context.Background(), 7, 9, func(ctx context.Context) error {
			tx := dbtx.Resolve(ctx, db).WithContext(ctx)
			if err := tx.Exec(`INSERT INTO materialization_probe (stage) VALUES ('snapshot')`).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO directory_snapshot (missing_column) VALUES (1)`).Error
		})
		if err == nil {
			t.Fatal("expected Snapshot write failure")
		}
		assertDirectoryMaterializationEffects(t, db, 0, 0, 0, 41)
	})

	t.Run("asset write", func(t *testing.T) {
		db := newDirectoryMaterializationTransactionDB(t, true)
		coordinator := newSnapshotMaterializationCoordinator(db, scanrepo.NewScanRepository(db))
		assetErr := errors.New("asset projection failed")

		err := coordinator.Materialize(context.Background(), 7, 9, func(ctx context.Context) error {
			tx := dbtx.Resolve(ctx, db).WithContext(ctx)
			if err := tx.Exec(`INSERT INTO materialization_probe (stage) VALUES ('asset')`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO directory_snapshot (scan_id, url) VALUES (7, 'https://example.com/admin')`).Error; err != nil {
				return err
			}
			return assetErr
		})
		if !errors.Is(err, assetErr) {
			t.Fatalf("asset failure = %v, want %v", err, assetErr)
		}
		assertDirectoryMaterializationEffects(t, db, 0, 0, 0, 41)
	})

	t.Run("summary refresh", func(t *testing.T) {
		db := newDirectoryMaterializationTransactionDB(t, false)
		coordinator := newSnapshotMaterializationCoordinator(db, scanrepo.NewScanRepository(db))

		err := coordinator.Materialize(context.Background(), 7, 9, func(ctx context.Context) error {
			return insertDirectoryMaterializationEffects(ctx, db, "summary")
		})
		if err == nil || !strings.Contains(err.Error(), "vulnerability_snapshot") {
			t.Fatalf("summary failure = %v, want missing vulnerability_snapshot", err)
		}
		assertDirectoryMaterializationEffects(t, db, 0, 0, 0, 41)
	})
}

func TestDirectoryMaterializationCoordinatorCommitsSnapshotAssetAndSummaryTogether(t *testing.T) {
	db := newDirectoryMaterializationTransactionDB(t, true)
	coordinator := newSnapshotMaterializationCoordinator(db, scanrepo.NewScanRepository(db))

	if err := coordinator.Materialize(context.Background(), 7, 9, func(ctx context.Context) error {
		return insertDirectoryMaterializationEffects(ctx, db, "success")
	}); err != nil {
		t.Fatalf("commit Directory materialization: %v", err)
	}
	assertDirectoryMaterializationEffects(t, db, 1, 1, 1, 1)
}

func insertDirectoryMaterializationEffects(ctx context.Context, root *gorm.DB, stage string) error {
	tx := dbtx.Resolve(ctx, root).WithContext(ctx)
	if err := tx.Exec(`INSERT INTO materialization_probe (stage) VALUES (?)`, stage).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO directory_snapshot (scan_id, url) VALUES (7, 'https://example.com/admin')`).Error; err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO directory (target_id, url) VALUES (9, 'https://example.com/admin')`).Error
}

func assertDirectoryMaterializationEffects(t *testing.T, db *gorm.DB, wantProbe, wantSnapshot, wantAsset, wantSummary int64) {
	t.Helper()
	for table, want := range map[string]int64{
		"materialization_probe": wantProbe,
		"directory_snapshot":    wantSnapshot,
		"directory":             wantAsset,
	} {
		var got int64
		if err := db.Table(table).Count(&got).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s rows = %d, want %d", table, got, want)
		}
	}
	var summary int64
	if err := db.Raw(`SELECT cached_directories_count FROM scan WHERE id = 7`).Scan(&summary).Error; err != nil {
		t.Fatalf("read Directory summary: %v", err)
	}
	if summary != wantSummary {
		t.Fatalf("cached_directories_count = %d, want %d", summary, wantSummary)
	}
}

func newDirectoryMaterializationTransactionDB(t *testing.T, includeCompleteSummarySchema bool) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open Directory materialization sqlite db: %v", err)
	}
	statements := []string{
		`CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			target_id INTEGER NOT NULL,
			deleted_at DATETIME,
			cached_subdomains_count INTEGER NOT NULL DEFAULT 0,
			cached_websites_count INTEGER NOT NULL DEFAULT 0,
			cached_endpoints_count INTEGER NOT NULL DEFAULT 0,
			cached_ips_count INTEGER NOT NULL DEFAULT 0,
			cached_directories_count INTEGER NOT NULL DEFAULT 0,
			cached_screenshots_count INTEGER NOT NULL DEFAULT 0,
			cached_vulns_total INTEGER NOT NULL DEFAULT 0,
			cached_vulns_critical INTEGER NOT NULL DEFAULT 0,
			cached_vulns_high INTEGER NOT NULL DEFAULT 0,
			cached_vulns_medium INTEGER NOT NULL DEFAULT 0,
			cached_vulns_low INTEGER NOT NULL DEFAULT 0,
			stats_updated_at DATETIME
		)`,
		`INSERT INTO scan (id, target_id, cached_directories_count) VALUES (7, 9, 41)`,
		`CREATE TABLE materialization_probe (stage TEXT NOT NULL)`,
		`CREATE TABLE directory (target_id INTEGER NOT NULL, url TEXT NOT NULL)`,
		`CREATE TABLE directory_snapshot (scan_id INTEGER NOT NULL, url TEXT NOT NULL)`,
		`CREATE TABLE subdomain_snapshot (scan_id INTEGER NOT NULL)`,
		`CREATE TABLE website_snapshot (scan_id INTEGER NOT NULL)`,
		`CREATE TABLE endpoint_snapshot (scan_id INTEGER NOT NULL)`,
		`CREATE TABLE host_port_mapping_snapshot (scan_id INTEGER NOT NULL, ip TEXT NOT NULL)`,
		`CREATE TABLE screenshot_snapshot (scan_id INTEGER NOT NULL)`,
	}
	if includeCompleteSummarySchema {
		statements = append(statements, `CREATE TABLE vulnerability_snapshot (scan_id INTEGER NOT NULL, severity TEXT NOT NULL)`)
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Directory materialization schema: %v", err)
		}
	}
	return db
}
