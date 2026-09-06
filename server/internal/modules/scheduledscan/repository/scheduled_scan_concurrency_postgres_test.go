package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const scheduledScanConcurrencySchema = "scheduled_scan_concurrency"

type scheduledScanAttemptResult struct {
	input *scheduledapp.FrozenDispatchInput
	err   error
}

type scheduledScanUpdateResult struct {
	item *scheduledapp.ScheduledScan
	err  error
}

func TestScheduledScanLifecycleConcurrencyPostgres(t *testing.T) {
	db := openScheduledScanConcurrencyPostgres(t)
	repo := NewScheduledScanRepository(db).WithClock(func() time.Time { return scheduledScanPostgresNow() })

	t.Run("lateral candidate query rotates schedules and keeps oldest occurrence", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		first, firstCandidate := seedScheduledScanPostgresFixture(t, repo, db, "first", 1)
		second, _ := seedScheduledScanPostgresFixture(t, repo, db, "second", 1)
		older := &scheduledScanOccurrenceModel{
			ScheduledScanID: first.ID,
			ScheduledFor:    firstCandidate.ScheduledFor.Add(-time.Minute),
		}
		if err := db.Create(older).Error; err != nil {
			t.Fatalf("seed older occurrence: %v", err)
		}

		candidate, err := repo.SelectAttemptCandidate(context.Background())
		if err != nil || candidate == nil || candidate.ID != older.ID {
			t.Fatalf("SelectAttemptCandidate() = %+v, %v; want oldest occurrence %d", candidate, err, older.ID)
		}
		if _, err := repo.StartAttempt(context.Background(), *candidate, scheduledScanPostgresNow()); err != nil {
			t.Fatalf("StartAttempt(first): %v", err)
		}
		candidate, err = repo.SelectAttemptCandidate(context.Background())
		if err != nil || candidate == nil || candidate.ScheduledScanID != second.ID {
			t.Fatalf("rotated candidate = %+v, %v; want Schedule %d", candidate, err, second.ID)
		}
	})

	t.Run("concurrent attempt start increments aggregates exactly once", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		schedule, candidate := seedScheduledScanPostgresFixture(t, repo, db, "exactly-once", 1)
		start := make(chan struct{})
		results := make(chan scheduledScanAttemptResult, 2)
		for range 2 {
			go func() {
				<-start
				input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
				results <- scheduledScanAttemptResult{input: input, err: err}
			}()
		}
		close(start)
		nonNil := 0
		for range 2 {
			result := <-results
			if result.err != nil {
				t.Fatalf("concurrent StartAttempt() error = %v", result.err)
			}
			if result.input != nil {
				nonNil++
			}
		}
		if nonNil != 1 {
			t.Fatalf("concurrent StartAttempt() accepted %d attempts, want 1", nonNil)
		}
		assertScheduledScanPostgresState(t, db, schedule.ID, true, 1, 2)
	})

	t.Run("update first freezes one latest organization input version", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		_, candidate := seedScheduledScanPostgresFixture(t, repo, db, "update-first", 1)
		installScheduledScanSleepTrigger(t, db, "scheduled_scan", "UPDATE OF configuration")
		updates := make(chan scheduledScanUpdateResult, 1)
		go func() {
			item, err := repo.Update(context.Background(), candidate.ScheduledScanID, &scheduledapp.ScheduledScanUpdate{
				Configuration: map[string]any{"version": 2},
			})
			updates <- scheduledScanUpdateResult{item: item, err: err}
		}()
		waitForScheduledScanPostgresQuery(t, db, `UPDATE "scheduled_scan"`)
		attempts := make(chan scheduledScanAttemptResult, 1)
		go func() {
			input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
			attempts <- scheduledScanAttemptResult{input: input, err: err}
		}()
		assertScheduledScanPostgresOperationBlocked(t, attempts)
		updated := <-updates
		if updated.err != nil || updated.item == nil {
			t.Fatalf("Update() = %+v, %v", updated.item, updated.err)
		}
		attempted := <-attempts
		if attempted.err != nil || attempted.input == nil || attempted.input.Configuration["version"] != float64(2) || attempted.input.TargetScoped || len(attempted.input.TargetIDs) != 1 || attempted.input.TargetIDs[0] != 7 {
			t.Fatalf("attempt after Update = %+v, %v", attempted.input, attempted.err)
		}
	})

	t.Run("attempt first keeps frozen organization input while later update commits", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		_, candidate := seedScheduledScanPostgresFixture(t, repo, db, "attempt-first-update", 1)
		installScheduledScanSleepTrigger(t, db, "scheduled_scan_occurrence", "UPDATE OF attempted_at")
		attempts := make(chan scheduledScanAttemptResult, 1)
		go func() {
			input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
			attempts <- scheduledScanAttemptResult{input: input, err: err}
		}()
		waitForScheduledScanPostgresQuery(t, db, `UPDATE "scheduled_scan_occurrence"`)
		updates := make(chan scheduledScanUpdateResult, 1)
		go func() {
			item, err := repo.Update(context.Background(), candidate.ScheduledScanID, &scheduledapp.ScheduledScanUpdate{
				Configuration: map[string]any{"version": 2},
			})
			updates <- scheduledScanUpdateResult{item: item, err: err}
		}()
		assertScheduledScanPostgresOperationBlocked(t, updates)
		attempted := <-attempts
		if attempted.err != nil || attempted.input == nil || attempted.input.Configuration["version"] != float64(1) {
			t.Fatalf("attempt before Update = %+v, %v", attempted.input, attempted.err)
		}
		updated := <-updates
		if updated.err != nil || updated.item == nil || updated.item.Configuration["version"] != float64(2) {
			t.Fatalf("later Update = %+v, %v", updated.item, updated.err)
		}
	})

	t.Run("disable first deletes only unattempted occurrences", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		schedule, candidate := seedScheduledScanPostgresFixture(t, repo, db, "disable-first", 1)
		installScheduledScanSleepTrigger(t, db, "scheduled_scan", "UPDATE OF is_enabled")
		disabled := false
		updates := make(chan scheduledScanUpdateResult, 1)
		go func() {
			item, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{IsEnabled: &disabled})
			updates <- scheduledScanUpdateResult{item: item, err: err}
		}()
		waitForScheduledScanPostgresQuery(t, db, `UPDATE "scheduled_scan"`)
		attempts := make(chan scheduledScanAttemptResult, 1)
		go func() {
			input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
			attempts <- scheduledScanAttemptResult{input: input, err: err}
		}()
		assertScheduledScanPostgresOperationBlocked(t, attempts)
		updated := <-updates
		if updated.err != nil || updated.item == nil || updated.item.IsEnabled || updated.item.NextRunTime != nil {
			t.Fatalf("disable Update = %+v, %v", updated.item, updated.err)
		}
		attempted := <-attempts
		if attempted.err != nil || attempted.input != nil {
			t.Fatalf("attempt after disable = %+v, %v", attempted.input, attempted.err)
		}
		assertScheduledScanPostgresState(t, db, schedule.ID, false, 0, 1)
		assertOrdinaryScanSurvives(t, db)
	})

	t.Run("attempt first survives disable without deleting ordinary scans", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		schedule, candidate := seedScheduledScanPostgresFixture(t, repo, db, "attempt-first-disable", 1)
		installScheduledScanSleepTrigger(t, db, "scheduled_scan_occurrence", "UPDATE OF attempted_at")
		attempts := make(chan scheduledScanAttemptResult, 1)
		go func() {
			input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
			attempts <- scheduledScanAttemptResult{input: input, err: err}
		}()
		waitForScheduledScanPostgresQuery(t, db, `UPDATE "scheduled_scan_occurrence"`)
		disabled := false
		updates := make(chan scheduledScanUpdateResult, 1)
		go func() {
			item, err := repo.Update(context.Background(), schedule.ID, &scheduledapp.ScheduledScanUpdate{IsEnabled: &disabled})
			updates <- scheduledScanUpdateResult{item: item, err: err}
		}()
		assertScheduledScanPostgresOperationBlocked(t, updates)
		attempted := <-attempts
		if attempted.err != nil || attempted.input == nil {
			t.Fatalf("attempt before disable = %+v, %v", attempted.input, attempted.err)
		}
		updated := <-updates
		if updated.err != nil || updated.item == nil || updated.item.IsEnabled {
			t.Fatalf("disable after attempt = %+v, %v", updated.item, updated.err)
		}
		assertScheduledScanPostgresState(t, db, schedule.ID, false, 1, 2)
		assertOrdinaryScanSurvives(t, db)
	})

	t.Run("delete first prevents attempt and preserves ordinary scans", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		schedule, candidate := seedScheduledScanPostgresFixture(t, repo, db, "delete-first", 1)
		installScheduledScanSleepTrigger(t, db, "scheduled_scan", "DELETE")
		deletes := make(chan error, 1)
		go func() { deletes <- repo.Delete(context.Background(), schedule.ID) }()
		waitForScheduledScanPostgresQuery(t, db, `DELETE FROM "scheduled_scan"`)
		attempts := make(chan scheduledScanAttemptResult, 1)
		go func() {
			input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
			attempts <- scheduledScanAttemptResult{input: input, err: err}
		}()
		assertScheduledScanPostgresOperationBlocked(t, attempts)
		if err := <-deletes; err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		attempted := <-attempts
		if attempted.err != nil || attempted.input != nil {
			t.Fatalf("attempt after Delete = %+v, %v", attempted.input, attempted.err)
		}
		assertScheduleAndOccurrencesAbsent(t, db, schedule.ID)
		assertOrdinaryScanSurvives(t, db)
	})

	t.Run("attempt first continues after delete and outcome write is expected no-op", func(t *testing.T) {
		resetScheduledScanConcurrencyFixture(t, db)
		schedule, candidate := seedScheduledScanPostgresFixture(t, repo, db, "attempt-first-delete", 1)
		installScheduledScanSleepTrigger(t, db, "scheduled_scan_occurrence", "UPDATE OF attempted_at")
		attempts := make(chan scheduledScanAttemptResult, 1)
		go func() {
			input, err := repo.StartAttempt(context.Background(), candidate, scheduledScanPostgresNow())
			attempts <- scheduledScanAttemptResult{input: input, err: err}
		}()
		waitForScheduledScanPostgresQuery(t, db, `UPDATE "scheduled_scan_occurrence"`)
		deletes := make(chan error, 1)
		go func() { deletes <- repo.Delete(context.Background(), schedule.ID) }()
		assertScheduledScanPostgresOperationBlocked(t, deletes)
		attempted := <-attempts
		if attempted.err != nil || attempted.input == nil {
			t.Fatalf("attempt before Delete = %+v, %v", attempted.input, attempted.err)
		}
		if err := <-deletes; err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		recorded, err := repo.RecordOutcome(context.Background(), attempted.input.OccurrenceID, scheduledapp.HandoffOutcome{Kind: scheduledapp.HandoffCompleted}, scheduledScanPostgresNow())
		if err != nil || recorded {
			t.Fatalf("RecordOutcome() after Delete = %t, %v; want expected no-op", recorded, err)
		}
		assertScheduleAndOccurrencesAbsent(t, db, schedule.ID)
		assertOrdinaryScanSurvives(t, db)
	})
}

func openScheduledScanConcurrencyPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_SCHEDULED_SCAN_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_SCHEDULED_SCAN_DSN to run the opt-in Scheduled Scan PostgreSQL concurrency verification")
	}
	if !strings.Contains(dsn, "search_path="+scheduledScanConcurrencySchema) {
		t.Fatalf("Scheduled Scan PostgreSQL DSN must pin search_path=%s", scheduledScanConcurrencySchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open Scheduled Scan PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open Scheduled Scan PostgreSQL SQL handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + scheduledScanConcurrencySchema + " CASCADE").Error; err != nil {
		t.Fatalf("drop Scheduled Scan concurrency schema: %v", err)
	}
	if err := db.Exec("CREATE SCHEMA " + scheduledScanConcurrencySchema).Error; err != nil {
		t.Fatalf("create Scheduled Scan concurrency schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = db.WithContext(cleanupCtx).Exec("DROP SCHEMA IF EXISTS " + scheduledScanConcurrencySchema + " CASCADE").Error
	})
	statements := []string{
		`CREATE TABLE organization (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`,
		`CREATE TABLE target (id INTEGER PRIMARY KEY, name TEXT NOT NULL, deleted_at TIMESTAMPTZ)`,
		`CREATE TABLE organization_target (
			organization_id INTEGER NOT NULL,
			target_id INTEGER NOT NULL,
			PRIMARY KEY (organization_id, target_id)
		)`,
		`CREATE TABLE scan (
			id INTEGER PRIMARY KEY,
			marker TEXT NOT NULL,
			input_source VARCHAR(32) NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory'))
		)`,
		`CREATE TABLE scheduled_scan (
			id SERIAL PRIMARY KEY, name VARCHAR(200) NOT NULL, scan_workflow_id VARCHAR(100) NOT NULL,
			configuration JSONB NOT NULL, input_source VARCHAR(32) NOT NULL CHECK (input_source IN ('scan_snapshot', 'target_inventory')),
			organization_id INTEGER, target_id INTEGER, agent_id INTEGER,
			cron_expression VARCHAR(100) NOT NULL,
			is_enabled BOOLEAN NOT NULL, run_count INTEGER NOT NULL DEFAULT 0,
			successful_handoff_count INTEGER NOT NULL DEFAULT 0,
			failed_handoff_count INTEGER NOT NULL DEFAULT 0,
			last_run_time TIMESTAMPTZ, next_run_time TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT scheduled_scan_enabled_cursor_consistency CHECK (
				(is_enabled = TRUE AND next_run_time IS NOT NULL) OR
				(is_enabled = FALSE AND next_run_time IS NULL)
			)
		)`,
		`CREATE TABLE scheduled_scan_occurrence (
			id BIGSERIAL PRIMARY KEY,
			scheduled_scan_id INTEGER NOT NULL REFERENCES scheduled_scan(id) ON DELETE CASCADE,
			scheduled_for TIMESTAMPTZ NOT NULL,
			attempted_at TIMESTAMPTZ, dispatched_at TIMESTAMPTZ,
			failure_kind VARCHAR(100), failure_message VARCHAR(2000),
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (scheduled_scan_id, scheduled_for)
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare Scheduled Scan PostgreSQL schema: %v", err)
		}
	}
	return db
}

func resetScheduledScanConcurrencyFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DROP FUNCTION IF EXISTS scheduled_scan_test_sleep() CASCADE").Error; err != nil {
		t.Fatalf("drop prior sleep trigger: %v", err)
	}
	if err := db.Exec("TRUNCATE scheduled_scan_occurrence, scheduled_scan, scan, organization_target, organization, target RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("reset Scheduled Scan concurrency fixture: %v", err)
	}
	if err := db.Exec("INSERT INTO organization (id, name) VALUES (5, 'Acme')").Error; err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := db.Exec("INSERT INTO target (id, name) VALUES (7, 'example.com')").Error; err != nil {
		t.Fatalf("seed target: %v", err)
	}
	if err := db.Exec("INSERT INTO organization_target (organization_id, target_id) VALUES (5, 7)").Error; err != nil {
		t.Fatalf("seed organization membership: %v", err)
	}
	if err := db.Exec("INSERT INTO scan (id, marker, input_source) VALUES (91, 'ordinary', 'scan_snapshot')").Error; err != nil {
		t.Fatalf("seed ordinary Scan: %v", err)
	}
}

func seedScheduledScanPostgresFixture(
	t *testing.T,
	repo *ScheduledScanRepository,
	db *gorm.DB,
	name string,
	version int,
) (*scheduledapp.ScheduledScan, scheduledapp.OccurrenceCandidate) {
	t.Helper()
	organizationID := 5
	schedule, err := repo.Create(context.Background(), &scheduledapp.ScheduledScanCreate{
		Name: name, ScanWorkflowID: "default", Configuration: map[string]any{"version": version},
		InputSource:    scandomain.InputSourceScanSnapshot,
		OrganizationID: &organizationID, AgentID: intPointer(42),
		CronExpression: "* * * * *", IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("create Schedule fixture: %v", err)
	}
	historicalAttemptedAt := scheduledScanPostgresNow().Add(-2 * time.Hour)
	historical := &scheduledScanOccurrenceModel{
		ScheduledScanID: schedule.ID,
		ScheduledFor:    historicalAttemptedAt.Add(-time.Minute),
		AttemptedAt:     &historicalAttemptedAt,
	}
	if err := db.Create(historical).Error; err != nil {
		t.Fatalf("seed historical attempted occurrence: %v", err)
	}
	pending := &scheduledScanOccurrenceModel{
		ScheduledScanID: schedule.ID,
		ScheduledFor:    scheduledScanPostgresNow().Add(-time.Hour),
	}
	if err := db.Create(pending).Error; err != nil {
		t.Fatalf("seed pending occurrence: %v", err)
	}
	return schedule, scheduledapp.OccurrenceCandidate{
		ID: pending.ID, ScheduledScanID: schedule.ID, ScheduledFor: pending.ScheduledFor,
	}
}

func installScheduledScanSleepTrigger(t *testing.T, db *gorm.DB, table, event string) {
	t.Helper()
	function := `CREATE OR REPLACE FUNCTION scheduled_scan_test_sleep() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_sleep(0.5);
			IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
			RETURN NEW;
		END
	$$`
	if err := db.Exec(function).Error; err != nil {
		t.Fatalf("create sleep trigger function: %v", err)
	}
	statement := fmt.Sprintf(
		"CREATE TRIGGER scheduled_scan_test_sleep_trigger BEFORE %s ON %s FOR EACH ROW EXECUTE FUNCTION scheduled_scan_test_sleep()",
		event,
		table,
	)
	if err := db.Exec(statement).Error; err != nil {
		t.Fatalf("create sleep trigger: %v", err)
	}
}

func waitForScheduledScanPostgresQuery(t *testing.T, db *gorm.DB, fragment string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		err := db.Raw(`
			SELECT COUNT(*)
			FROM pg_stat_activity
			WHERE pid <> pg_backend_pid()
				AND state = 'active'
				AND query LIKE ?`, "%"+fragment+"%").Scan(&count).Error
		if err != nil {
			t.Fatalf("inspect PostgreSQL activity: %v", err)
		}
		if count > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("PostgreSQL query containing %q did not enter the sleep trigger", fragment)
}

func assertScheduledScanPostgresOperationBlocked[T any](t *testing.T, result <-chan T) {
	t.Helper()
	select {
	case <-result:
		t.Fatal("operation crossed an uncommitted Schedule lifecycle transaction")
	case <-time.After(100 * time.Millisecond):
	}
}

func assertScheduledScanPostgresState(t *testing.T, db *gorm.DB, scheduleID int, enabled bool, runCount, occurrenceCount int64) {
	t.Helper()
	var schedule struct {
		IsEnabled   bool       `gorm:"column:is_enabled"`
		RunCount    int64      `gorm:"column:run_count"`
		LastRunTime *time.Time `gorm:"column:last_run_time"`
		NextRunTime *time.Time `gorm:"column:next_run_time"`
	}
	if err := db.Table("scheduled_scan").Where("id = ?", scheduleID).Take(&schedule).Error; err != nil {
		t.Fatalf("read Schedule state: %v", err)
	}
	if schedule.IsEnabled != enabled || schedule.RunCount != runCount {
		t.Fatalf("Schedule state = %+v, want enabled=%t runCount=%d", schedule, enabled, runCount)
	}
	if enabled && schedule.NextRunTime == nil || !enabled && schedule.NextRunTime != nil {
		t.Fatalf("Schedule cursor invariant failed: %+v", schedule)
	}
	if runCount > 0 && schedule.LastRunTime == nil {
		t.Fatalf("Schedule attempt aggregate missing lastRunTime: %+v", schedule)
	}
	var count int64
	if err := db.Model(&scheduledScanOccurrenceModel{}).Where("scheduled_scan_id = ?", scheduleID).Count(&count).Error; err != nil {
		t.Fatalf("count occurrence rows: %v", err)
	}
	if count != occurrenceCount {
		t.Fatalf("occurrence count = %d, want %d", count, occurrenceCount)
	}
}

func assertScheduleAndOccurrencesAbsent(t *testing.T, db *gorm.DB, scheduleID int) {
	t.Helper()
	var scheduleCount, occurrenceCount int64
	if err := db.Table("scheduled_scan").Where("id = ?", scheduleID).Count(&scheduleCount).Error; err != nil {
		t.Fatalf("count Schedules: %v", err)
	}
	if err := db.Table("scheduled_scan_occurrence").Where("scheduled_scan_id = ?", scheduleID).Count(&occurrenceCount).Error; err != nil {
		t.Fatalf("count occurrences: %v", err)
	}
	if scheduleCount != 0 || occurrenceCount != 0 {
		t.Fatalf("deleted scheduling data = Schedules %d occurrences %d", scheduleCount, occurrenceCount)
	}
}

func assertOrdinaryScanSurvives(t *testing.T, db *gorm.DB) {
	t.Helper()
	var count int64
	if err := db.Table("scan").Where("id = 91 AND marker = 'ordinary'").Count(&count).Error; err != nil {
		t.Fatalf("read ordinary Scan: %v", err)
	}
	if count != 1 {
		t.Fatalf("Schedule lifecycle changed ordinary Scan count to %d", count)
	}
}

func scheduledScanPostgresNow() time.Time {
	return time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
}

func intPointer(value int) *int { return &value }
