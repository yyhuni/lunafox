package repository

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const targetCleanupCommandPostgresSchema = "target_cleanup_command_contract"

func TestTargetCleanupCommandConcurrencyPostgres(t *testing.T) {
	db := openTargetCleanupCommandPostgres(t)
	repository := NewTargetRepository(db)

	t.Run("concurrent single delete converges on one job", func(t *testing.T) {
		resetTargetCleanupCommandPostgresFixture(t, db)
		targetID := seedTargetCleanupCommandPostgresTarget(t, db, "single-concurrent.example")
		blocker := lockTargetCleanupCommandPostgresTarget(t, db, targetID)
		operationCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := make(chan struct{})
		results := make(chan targetCleanupDeleteResult, 2)
		var group sync.WaitGroup
		for range 2 {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				deleted, err := repository.TombstoneAndEnsureCleanup(operationCtx, targetID)
				results <- targetCleanupDeleteResult{deleted: deleted, err: err}
			}()
		}
		close(start)
		assertTargetCleanupCommandOperationsBlocked(t, results)
		if err := blocker.Commit().Error; err != nil {
			t.Fatalf("release Target row lock: %v", err)
		}
		group.Wait()
		close(results)

		transitions := 0
		for result := range results {
			if result.err != nil {
				t.Fatalf("concurrent TombstoneAndEnsureCleanup() error = %v", result.err)
			}
			if result.deleted {
				transitions++
			}
		}
		if transitions != 1 {
			t.Fatalf("active-to-tombstone transitions = %d, want 1", transitions)
		}
		assertTargetCleanupCommandPostgresState(t, db, targetID, true, 1)
	})

	t.Run("overlapping batch deletes converge on one job per target", func(t *testing.T) {
		resetTargetCleanupCommandPostgresFixture(t, db)
		firstID := seedTargetCleanupCommandPostgresTarget(t, db, "batch-first.example")
		secondID := seedTargetCleanupCommandPostgresTarget(t, db, "batch-second.example")
		thirdID := seedTargetCleanupCommandPostgresTarget(t, db, "batch-third.example")
		blocker := lockTargetCleanupCommandPostgresTarget(t, db, secondID)
		operationCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		start := make(chan struct{})
		results := make(chan targetCleanupBatchDeleteResult, 2)
		var group sync.WaitGroup
		for _, ids := range [][]int{{thirdID, secondID, thirdID}, {secondID, firstID}} {
			ids := append([]int(nil), ids...)
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				deletedCount, err := repository.BatchTombstoneAndEnsureCleanup(operationCtx, ids)
				results <- targetCleanupBatchDeleteResult{deletedCount: deletedCount, err: err}
			}()
		}
		close(start)
		assertTargetCleanupCommandOperationsBlocked(t, results)
		if err := blocker.Commit().Error; err != nil {
			t.Fatalf("release Target row lock: %v", err)
		}
		group.Wait()
		close(results)

		var transitions int64
		for result := range results {
			if result.err != nil {
				t.Fatalf("concurrent BatchTombstoneAndEnsureCleanup() error = %v", result.err)
			}
			transitions += result.deletedCount
		}
		if transitions != 3 {
			t.Fatalf("batch active-to-tombstone transitions = %d, want 3", transitions)
		}
		for _, targetID := range []int{firstID, secondID, thirdID} {
			assertTargetCleanupCommandPostgresState(t, db, targetID, true, 1)
		}
	})
}

type targetCleanupDeleteResult struct {
	deleted bool
	err     error
}

type targetCleanupBatchDeleteResult struct {
	deletedCount int64
	err          error
}

func openTargetCleanupCommandPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_TARGET_CLEANUP_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_TARGET_CLEANUP_DSN with search_path=" + targetCleanupCommandPostgresSchema + " to run Target cleanup command concurrency verification")
	}
	if !strings.Contains(dsn, "search_path="+targetCleanupCommandPostgresSchema) {
		t.Fatalf("Target cleanup PostgreSQL DSN must pin search_path=%s", targetCleanupCommandPostgresSchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open Target cleanup PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open Target cleanup PostgreSQL SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + targetCleanupCommandPostgresSchema + " CASCADE").Error; err != nil {
		t.Fatalf("drop Target cleanup PostgreSQL schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA " + targetCleanupCommandPostgresSchema).Error; err != nil {
		t.Fatalf("create Target cleanup PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + targetCleanupCommandPostgresSchema + " CASCADE").Error
	})
	for _, statement := range []string{
		`CREATE TABLE target (
			id SERIAL PRIMARY KEY,
			name VARCHAR(300) NOT NULL,
			type VARCHAR(20) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_scanned_at TIMESTAMPTZ,
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE TABLE target_cleanup_job (
			id SERIAL PRIMARY KEY,
			target_id INTEGER NOT NULL UNIQUE REFERENCES target(id),
			status VARCHAR(20) NOT NULL,
			retry_count INTEGER NOT NULL DEFAULT 0,
			next_retry_at TIMESTAMPTZ NOT NULL,
			last_error VARCHAR(2000) NOT NULL DEFAULT '',
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Target cleanup PostgreSQL schema: %v", err)
		}
	}
	return db
}

func resetTargetCleanupCommandPostgresFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("TRUNCATE target_cleanup_job, target RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("reset Target cleanup PostgreSQL fixture: %v", err)
	}
}

func seedTargetCleanupCommandPostgresTarget(t *testing.T, db *gorm.DB, name string) int {
	t.Helper()
	target := model.Target{Name: name, Type: catalogdomain.TargetTypeDomain}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("seed Target cleanup PostgreSQL target: %v", err)
	}
	return target.ID
}

func lockTargetCleanupCommandPostgresTarget(t *testing.T, db *gorm.DB, targetID int) *gorm.DB {
	t.Helper()
	transaction := db.Begin()
	if transaction.Error != nil {
		t.Fatalf("begin Target row lock transaction: %v", transaction.Error)
	}
	t.Cleanup(func() {
		if transaction.Error == nil {
			_ = transaction.Rollback().Error
		}
	})
	var lockedID int
	if err := transaction.Raw("SELECT id FROM target WHERE id = ? FOR UPDATE", targetID).Scan(&lockedID).Error; err != nil {
		t.Fatalf("lock Target row: %v", err)
	}
	if lockedID != targetID {
		t.Fatalf("locked Target ID = %d, want %d", lockedID, targetID)
	}
	return transaction
}

func assertTargetCleanupCommandOperationsBlocked[T any](t *testing.T, results <-chan T) {
	t.Helper()
	select {
	case <-results:
		t.Fatal("Target deletion crossed an uncommitted Target row lock")
	case <-time.After(100 * time.Millisecond):
	}
}

func assertTargetCleanupCommandPostgresState(t *testing.T, db *gorm.DB, targetID int, wantDeleted bool, wantJobs int64) {
	t.Helper()
	var target model.Target
	if err := db.Where("id = ?", targetID).First(&target).Error; err != nil {
		t.Fatalf("read Target %d: %v", targetID, err)
	}
	if (target.DeletedAt != nil) != wantDeleted {
		t.Fatalf("Target %d deleted = %t, want %t", targetID, target.DeletedAt != nil, wantDeleted)
	}
	var jobs int64
	if err := db.Model(&model.TargetCleanupJob{}).Where("target_id = ?", targetID).Count(&jobs).Error; err != nil {
		t.Fatalf("count Target cleanup Jobs: %v", err)
	}
	if jobs != wantJobs {
		t.Fatalf("Target %d cleanup Jobs = %d, want %d", targetID, jobs, wantJobs)
	}
}
