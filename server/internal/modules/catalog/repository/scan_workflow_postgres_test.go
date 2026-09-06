package repository

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const scanWorkflowPostgresContractSchema = "scan_workflow_contract"

// TestScanWorkflowRepositoryPostgresContract exercises the PostgreSQL-only
// namespace, partial-index, JSONB, ordering, and CAS guarantees. It is opt-in
// because it creates and drops an isolated schema.
func TestScanWorkflowRepositoryPostgresContract(t *testing.T) {
	repository := newScanWorkflowPostgresRepositoryForTest(t)
	requestID := uuid.NewString()
	first := userWorkflowForRepositoryTest(t, "wf-123e4567-e89b-12d3-a456-426614174101", "Alpha", "first")
	first.RequestID = requestID
	if err := repository.CreateScanWorkflow(&first); err != nil {
		t.Fatal(err)
	}
	builtin := builtinWorkflowForRepositoryTest(t, "Built-in")
	if err := repository.CreateScanWorkflow(&builtin); err != nil {
		t.Fatal(err)
	}
	second := userWorkflowForRepositoryTest(t, "wf-123e4567-e89b-12d3-a456-426614174102", "Bravo", "second")
	if err := repository.CreateScanWorkflow(&second); err != nil {
		t.Fatal(err)
	}

	workflows, total, err := repository.ListScanWorkflows(catalogdomain.ScanWorkflowListFilter{Page: 1, PageSize: 2})
	if err != nil || total != 3 || len(workflows) != 2 || workflows[0].ScanWorkflowID != "default" || workflows[1].DisplayName != "Alpha" {
		t.Fatalf("stable list = %+v, total=%d, error=%v", workflows, total, err)
	}

	duplicate := userWorkflowForRepositoryTest(t, "wf-123e4567-e89b-12d3-a456-426614174103", "Duplicate", "")
	duplicate.RequestID = requestID
	if err := repository.CreateScanWorkflow(&duplicate); err == nil {
		t.Fatal("duplicate requestId must fail the PostgreSQL partial unique index")
	}

	start := make(chan struct{})
	errors := make(chan error, 2)
	var group sync.WaitGroup
	for _, displayName := range []string{"Concurrent one", "Concurrent two"} {
		group.Add(1)
		go func(displayName string) {
			defer group.Done()
			<-start
			candidate, err := repository.GetScanWorkflowByID(first.ScanWorkflowID)
			if err != nil {
				errors <- err
				return
			}
			candidate.DisplayName = displayName
			updated, err := repository.UpdateUserScanWorkflow(candidate, 1)
			if err != nil {
				errors <- err
				return
			}
			if updated {
				errors <- nil
				return
			}
			errors <- errScanWorkflowCASLost
		}(displayName)
	}
	close(start)
	group.Wait()
	close(errors)
	updates := 0
	for err := range errors {
		if err == nil {
			updates++
			continue
		}
		if err != errScanWorkflowCASLost {
			t.Fatalf("concurrent update error: %v", err)
		}
	}
	if updates != 1 {
		t.Fatalf("concurrent CAS updates = %d, want 1", updates)
	}
}

var errScanWorkflowCASLost = &scanWorkflowCASLostError{}

type scanWorkflowCASLostError struct{}

func (*scanWorkflowCASLostError) Error() string { return "scan workflow CAS lost" }

func newScanWorkflowPostgresRepositoryForTest(t *testing.T) *ScanWorkflowRepository {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_SCAN_WORKFLOW_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_SCAN_WORKFLOW_DSN with search_path=scan_workflow_contract to run PostgreSQL workflow repository verification")
	}
	if !strings.Contains(dsn, "search_path="+scanWorkflowPostgresContractSchema) {
		t.Fatalf("scan workflow PostgreSQL DSN must pin search_path=%s", scanWorkflowPostgresContractSchema)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("DROP SCHEMA IF EXISTS " + scanWorkflowPostgresContractSchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE SCHEMA " + scanWorkflowPostgresContractSchema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA IF EXISTS " + scanWorkflowPostgresContractSchema + " CASCADE").Error })
	for _, statement := range []string{
		`CREATE TABLE scan_workflow (
			scan_workflow_id TEXT PRIMARY KEY, display_name TEXT NOT NULL, description TEXT NOT NULL,
			stages JSONB NOT NULL, is_builtin BOOLEAN NOT NULL, definition_digest TEXT,
			request_id UUID, version BIGINT NOT NULL, create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX idx_scan_workflow_request_id ON scan_workflow(request_id) WHERE request_id IS NOT NULL`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	return NewScanWorkflowRepository(db)
}
