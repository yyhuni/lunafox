package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const blacklistPolicyPostgresSchema = "blacklist_policy_contract"

// TestPolicyRepositoryPostgresCompareAndReplaceRace exercises the production
// row-lock path; SQLite cannot prove PostgreSQL FOR UPDATE serialization.
func TestPolicyRepositoryPostgresCompareAndReplaceRace(t *testing.T) {
	db := openBlacklistPolicyPostgresDB(t)
	seedPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"before.example"})
	repository := NewPolicyRepository(db)
	current, err := repository.GetGlobal(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	etag, err := blacklistdomain.ETag(current.Patterns)
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, patterns := range [][]string{{"first.example"}, {"second.example"}} {
		patterns := patterns
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, _, err := repository.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, patterns)
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPolicyETagConflict):
			conflicts++
		default:
			t.Fatalf("concurrent ReplacePatterns() error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent ReplacePatterns() successes=%d conflicts=%d, want 1/1", successes, conflicts)
	}
}

func openBlacklistPolicyPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LUNAFOX_POSTGRES_BLACKLIST_DSN"))
	if dsn == "" {
		t.Skip("set LUNAFOX_POSTGRES_BLACKLIST_DSN with search_path=" + blacklistPolicyPostgresSchema + " to run PostgreSQL blacklist repository verification")
	}
	if !strings.Contains(dsn, "search_path="+blacklistPolicyPostgresSchema) {
		t.Fatalf("blacklist PostgreSQL DSN must pin search_path=%s", blacklistPolicyPostgresSchema)
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
	if err := db.Exec("DROP SCHEMA IF EXISTS " + blacklistPolicyPostgresSchema + " CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE SCHEMA " + blacklistPolicyPostgresSchema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP SCHEMA IF EXISTS " + blacklistPolicyPostgresSchema + " CASCADE").Error })
	if err := db.Exec(`
		CREATE TABLE blacklist_policy (
			id SERIAL PRIMARY KEY,
			scope TEXT NOT NULL,
			target_id INTEGER,
			patterns JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}
