package agentdata

import (
	"context"
	"errors"
	"testing"

	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type executionInputBlacklistSnapshotStoreStub struct {
	patterns []string
	err      error
	calls    int
	scanID   int
}

func (stub *executionInputBlacklistSnapshotStoreStub) LoadBlacklistSnapshot(_ context.Context, scanID int) ([]string, error) {
	stub.calls++
	stub.scanID = scanID
	if stub.err != nil {
		return nil, stub.err
	}
	return append([]string{}, stub.patterns...), nil
}

func TestExecutionInputBlacklistSnapshotSourceBuildsOneImmutableFilter(t *testing.T) {
	store := &executionInputBlacklistSnapshotStoreStub{patterns: []string{"*.blocked.example", "192.0.2.0/24"}}
	source := NewExecutionInputBlacklistSnapshotSource(store)
	if source == nil {
		t.Fatal("expected snapshot source")
	}
	filter, err := source.ResolveExecutionInputBlacklistFilter(context.Background(), 42)
	if err != nil {
		t.Fatalf("ResolveExecutionInputBlacklistFilter() error = %v", err)
	}
	if store.calls != 1 || store.scanID != 42 {
		t.Fatalf("snapshot load calls=%d scanID=%d, want one call for Scan 42", store.calls, store.scanID)
	}
	if excluded, err := filter.ShouldExcludeHostname("deep.blocked.example"); err != nil || !excluded {
		t.Fatalf("wildcard filter result = excluded:%t err:%v", excluded, err)
	}
	if excluded, err := filter.ShouldExcludeHostname("192.0.2.17"); err != nil || !excluded {
		t.Fatalf("CIDR filter result = excluded:%t err:%v", excluded, err)
	}
	store.patterns[0] = "198.51.100.0/24"
	if excluded, err := filter.ShouldExcludeHostname("192.0.2.17"); err != nil || !excluded {
		t.Fatalf("compiled filter changed with store mutation: excluded:%t err:%v", excluded, err)
	}
}

func TestExecutionInputBlacklistSnapshotSourceFailsClosed(t *testing.T) {
	tests := []struct {
		name  string
		store *executionInputBlacklistSnapshotStoreStub
		want  error
	}{
		{name: "missing snapshot", store: &executionInputBlacklistSnapshotStoreStub{err: scanapp.ErrScanBlacklistSnapshotNotFound}, want: ErrExecutionArtifactDataLoss},
		{name: "corrupt snapshot", store: &executionInputBlacklistSnapshotStoreStub{err: scanapp.ErrScanBlacklistSnapshotDataIntegrity}, want: ErrExecutionArtifactDataLoss},
		{name: "noncanonical store output", store: &executionInputBlacklistSnapshotStoreStub{patterns: []string{"Example.COM"}}, want: ErrExecutionArtifactDataLoss},
		{name: "store unavailable", store: &executionInputBlacklistSnapshotStoreStub{err: errors.New("database unavailable")}, want: ErrExecutionArtifactUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := NewExecutionInputBlacklistSnapshotSource(test.store)
			if _, err := source.ResolveExecutionInputBlacklistFilter(context.Background(), 7); !errors.Is(err, test.want) {
				t.Fatalf("source error = %v, want %v", err, test.want)
			}
		})
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	source := NewExecutionInputBlacklistSnapshotSource(&executionInputBlacklistSnapshotStoreStub{})
	if _, err := source.ResolveExecutionInputBlacklistFilter(cancelled, 7); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled source error = %v, want context cancellation", err)
	}
	if source := NewExecutionInputBlacklistSnapshotSource(nil); source != nil {
		t.Fatalf("nil store source = %#v, want nil", source)
	}
}
