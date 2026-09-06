package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestTargetTombstoneAndEnsureCleanupIsIdempotent(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	targetID := seedTargetCleanupCommandTarget(t, db, "delete-once.example", false)

	deleted, err := repository.TombstoneAndEnsureCleanup(context.Background(), targetID)
	if err != nil {
		t.Fatalf("TombstoneAndEnsureCleanup() error = %v", err)
	}
	if !deleted {
		t.Fatal("first delete must report an active-to-tombstone transition")
	}
	assertTargetCleanupCommandState(t, db, targetID, true, 1)

	deleted, err = repository.TombstoneAndEnsureCleanup(context.Background(), targetID)
	if err != nil {
		t.Fatalf("repeated TombstoneAndEnsureCleanup() error = %v", err)
	}
	if deleted {
		t.Fatal("repeated delete must not report another transition")
	}
	assertTargetCleanupCommandState(t, db, targetID, true, 1)
}

func TestTargetTombstoneAndEnsureCleanupRejectsNeverExistingTarget(t *testing.T) {
	repository := NewTargetRepository(openTargetCleanupCommandDB(t))
	if _, err := repository.TombstoneAndEnsureCleanup(context.Background(), 404); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("TombstoneAndEnsureCleanup() error = %v, want record not found", err)
	}
}

func TestTargetTombstoneAndEnsureCleanupRegistersAJobForAnExistingTombstone(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	targetID := seedTargetCleanupCommandTarget(t, db, "already-tombstoned.example", true)

	deleted, err := repository.TombstoneAndEnsureCleanup(context.Background(), targetID)
	if err != nil {
		t.Fatalf("TombstoneAndEnsureCleanup() error = %v", err)
	}
	if deleted {
		t.Fatal("existing tombstone must not report another active-to-tombstone transition")
	}
	assertTargetCleanupCommandState(t, db, targetID, true, 1)
}

func TestTargetTombstoneAndEnsureCleanupRollsBackWhenJobRegistrationFails(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	targetID := seedTargetCleanupCommandTarget(t, db, "rollback-single.example", false)
	installTargetCleanupJobRejectTrigger(t, db)

	if _, err := repository.TombstoneAndEnsureCleanup(context.Background(), targetID); err == nil {
		t.Fatal("TombstoneAndEnsureCleanup() succeeded after cleanup job registration failure")
	}
	assertTargetCleanupCommandState(t, db, targetID, false, 0)
}

func TestBatchTargetTombstoneAndEnsureCleanupValidatesEverythingBeforeWriting(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	firstID := seedTargetCleanupCommandTarget(t, db, "batch-first.example", false)
	secondID := seedTargetCleanupCommandTarget(t, db, "batch-second.example", false)

	if _, err := repository.BatchTombstoneAndEnsureCleanup(context.Background(), []int{secondID, firstID, secondID, 999}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("BatchTombstoneAndEnsureCleanup() error = %v, want record not found", err)
	}
	assertTargetCleanupCommandState(t, db, firstID, false, 0)
	assertTargetCleanupCommandState(t, db, secondID, false, 0)

	count, err := repository.BatchTombstoneAndEnsureCleanup(context.Background(), []int{secondID, firstID, secondID})
	if err != nil {
		t.Fatalf("BatchTombstoneAndEnsureCleanup() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("deleted count = %d, want 2 unique active transitions", count)
	}
	assertTargetCleanupCommandState(t, db, firstID, true, 1)
	assertTargetCleanupCommandState(t, db, secondID, true, 1)

	count, err = repository.BatchTombstoneAndEnsureCleanup(context.Background(), []int{firstID, secondID})
	if err != nil {
		t.Fatalf("repeated BatchTombstoneAndEnsureCleanup() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("repeated deleted count = %d, want 0", count)
	}
}

func TestBatchTargetTombstoneAndEnsureCleanupRollsBackAllTargetsWhenJobRegistrationFails(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	firstID := seedTargetCleanupCommandTarget(t, db, "rollback-batch-one.example", false)
	secondID := seedTargetCleanupCommandTarget(t, db, "rollback-batch-two.example", false)
	installTargetCleanupJobRejectTrigger(t, db)

	if _, err := repository.BatchTombstoneAndEnsureCleanup(context.Background(), []int{secondID, firstID}); err == nil {
		t.Fatal("BatchTombstoneAndEnsureCleanup() succeeded after cleanup job registration failure")
	}
	assertTargetCleanupCommandState(t, db, firstID, false, 0)
	assertTargetCleanupCommandState(t, db, secondID, false, 0)
}

func TestTargetCleanupBatchIDNormalizationPreservesInputDeduplicationAndUsesLockOrder(t *testing.T) {
	unique := stableUniqueTargetIDs([]int{9, 3, 9, 7, 3})
	if len(unique) != 3 || unique[0] != 9 || unique[1] != 3 || unique[2] != 7 {
		t.Fatalf("stable unique IDs = %v, want [9 3 7]", unique)
	}
	lockOrder := targetIDsInLockOrder(unique)
	if len(lockOrder) != 3 || lockOrder[0] != 3 || lockOrder[1] != 7 || lockOrder[2] != 9 {
		t.Fatalf("lock-order IDs = %v, want [3 7 9]", lockOrder)
	}
}

func TestTargetTombstoneCommandsDoNotTouchRelatedRowsSynchronously(t *testing.T) {
	trace := &targetCleanupSQLTrace{}
	db := openTargetCleanupCommandDBWithLogger(t, trace)
	repository := NewTargetRepository(db)
	firstID := seedTargetCleanupCommandTarget(t, db, "sql-trace-one.example", false)
	secondID := seedTargetCleanupCommandTarget(t, db, "sql-trace-two.example", false)

	trace.Reset()
	if _, err := repository.TombstoneAndEnsureCleanup(context.Background(), firstID); err != nil {
		t.Fatalf("TombstoneAndEnsureCleanup() error = %v", err)
	}
	assertTargetCleanupSQLTraceOnlyTouchesCommandTables(t, trace)

	trace.Reset()
	if _, err := repository.BatchTombstoneAndEnsureCleanup(context.Background(), []int{secondID, firstID, secondID}); err != nil {
		t.Fatalf("BatchTombstoneAndEnsureCleanup() error = %v", err)
	}
	assertTargetCleanupSQLTraceOnlyTouchesCommandTables(t, trace)
}

func TestTargetUpdateCannotClearDeletedAtThroughModelSave(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	targetID := seedTargetCleanupCommandTarget(t, db, "already-deleted.example", true)

	err := repository.Update(&catalogdomain.Target{
		ID:   targetID,
		Name: "renamed.example",
		Type: catalogdomain.TargetTypeDomain,
	})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Update() error = %v, want record not found", err)
	}
	var target model.Target
	if err := db.Where("id = ?", targetID).First(&target).Error; err != nil {
		t.Fatal(err)
	}
	if target.DeletedAt == nil || target.Name != "already-deleted.example" {
		t.Fatalf("deleted target was mutated: %+v", target)
	}
}

func TestTargetUpdateWritesOnlyBusinessFields(t *testing.T) {
	db := openTargetCleanupCommandDB(t)
	repository := NewTargetRepository(db)
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	lastScannedAt := createdAt.Add(time.Hour)
	persisted := model.Target{
		Name:          "before.example",
		Type:          catalogdomain.TargetTypeDomain,
		CreatedAt:     createdAt,
		LastScannedAt: &lastScannedAt,
	}
	if err := db.Create(&persisted).Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}

	unexpectedCreatedAt := createdAt.Add(24 * time.Hour)
	unexpectedLastScannedAt := lastScannedAt.Add(24 * time.Hour)
	if err := repository.Update(&catalogdomain.Target{
		ID:            persisted.ID,
		Name:          "after.example",
		Type:          catalogdomain.TargetTypeDomain,
		CreatedAt:     unexpectedCreatedAt,
		LastScannedAt: &unexpectedLastScannedAt,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	var updated model.Target
	if err := db.Where("id = ?", persisted.ID).First(&updated).Error; err != nil {
		t.Fatalf("read updated target: %v", err)
	}
	if updated.Name != "after.example" || updated.Type != catalogdomain.TargetTypeDomain {
		t.Fatalf("business fields were not updated: %+v", updated)
	}
	if !updated.CreatedAt.Equal(createdAt) || updated.LastScannedAt == nil || !updated.LastScannedAt.Equal(lastScannedAt) || updated.DeletedAt != nil {
		t.Fatalf("update overwrote lifecycle fields: %+v", updated)
	}
}

func openTargetCleanupCommandDB(t *testing.T) *gorm.DB {
	t.Helper()
	return openTargetCleanupCommandDBWithLogger(t, nil)
}

func openTargetCleanupCommandDBWithLogger(t *testing.T, sqlLogger gormlogger.Interface) *gorm.DB {
	t.Helper()
	config := &gorm.Config{}
	if sqlLogger != nil {
		config.Logger = sqlLogger
	}
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), config)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE target (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at DATETIME,
			last_scanned_at DATETIME,
			deleted_at DATETIME
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE target_cleanup_job (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_id INTEGER NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'pending',
			retry_count INTEGER NOT NULL DEFAULT 0,
			next_retry_at DATETIME NOT NULL,
			last_error TEXT NOT NULL DEFAULT '',
			completed_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func installTargetCleanupJobRejectTrigger(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`
		CREATE TRIGGER reject_target_cleanup_job
		BEFORE INSERT ON target_cleanup_job
		BEGIN
			SELECT RAISE(ABORT, 'target cleanup job rejected');
		END;
	`).Error; err != nil {
		t.Fatalf("install cleanup job reject trigger: %v", err)
	}
}

func seedTargetCleanupCommandTarget(t *testing.T, db *gorm.DB, name string, deleted bool) int {
	t.Helper()
	target := model.Target{Name: name, Type: catalogdomain.TargetTypeDomain}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	if deleted {
		if err := db.Model(&model.Target{}).Where("id = ?", target.ID).Update("deleted_at", gorm.Expr("CURRENT_TIMESTAMP")).Error; err != nil {
			t.Fatal(err)
		}
	}
	return target.ID
}

func assertTargetCleanupCommandState(t *testing.T, db *gorm.DB, targetID int, wantDeleted bool, wantJobs int) {
	t.Helper()
	var target model.Target
	if err := db.Where("id = ?", targetID).First(&target).Error; err != nil {
		t.Fatal(err)
	}
	if (target.DeletedAt != nil) != wantDeleted {
		t.Fatalf("target %d deleted = %t, want %t", targetID, target.DeletedAt != nil, wantDeleted)
	}
	var jobs int64
	if err := db.Model(&model.TargetCleanupJob{}).Where("target_id = ?", targetID).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if jobs != int64(wantJobs) {
		t.Fatalf("target %d jobs = %d, want %d", targetID, jobs, wantJobs)
	}
}

type targetCleanupSQLTrace struct {
	mu         sync.Mutex
	statements []string
}

func (trace *targetCleanupSQLTrace) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return trace
}

func (*targetCleanupSQLTrace) Info(context.Context, string, ...interface{}) {}

func (*targetCleanupSQLTrace) Warn(context.Context, string, ...interface{}) {}

func (*targetCleanupSQLTrace) Error(context.Context, string, ...interface{}) {}

func (trace *targetCleanupSQLTrace) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	statement, _ := fc()
	if strings.TrimSpace(statement) == "" {
		return
	}
	trace.mu.Lock()
	defer trace.mu.Unlock()
	trace.statements = append(trace.statements, statement)
}

func (trace *targetCleanupSQLTrace) Reset() {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	trace.statements = nil
}

func (trace *targetCleanupSQLTrace) Statements() []string {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	return append([]string(nil), trace.statements...)
}

func assertTargetCleanupSQLTraceOnlyTouchesCommandTables(t *testing.T, trace *targetCleanupSQLTrace) {
	t.Helper()
	statements := trace.Statements()
	if len(statements) == 0 {
		t.Fatal("expected synchronous delete SQL trace")
	}
	for _, statement := range statements {
		for _, table := range []string{
			"scheduled_scan",
			"scheduled_scan_occurrence",
			"scan",
			"scan_task",
			"subdomain",
			"host_port_mapping",
			"website",
			"endpoint",
			"directory",
			"screenshot",
			"vulnerability",
		} {
			if sqlStatementTouchesTable(statement, table) {
				t.Fatalf("synchronous delete SQL touched %s: %s", table, statement)
			}
		}
	}
}

func sqlStatementTouchesTable(statement, table string) bool {
	normalized := strings.ToLower(strings.NewReplacer(
		"`", " ",
		"\"", " ",
		"(", " ",
		")", " ",
		",", " ",
		";", " ",
	).Replace(statement))
	for _, prefix := range []string{"from ", "join ", "update ", "into ", "delete from "} {
		if strings.Contains(normalized, prefix+table+" ") {
			return true
		}
	}
	return false
}
