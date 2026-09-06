package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	model "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository/persistence"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const nucleiPOCActivationPostgresSchema = "nuclei_poc_activation_contract"

// TestNucleiPOCRepositoryPostgresAdvisoryLockSerializesAcrossConnections is
// opt-in because the normal unit suite intentionally runs without a database
// service. The holder transaction proves that a separate repository instance
// cannot enter the catalog mutation callback until the transaction-scoped
// advisory lock is released.
func TestNucleiPOCRepositoryPostgresAdvisoryLockSerializesAcrossConnections(t *testing.T) {
	db := openNucleiPOCActivationPostgresDB(t)
	now := time.Now().UTC()
	sourceID := uuid.New()
	if err := db.Create(&model.Source{ID: sourceID, SourceType: "git", RepoURL: "https://example.com/templates.git", IsActive: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(uuid.New(), sourceID, "postgres-lock", "PostgreSQL lock")
	if err := db.Select("*").Create(&model.POC{ID: uuid.New(), SourceID: sourceID, TemplateID: candidate.TemplateID, DisplayName: candidate.DisplayName, Severity: candidate.Severity, Tags: []byte(`[]`), CVE: []byte(`[]`), CWE: []byte(`[]`), References: []byte(`[]`), RelativePath: candidate.RelativePath, ContentSHA256: candidate.ContentSHA256, Content: candidate.Content, IsEnabled: false, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	holder := db.Begin()
	if holder.Error != nil {
		t.Fatal(holder.Error)
	}
	defer func() { _ = holder.Rollback().Error }()
	if err := holder.Exec("SELECT pg_advisory_xact_lock(?)", nucleiPOCCatalogAdvisoryLockKey).Error; err != nil {
		t.Fatal(err)
	}

	type result struct {
		changed int64
		err     error
	}
	completed := make(chan result, 1)
	go func() {
		changed, err := NewNucleiPOCRepository(db).SetPOCActivation(context.Background(), true, nil)
		completed <- result{changed: changed, err: err}
	}()

	select {
	case got := <-completed:
		t.Fatalf("catalog mutation returned while advisory lock was held: changed=%d err=%v", got.changed, got.err)
	case <-time.After(200 * time.Millisecond):
	}
	if err := holder.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-completed:
		if got.err != nil || got.changed != 1 {
			t.Fatalf("catalog mutation after advisory lock release changed=%d err=%v", got.changed, got.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("catalog mutation did not complete after advisory lock release")
	}
}

func openNucleiPOCActivationPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_NUCLEI_POC_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_NUCLEI_POC_POSTGRES_DSN with search_path=" + nucleiPOCActivationPostgresSchema + " to run PostgreSQL catalog-lock verification")
	}
	if !strings.Contains(dsn, "search_path="+nucleiPOCActivationPostgresSchema) {
		t.Fatalf("Nuclei POC PostgreSQL DSN must pin search_path=%s", nucleiPOCActivationPostgresSchema)
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
	if err := db.Exec("DROP SCHEMA IF EXISTS " + nucleiPOCActivationPostgresSchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE SCHEMA " + nucleiPOCActivationPostgresSchema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA IF EXISTS " + nucleiPOCActivationPostgresSchema + " CASCADE").Error })
	if err := db.AutoMigrate(&model.Source{}, &model.SyncTask{}, &model.CandidateImport{}, &model.POC{}, &model.RequestTombstone{}); err != nil {
		t.Fatal(err)
	}
	return db
}
