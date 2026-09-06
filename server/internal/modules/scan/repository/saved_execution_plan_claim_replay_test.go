package repository

import (
	"bytes"
	"context"
	"errors"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestFindSavedPlanClaimReplayTreatsOnlyExpectedEmptyLookupAsSilent(t *testing.T) {
	var output bytes.Buffer
	databaseLogger := logger.New(log.New(&output, "", 0), logger.Config{
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: false,
		ParameterizedQueries:      true,
	})
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "saved-plan-replay.db")), &gorm.Config{Logger: databaseLogger})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Exec(`CREATE TABLE scan_task (
		id INTEGER PRIMARY KEY,
		scan_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		assigned_agent_id INTEGER,
		assigned_session_id TEXT,
		assigned_session_epoch INTEGER,
		assigned_request_id TEXT,
		resolved_execution_plan BLOB NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create scan_task: %v", err)
	}

	output.Reset()
	replay, err := findSavedPlanClaimReplay(db.WithContext(context.Background()), 7, "session-7", 9, "request-7")
	if err != nil || replay != nil {
		t.Fatalf("empty replay lookup = %#v, %v; want nil, nil", replay, err)
	}
	if output.Len() != 0 {
		t.Fatalf("empty replay lookup emitted a database warning/error: %q", output.String())
	}

	output.Reset()
	if err := db.Table("scan_task").Where("id = ?", 99).Take(&struct{ ID int }{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("ordinary record-not-found query = %v, want ErrRecordNotFound", err)
	}
	if !strings.Contains(output.String(), "record not found") {
		t.Fatalf("global RecordNotFound visibility was weakened: %q", output.String())
	}

	output.Reset()
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = findSavedPlanClaimReplay(db.WithContext(cancelledContext), 7, "session-7", 9, "request-7")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled caller context error = %v, want context.Canceled", err)
	}
	if output.Len() == 0 {
		t.Fatal("cancelled database lookup error was not logged")
	}
}
