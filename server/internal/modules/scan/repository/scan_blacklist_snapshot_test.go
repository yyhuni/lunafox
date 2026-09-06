package repository

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScanRepositoryCreateWithScanTasksAndPlansFreezesOneBlacklistSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-blacklist-snapshot-freeze?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan tables: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	repository := NewScanRepository(db)
	patterns := []string{"*.example.com", "192.0.2.0/24"}
	scan := &ScanCreateRecord{TargetID: 1, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}

	if err := repository.CreateWithScanTasksAndPlans(context.Background(), scan, func(_ context.Context, targetID int) ([]string, error) {
		if targetID != 1 {
			t.Fatalf("resolver target id = %d, want 1", targetID)
		}
		return append([]string{}, patterns...), nil
	}, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil }); err != nil {
		t.Fatalf("CreateWithScanTasksAndPlans failed: %v", err)
	}

	patterns[0] = "changed.example.com"
	loaded, err := repository.LoadBlacklistSnapshot(context.Background(), scan.ID)
	if err != nil {
		t.Fatalf("LoadBlacklistSnapshot failed: %v", err)
	}
	if want := []string{"*.example.com", "192.0.2.0/24"}; !slices.Equal(loaded, want) {
		t.Fatalf("frozen patterns = %#v, want %#v", loaded, want)
	}
	var count int64
	if err := db.Table("scan_blacklist_snapshot").Where("scan_id = ?", scan.ID).Count(&count).Error; err != nil {
		t.Fatalf("count snapshot rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("snapshot rows for Scan %d = %d, want 1", scan.ID, count)
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlansRejectsInvalidEffectivePatternsBeforeScanInsert(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-blacklist-snapshot-invalid?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan tables: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	repository := NewScanRepository(db)
	scan := &ScanCreateRecord{TargetID: 1, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}

	err = repository.CreateWithScanTasksAndPlans(context.Background(), scan, func(context.Context, int) ([]string, error) {
		return []string{"Example.COM"}, nil
	}, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
	if !errors.Is(err, blacklistdomain.ErrNonCanonicalPatterns) {
		t.Fatalf("CreateWithScanTasksAndPlans error = %v, want non-canonical patterns", err)
	}
	assertNoScanOrBlacklistSnapshot(t, db)
}

func TestScanRepositoryCreateWithScanTasksAndPlansRollsBackResolverAndScanWhenSnapshotWriteFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-blacklist-snapshot-write-failure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	withoutSnapshot := strings.Split(createWithScanTasksScanDDL, "CREATE TABLE scan_blacklist_snapshot")[0]
	if err := db.Exec(withoutSnapshot).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec("CREATE TABLE resolver_observation (id INTEGER PRIMARY KEY AUTOINCREMENT)").Error; err != nil {
		t.Fatalf("create resolver observation table: %v", err)
	}
	repository := NewScanRepository(db)
	scan := &ScanCreateRecord{TargetID: 1, ScanWorkflowID: "default", InputSource: scandomain.InputSourceScanSnapshot, TriggerType: scandomain.ScanTriggerTypeManual, Status: "pending"}

	err = repository.CreateWithScanTasksAndPlans(context.Background(), scan, func(ctx context.Context, _ int) ([]string, error) {
		if err := dbtx.Resolve(ctx, db).Exec("INSERT INTO resolver_observation DEFAULT VALUES").Error; err != nil {
			return nil, err
		}
		return []string{}, nil
	}, func(_ int, _ int, _ *scandomain.CreateScanTask) error { return nil })
	if err == nil {
		t.Fatal("expected snapshot write failure")
	}
	assertNoScanOrBlacklistSnapshot(t, db)
	var observations int64
	if err := db.Table("resolver_observation").Count(&observations).Error; err != nil {
		t.Fatalf("count resolver observations: %v", err)
	}
	if observations != 0 {
		t.Fatalf("resolver side effect committed outside Scan transaction: %d", observations)
	}
}

func TestScanRepositoryCreateWithScanTasksAndPlansRollsBackSnapshotWhenTaskFinalizerFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-blacklist-snapshot-finalizer-failure?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(createWithScanTasksScanDDL).Error; err != nil {
		t.Fatalf("create scan table: %v", err)
	}
	setupActiveTargetsForScanCreate(t, db)
	if err := db.Exec(createWithScanTasksScanTaskDDL).Error; err != nil {
		t.Fatalf("create scan task table: %v", err)
	}
	repository := NewScanRepository(db)
	scan := &ScanCreateRecord{
		TargetID:       1,
		ScanWorkflowID: "default",
		InputSource:    scandomain.InputSourceScanSnapshot,
		TriggerType:    scandomain.ScanTriggerTypeManual,
		Status:         "pending",
		ScanTasks: []CreateScanTaskRecord{{
			StageOrder: 1,
			StageID:    "discovery",
			StepOrder:  1,
			StepID:     "subdomains",
			EngineID:   "engine.lunafox.subdomain_discovery",
			Status:     "pending",
		}},
	}
	want := errors.New("plan task failed")
	err = repository.CreateWithScanTasksAndPlans(context.Background(), scan, func(context.Context, int) ([]string, error) {
		return []string{"example.com"}, nil
	}, func(_ int, _ int, _ *scandomain.CreateScanTask) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("CreateWithScanTasksAndPlans error = %v, want %v", err, want)
	}
	assertNoScanOrBlacklistSnapshot(t, db)
	var taskCount int64
	if err := db.Table("scan_task").Count(&taskCount).Error; err != nil {
		t.Fatalf("count scan tasks: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("finalizer failure left tasks: %d", taskCount)
	}
}

func TestScanRepositoryLoadBlacklistSnapshotRejectsMissingAndCorruptRowsWithoutPolicyJoin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-blacklist-snapshot-read?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec("CREATE TABLE scan_blacklist_snapshot (scan_id INTEGER PRIMARY KEY, patterns TEXT NOT NULL)").Error; err != nil {
		t.Fatalf("create snapshot table: %v", err)
	}
	repository := NewScanRepository(db)
	if _, err := repository.LoadBlacklistSnapshot(context.Background(), 1); !errors.Is(err, ErrScanBlacklistSnapshotNotFound) {
		t.Fatalf("missing snapshot error = %v, want not found", err)
	}
	if err := db.Exec(`INSERT INTO scan_blacklist_snapshot (scan_id, patterns) VALUES (2, '["192.0.2.1"]')`).Error; err != nil {
		t.Fatalf("insert valid snapshot: %v", err)
	}
	patterns, err := repository.LoadBlacklistSnapshot(context.Background(), 2)
	if err != nil {
		t.Fatalf("load valid snapshot: %v", err)
	}
	if !slices.Equal(patterns, []string{"192.0.2.1"}) {
		t.Fatalf("loaded patterns = %#v", patterns)
	}
	if err := db.Exec(`INSERT INTO scan_blacklist_snapshot (scan_id, patterns) VALUES (3, 'null')`).Error; err != nil {
		t.Fatalf("insert corrupt snapshot: %v", err)
	}
	if _, err := repository.LoadBlacklistSnapshot(context.Background(), 3); !errors.Is(err, ErrScanBlacklistSnapshotDataIntegrity) {
		t.Fatalf("corrupt snapshot error = %v, want data integrity", err)
	}
}

func assertNoScanOrBlacklistSnapshot(t *testing.T, db *gorm.DB) {
	t.Helper()
	var scanCount int64
	if err := db.Table("scan").Count(&scanCount).Error; err != nil {
		t.Fatalf("count scans: %v", err)
	}
	if scanCount != 0 {
		t.Fatalf("unexpected committed scans: %d", scanCount)
	}
	var snapshotCount int64
	if err := db.Table("scan_blacklist_snapshot").Count(&snapshotCount).Error; err != nil && !strings.Contains(err.Error(), "no such table") {
		t.Fatalf("count snapshots: %v", err)
	}
	if snapshotCount != 0 {
		t.Fatalf("unexpected committed snapshots: %d", snapshotCount)
	}
}
