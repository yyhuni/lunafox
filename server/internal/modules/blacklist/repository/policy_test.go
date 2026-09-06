package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	blacklistdomain "github.com/yyhuni/lunafox/server/internal/modules/blacklist/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPolicyRepositoryReadsEmptySingletonPolicies(t *testing.T) {
	db := openPolicyRepositoryTestDB(t)
	targetID := 7
	seedPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{})
	seedPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{})
	repository := NewPolicyRepository(db)

	global, err := repository.GetGlobal(context.Background())
	if err != nil || global == nil || len(global.Patterns) != 0 {
		t.Fatalf("GetGlobal() = %#v, %v", global, err)
	}
	local, err := repository.GetTarget(context.Background(), targetID)
	if err != nil || local == nil || local.TargetID == nil || *local.TargetID != targetID || len(local.Patterns) != 0 {
		t.Fatalf("GetTarget() = %#v, %v", local, err)
	}
	patterns, err := repository.ReadEffectivePatternsForScan(context.Background(), targetID)
	if err != nil || len(patterns) != 0 {
		t.Fatalf("ReadEffectivePatternsForScan() = %v, %v", patterns, err)
	}
}

func TestPolicyRepositoryRejectsMissingOrCorruptRows(t *testing.T) {
	db := openPolicyRepositoryTestDB(t)
	repository := NewPolicyRepository(db)
	if _, err := repository.GetGlobal(context.Background()); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("GetGlobal missing error = %v", err)
	}
	if err := db.Create(&model.Policy{Scope: string(blacklistdomain.ScopeGlobal), Patterns: modelJSON([]byte(`["Example.com"]`)), UpdatedAt: time.Now().UTC()}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetGlobal(context.Background()); !errors.Is(err, ErrPolicyDataIntegrity) {
		t.Fatalf("GetGlobal corrupt error = %v", err)
	}
}

func TestPolicyRepositoryReplaceUsesCurrentETagAndSkipsCanonicalNoop(t *testing.T) {
	db := openPolicyRepositoryTestDB(t)
	seedPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"example.com"})
	repository := NewPolicyRepository(db)
	current, err := repository.GetGlobal(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	etag, err := blacklistdomain.ETag(current.Patterns)
	if err != nil {
		t.Fatal(err)
	}
	unchangedAt := current.UpdatedAt
	returned, changed, err := repository.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, []string{"example.com"})
	if err != nil || changed || !returned.UpdatedAt.Equal(unchangedAt) {
		t.Fatalf("canonical no-op = %#v changed=%t err=%v", returned, changed, err)
	}
	if _, changed, err := repository.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, "stale", []string{"example.com"}); !errors.Is(err, ErrPolicyETagConflict) || changed {
		t.Fatalf("stale replace = changed=%t err=%v", changed, err)
	}
	updated, changed, err := repository.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, []string{"*.example.com", "192.0.2.4"})
	if err != nil || !changed || len(updated.Patterns) != 2 || !updated.UpdatedAt.After(unchangedAt) {
		t.Fatalf("replace = %#v changed=%t err=%v", updated, changed, err)
	}
	if _, _, err := repository.ReplacePatterns(context.Background(), blacklistdomain.ScopeGlobal, nil, etag, []string{"not a hostname"}); !errors.Is(err, blacklistdomain.ErrInvalidPattern) {
		t.Fatalf("invalid replace error = %v", err)
	}
}

func TestPolicyRepositoryEffectiveUnionRequiresBothCanonicalRows(t *testing.T) {
	db := openPolicyRepositoryTestDB(t)
	targetID := 9
	seedPolicy(t, db, blacklistdomain.ScopeGlobal, nil, []string{"*.example.com", "192.0.2.0/24"})
	seedPolicy(t, db, blacklistdomain.ScopeTarget, &targetID, []string{"example.com", "192.0.2.0/24"})
	repository := NewPolicyRepository(db)
	patterns, err := repository.ReadEffectivePatternsForScan(context.Background(), targetID)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"*.example.com", "192.0.2.0/24", "example.com"}
	if len(patterns) != len(want) {
		t.Fatalf("patterns = %v, want %v", patterns, want)
	}
	for index := range want {
		if patterns[index] != want[index] {
			t.Fatalf("patterns = %v, want %v", patterns, want)
		}
	}
	if _, err := repository.ReadEffectivePatternsForScan(context.Background(), 10); !errors.Is(err, ErrPolicyNotFound) {
		t.Fatalf("missing local policy error = %v", err)
	}
}

func openPolicyRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Policy{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedPolicy(t *testing.T, db *gorm.DB, scope blacklistdomain.Scope, targetID *int, patterns []string) {
	t.Helper()
	canonical, err := blacklistdomain.CanonicalizePolicyPatterns(patterns)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := blacklistdomain.CanonicalPatternsJSON(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Policy{Scope: string(scope), TargetID: targetID, Patterns: modelJSON(payload), UpdatedAt: time.Now().UTC().Add(-time.Second)}).Error; err != nil {
		t.Fatal(err)
	}
}
