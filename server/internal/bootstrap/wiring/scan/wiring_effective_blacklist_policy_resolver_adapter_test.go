package scanwiring

import (
	"context"
	"errors"
	"slices"
	"testing"

	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type blacklistPolicyResolverStub struct {
	targetID int
	patterns []string
	err      error
}

func (stub *blacklistPolicyResolverStub) GetGlobal(context.Context) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (stub *blacklistPolicyResolverStub) GetTarget(context.Context, int) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (stub *blacklistPolicyResolverStub) ReplaceGlobal(context.Context, blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (stub *blacklistPolicyResolverStub) ReplaceTarget(context.Context, int, blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (stub *blacklistPolicyResolverStub) ResolveEffectivePatternsForScan(_ context.Context, targetID int) ([]string, error) {
	stub.targetID = targetID
	return append([]string{}, stub.patterns...), stub.err
}

func TestEffectiveBlacklistPolicyResolverAdapterDelegatesToBlacklistApplication(t *testing.T) {
	stub := &blacklistPolicyResolverStub{patterns: []string{"*.example.com"}}
	adapter, err := NewEffectiveBlacklistPolicyResolverAdapter(stub)
	if err != nil {
		t.Fatalf("NewEffectiveBlacklistPolicyResolverAdapter failed: %v", err)
	}
	patterns, err := adapter.ResolveEffectivePatternsForScan(context.Background(), 42)
	if err != nil {
		t.Fatalf("ResolveEffectivePatternsForScan failed: %v", err)
	}
	if stub.targetID != 42 || !slices.Equal(patterns, []string{"*.example.com"}) {
		t.Fatalf("resolver delegation = target=%d patterns=%#v", stub.targetID, patterns)
	}
	patterns[0] = "mutated"
	if stub.patterns[0] != "*.example.com" {
		t.Fatalf("adapter returned an aliased pattern slice: %#v", stub.patterns)
	}
}

func TestEffectiveBlacklistPolicyResolverAdapterPreservesFailureAndRejectsNilDependency(t *testing.T) {
	if _, err := NewEffectiveBlacklistPolicyResolverAdapter(nil); err == nil {
		t.Fatal("expected nil policy service to fail")
	}
	want := errors.New("policy read failed")
	adapter, err := NewEffectiveBlacklistPolicyResolverAdapter(&blacklistPolicyResolverStub{err: want})
	if err != nil {
		t.Fatalf("NewEffectiveBlacklistPolicyResolverAdapter failed: %v", err)
	}
	if _, err := adapter.ResolveEffectivePatternsForScan(context.Background(), 1); !errors.Is(err, want) {
		t.Fatalf("resolver error = %v, want %v", err, want)
	}
}

func TestScanBlacklistSnapshotStoreAdapterMapsPrivateSnapshotFailures(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scan-blacklist-snapshot-store-adapter?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec("CREATE TABLE scan_blacklist_snapshot (scan_id INTEGER PRIMARY KEY, patterns TEXT NOT NULL)").Error; err != nil {
		t.Fatalf("create snapshot table: %v", err)
	}
	store := NewScanBlacklistSnapshotStoreAdapter(scanrepo.NewScanRepository(db))
	if store == nil {
		t.Fatal("expected snapshot store adapter")
	}
	if _, err := store.LoadBlacklistSnapshot(context.Background(), 1); !errors.Is(err, scanapp.ErrScanBlacklistSnapshotNotFound) {
		t.Fatalf("missing snapshot error = %v, want application not found", err)
	}
	if err := db.Exec(`INSERT INTO scan_blacklist_snapshot (scan_id, patterns) VALUES (2, '["example.com"]'), (3, 'null')`).Error; err != nil {
		t.Fatalf("insert snapshots: %v", err)
	}
	patterns, err := store.LoadBlacklistSnapshot(context.Background(), 2)
	if err != nil {
		t.Fatalf("load valid snapshot: %v", err)
	}
	patterns[0] = "mutated"
	fresh, err := store.LoadBlacklistSnapshot(context.Background(), 2)
	if err != nil || !slices.Equal(fresh, []string{"example.com"}) {
		t.Fatalf("snapshot store returned aliased patterns: patterns=%#v err=%v", fresh, err)
	}
	if _, err := store.LoadBlacklistSnapshot(context.Background(), 3); !errors.Is(err, scanapp.ErrScanBlacklistSnapshotDataIntegrity) {
		t.Fatalf("corrupt snapshot error = %v, want application data integrity", err)
	}
}
