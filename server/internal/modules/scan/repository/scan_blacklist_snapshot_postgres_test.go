package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
	blacklistmodel "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository/persistence"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	scanBlacklistSnapshotPostgresDSNEnv = "LUNAFOX_POSTGRES_SCAN_BLACKLIST_DSN"
	scanBlacklistSnapshotPostgresSchema = "scan_blacklist_snapshot_contract"
)

// TestScanBlacklistSnapshotPostgresUpdateAndScanLockOrdering exercises the
// PostgreSQL FOR SHARE/FOR UPDATE interaction that SQLite cannot model.
func TestScanBlacklistSnapshotPostgresUpdateAndScanLockOrdering(t *testing.T) {
	t.Run("update first is visible", func(t *testing.T) {
		db := openScanBlacklistSnapshotPostgresDB(t)
		targetID := 1
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{"target.before.example"})
		policyStore := blacklistrepo.NewPolicyRepository(db)
		global, err := policyStore.GetGlobal(context.Background())
		if err != nil {
			t.Fatalf("get global policy: %v", err)
		}
		etag, err := blacklistdomain.ETag(global.Patterns)
		if err != nil {
			t.Fatalf("derive global etag: %v", err)
		}
		if _, changed, err := policyStore.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, []string{"global.after.example"}); err != nil || !changed {
			t.Fatalf("replace global policy: changed=%t err=%v", changed, err)
		}

		scan := createPostgresScanWithPolicy(t, NewScanRepository(db), targetID, policyStore.ReadEffectivePatternsForScan)
		patterns, err := NewScanRepository(db).LoadBlacklistSnapshot(context.Background(), scan.ID)
		if err != nil {
			t.Fatalf("load Scan snapshot: %v", err)
		}
		want := []string{"global.after.example", "target.before.example"}
		if !reflect.DeepEqual(patterns, want) {
			t.Fatalf("update-first snapshot = %#v, want %#v", patterns, want)
		}
	})

	t.Run("scan first blocks both scope updates and freezes no mixed state", func(t *testing.T) {
		db := openScanBlacklistSnapshotPostgresDB(t)
		targetID := 2
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{"target.before.example"})
		policyStore := blacklistrepo.NewPolicyRepository(db)
		scanRepository := NewScanRepository(db)
		globalETag := postgresPolicyETag(t, policyStore, blacklistdomain.ScopeGlobal, nil)
		targetETag := postgresPolicyETag(t, policyStore, blacklistdomain.ScopeTarget, &targetID)

		locked := make(chan struct{})
		release := make(chan struct{})
		defer closeIfOpen(release)
		scanDone := make(chan error, 1)
		go func() {
			scan := &ScanCreateRecord{TargetID: targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
			scanDone <- scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, func(ctx context.Context, id int) ([]string, error) {
				patterns, err := policyStore.ReadEffectivePatternsForScan(ctx, id)
				if err != nil {
					return nil, err
				}
				close(locked)
				<-release
				return patterns, nil
			}, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
		}()
		awaitPostgresSignal(t, locked, "Scan policy shared locks")

		globalPatchDone := make(chan error, 1)
		targetPatchDone := make(chan error, 1)
		go func() {
			_, _, err := policyStore.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, globalETag, []string{"global.after.example"})
			globalPatchDone <- err
		}()
		go func() {
			_, _, err := policyStore.ReplacePatterns(context.Background(), blacklistdomain.ScopeTarget, &targetID, targetETag, []string{"target.after.example"})
			targetPatchDone <- err
		}()
		assertPostgresBlocked(t, globalPatchDone, "global Policy PATCH")
		assertPostgresBlocked(t, targetPatchDone, "Target Policy PATCH")
		close(release)
		if err := awaitPostgresResult(t, scanDone, "Scan create"); err != nil {
			t.Fatalf("Scan create failed: %v", err)
		}
		if err := awaitPostgresResult(t, globalPatchDone, "global Policy PATCH"); err != nil {
			t.Fatalf("global Policy PATCH failed: %v", err)
		}
		if err := awaitPostgresResult(t, targetPatchDone, "Target Policy PATCH"); err != nil {
			t.Fatalf("Target Policy PATCH failed: %v", err)
		}

		var scanID int
		if err := db.Table("scan").Select("MAX(id)").Scan(&scanID).Error; err != nil {
			t.Fatalf("read Scan id: %v", err)
		}
		patterns, err := scanRepository.LoadBlacklistSnapshot(context.Background(), scanID)
		if err != nil {
			t.Fatalf("load locked snapshot: %v", err)
		}
		want := []string{"global.before.example", "target.before.example"}
		if !reflect.DeepEqual(patterns, want) {
			t.Fatalf("scan-first snapshot mixed or changed: %#v, want %#v", patterns, want)
		}
	})
}

func TestScanBlacklistSnapshotPostgresConcurrentSharedLocksAndRollbackRelease(t *testing.T) {
	t.Run("concurrent scans retain read committed isolation", func(t *testing.T) {
		db := openScanBlacklistSnapshotPostgresDB(t)
		targetID := 3
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.example"})
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{"target.example"})
		policyStore := blacklistrepo.NewPolicyRepository(db)
		scanRepository := NewScanRepository(db)
		locked := make(chan struct{}, 2)
		release := make(chan struct{})
		defer closeIfOpen(release)
		results := make(chan error, 2)
		resolver := func(ctx context.Context, id int) ([]string, error) {
			patterns, err := policyStore.ReadEffectivePatternsForScan(ctx, id)
			if err != nil {
				return nil, err
			}
			var isolation string
			if err := dbtx.Resolve(ctx, db).Raw("SHOW transaction_isolation").Scan(&isolation).Error; err != nil {
				return nil, err
			}
			if isolation != "read committed" {
				return nil, fmt.Errorf("transaction isolation = %q, want read committed", isolation)
			}
			locked <- struct{}{}
			<-release
			return patterns, nil
		}
		for range 2 {
			go func() {
				scan := &ScanCreateRecord{TargetID: targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
				results <- scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, resolver, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
			}()
		}
		awaitPostgresSignal(t, locked, "first Scan shared locks")
		awaitPostgresSignal(t, locked, "second Scan shared locks")
		close(release)
		for range 2 {
			if err := awaitPostgresResult(t, results, "concurrent Scan create"); err != nil {
				t.Fatalf("concurrent Scan create failed: %v", err)
			}
		}
	})

	t.Run("rollback releases waiting update lock", func(t *testing.T) {
		db := openScanBlacklistSnapshotPostgresDB(t)
		targetID := 4
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"global.before.example"})
		seedPostgresScanBlacklistPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{})
		policyStore := blacklistrepo.NewPolicyRepository(db)
		scanRepository := NewScanRepository(db)
		globalETag := postgresPolicyETag(t, policyStore, blacklistdomain.ScopeGlobal, nil)
		locked := make(chan struct{})
		release := make(chan struct{})
		defer closeIfOpen(release)
		wantRollback := errors.New("force Scan rollback")
		scanDone := make(chan error, 1)
		go func() {
			scan := &ScanCreateRecord{TargetID: targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
			scanDone <- scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, func(ctx context.Context, id int) ([]string, error) {
				if _, err := policyStore.ReadEffectivePatternsForScan(ctx, id); err != nil {
					return nil, err
				}
				close(locked)
				<-release
				return nil, wantRollback
			}, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
		}()
		awaitPostgresSignal(t, locked, "Scan policy shared locks")
		patchDone := make(chan error, 1)
		go func() {
			_, _, err := policyStore.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, globalETag, []string{"global.after.example"})
			patchDone <- err
		}()
		assertPostgresBlocked(t, patchDone, "global Policy PATCH before rollback")
		close(release)
		if err := awaitPostgresResult(t, scanDone, "rolled-back Scan create"); !errors.Is(err, wantRollback) {
			t.Fatalf("Scan rollback error = %v, want %v", err, wantRollback)
		}
		if err := awaitPostgresResult(t, patchDone, "global Policy PATCH after rollback"); err != nil {
			t.Fatalf("Policy PATCH remained blocked after rollback: %v", err)
		}
		var scans, snapshots int64
		if err := db.Table("scan").Count(&scans).Error; err != nil {
			t.Fatalf("count rolled-back Scans: %v", err)
		}
		if err := db.Table("scan_blacklist_snapshot").Count(&snapshots).Error; err != nil {
			t.Fatalf("count rolled-back snapshots: %v", err)
		}
		if scans != 0 || snapshots != 0 {
			t.Fatalf("rollback left rows: scans=%d snapshots=%d", scans, snapshots)
		}
	})
}

func TestScanCreatePostgresTargetDeletionLockOrdering(t *testing.T) {
	t.Run("delete commits before scan create", func(t *testing.T) {
		db := openScanBlacklistSnapshotPostgresDB(t)
		if err := db.Exec("UPDATE target SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", 1).Error; err != nil {
			t.Fatalf("tombstone Target: %v", err)
		}

		resolverCalled := false
		scan := &ScanCreateRecord{TargetID: 1, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
		err := NewScanRepository(db).CreateWithScanTasksAndPlans(context.Background(), scan, func(context.Context, int) ([]string, error) {
			resolverCalled = true
			return []string{}, nil
		}, func(int, int, *scandomain.CreateScanTask) error { return nil })
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("Scan create error = %v, want record not found", err)
		}
		if resolverCalled {
			t.Fatal("Scan create resolved dependent state after Target deletion")
		}
		var count int64
		if err := db.Table("scan").Where("target_id = ?", 1).Count(&count).Error; err != nil {
			t.Fatalf("count Scans: %v", err)
		}
		if count != 0 {
			t.Fatalf("deleted Target created %d Scans", count)
		}
	})

	t.Run("scan create locks Target before delete", func(t *testing.T) {
		db := openScanBlacklistSnapshotPostgresDB(t)
		locked := make(chan struct{})
		release := make(chan struct{})
		defer closeIfOpen(release)
		scanDone := make(chan error, 1)
		go func() {
			scan := &ScanCreateRecord{TargetID: 2, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
			scanDone <- NewScanRepository(db).CreateWithScanTasksAndPlans(context.Background(), scan, func(context.Context, int) ([]string, error) {
				close(locked)
				<-release
				return []string{}, nil
			}, func(int, int, *scandomain.CreateScanTask) error { return nil })
		}()
		awaitPostgresSignal(t, locked, "Scan Target shared lock")

		deleteDone := make(chan error, 1)
		go func() {
			deleteDone <- db.Exec("UPDATE target SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", 2).Error
		}()
		assertPostgresBlocked(t, deleteDone, "Target DELETE")
		close(release)
		if err := awaitPostgresResult(t, scanDone, "Scan create"); err != nil {
			t.Fatalf("Scan create failed: %v", err)
		}
		if err := awaitPostgresResult(t, deleteDone, "Target DELETE"); err != nil {
			t.Fatalf("Target DELETE failed: %v", err)
		}

		var deletedAt *time.Time
		if err := db.Table("target").Select("deleted_at").Where("id = ?", 2).Scan(&deletedAt).Error; err != nil {
			t.Fatalf("read tombstone: %v", err)
		}
		if deletedAt == nil {
			t.Fatal("Target DELETE did not commit after Scan creation released its lock")
		}
		var count int64
		if err := db.Table("scan").Where("target_id = ?", 2).Count(&count).Error; err != nil {
			t.Fatalf("count Scans: %v", err)
		}
		if count != 1 {
			t.Fatalf("Scan create before delete committed %d Scans, want 1", count)
		}
	})
}

func openScanBlacklistSnapshotPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(scanBlacklistSnapshotPostgresDSNEnv))
	if dsn == "" {
		t.Skip("set " + scanBlacklistSnapshotPostgresDSNEnv + " with search_path=" + scanBlacklistSnapshotPostgresSchema + " to run PostgreSQL Scan blacklist concurrency verification")
	}
	if !strings.Contains(dsn, "search_path="+scanBlacklistSnapshotPostgresSchema) {
		t.Fatalf("Scan blacklist PostgreSQL DSN must pin search_path=%s", scanBlacklistSnapshotPostgresSchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + scanBlacklistSnapshotPostgresSchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE SCHEMA " + scanBlacklistSnapshotPostgresSchema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA IF EXISTS " + scanBlacklistSnapshotPostgresSchema + " CASCADE").Error })
	if err := db.Exec(scanBlacklistSnapshotPostgresDDL).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

const scanBlacklistSnapshotPostgresDDL = `
CREATE TABLE target (
	id INTEGER PRIMARY KEY,
	deleted_at TIMESTAMPTZ
);
INSERT INTO target (id, deleted_at) VALUES
	(1, NULL), (2, NULL), (3, NULL), (4, NULL);
CREATE TABLE blacklist_policy (
	id SERIAL PRIMARY KEY,
	scope TEXT NOT NULL,
	target_id INTEGER,
	patterns JSONB NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE scan (
	id SERIAL PRIMARY KEY,
	target_id INTEGER NOT NULL,
	scan_workflow_id TEXT NOT NULL,
	configuration JSONB,
	input_source TEXT NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
	trigger_type TEXT NOT NULL DEFAULT 'manual' CHECK (trigger_type IN ('manual', 'scheduled', 'ai')),
	status TEXT,
	results_dir TEXT,
	container_ids TEXT[],
	agent_id INTEGER,
	assignment_mode TEXT,
	error_message TEXT,
	failure_kind TEXT,
	progress INTEGER,
	current_stage TEXT,
	stage_progress JSONB,
	created_at TIMESTAMPTZ,
	stopped_at TIMESTAMPTZ,
	deleted_at TIMESTAMPTZ,
	cached_subdomains_count INTEGER,
	cached_websites_count INTEGER,
	cached_endpoints_count INTEGER,
	cached_ips_count INTEGER,
	cached_directories_count INTEGER,
	cached_screenshots_count INTEGER,
	cached_vulns_total INTEGER,
	cached_vulns_critical INTEGER,
	cached_vulns_high INTEGER,
	cached_vulns_medium INTEGER,
	cached_vulns_low INTEGER,
	stats_updated_at TIMESTAMPTZ
);
CREATE TABLE scan_blacklist_snapshot (
	scan_id INTEGER PRIMARY KEY,
	patterns JSONB NOT NULL
);`

func seedPostgresScanBlacklistPolicy(t *testing.T, db *gorm.DB, scope blacklistdomain.Scope, targetID *int, patterns []string) {
	t.Helper()
	canonical, err := blacklistdomain.CanonicalizePolicyPatterns(patterns)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := blacklistdomain.CanonicalPatternsJSON(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&blacklistmodel.Policy{
		Scope:     string(scope),
		TargetID:  targetID,
		Patterns:  datatypes.JSON(append([]byte(nil), payload...)),
		UpdatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func postgresPolicyETag(t *testing.T, policyStore *blacklistrepo.PolicyRepository, scope blacklistdomain.Scope, targetID *int) string {
	t.Helper()
	var policy *blacklistdomain.Policy
	var err error
	if scope == blacklistdomain.ScopeGlobal {
		policy, err = policyStore.GetGlobal(context.Background())
	} else {
		policy, err = policyStore.GetTarget(context.Background(), *targetID)
	}
	if err != nil {
		t.Fatal(err)
	}
	etag, err := blacklistdomain.ETag(policy.Patterns)
	if err != nil {
		t.Fatal(err)
	}
	return etag
}

func createPostgresScanWithPolicy(t *testing.T, scanRepository *ScanRepository, targetID int, resolver func(context.Context, int) ([]string, error)) *ScanCreateRecord {
	t.Helper()
	scan := &ScanCreateRecord{TargetID: targetID, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}
	if err := scanRepository.CreateWithScanTasksAndPlans(context.Background(), scan, resolver, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil }); err != nil {
		t.Fatalf("create Scan: %v", err)
	}
	return scan
}

func awaitPostgresSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func assertPostgresBlocked(t *testing.T, result <-chan error, description string) {
	t.Helper()
	select {
	case err := <-result:
		t.Fatalf("%s completed before Scan released locks: %v", description, err)
	case <-time.After(150 * time.Millisecond):
	}
}

func awaitPostgresResult(t *testing.T, result <-chan error, description string) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
		return nil
	}
}

func closeIfOpen(channel chan struct{}) {
	select {
	case <-channel:
	default:
		close(channel)
	}
}
