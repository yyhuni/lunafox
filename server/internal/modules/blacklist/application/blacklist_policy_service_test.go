package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type blacklistPolicyStoreStub struct {
	global             *BlacklistPolicyRecord
	targets            map[int]*BlacklistPolicyRecord
	effective          []string
	getGlobalErr       error
	getTargetErr       error
	replaceGlobalErr   error
	replaceTargetErr   error
	effectiveErr       error
	lastGlobalETag     string
	lastGlobalPatterns []string
	lastTargetID       int
	lastTargetETag     string
	lastTargetPatterns []string
}

func (stub *blacklistPolicyStoreStub) GetGlobal(context.Context) (*BlacklistPolicyRecord, error) {
	return cloneBlacklistPolicyRecord(stub.global), stub.getGlobalErr
}

func (stub *blacklistPolicyStoreStub) GetTarget(_ context.Context, targetID int) (*BlacklistPolicyRecord, error) {
	if stub.getTargetErr != nil {
		return nil, stub.getTargetErr
	}
	return cloneBlacklistPolicyRecord(stub.targets[targetID]), nil
}

func (stub *blacklistPolicyStoreStub) ReplaceGlobal(_ context.Context, etag string, patterns []string) (*BlacklistPolicyRecord, error) {
	stub.lastGlobalETag = etag
	stub.lastGlobalPatterns = cloneBlacklistPatterns(patterns)
	if stub.replaceGlobalErr != nil {
		return nil, stub.replaceGlobalErr
	}
	stub.global.Patterns = cloneBlacklistPatterns(patterns)
	return cloneBlacklistPolicyRecord(stub.global), nil
}

func (stub *blacklistPolicyStoreStub) ReplaceTarget(_ context.Context, targetID int, etag string, patterns []string) (*BlacklistPolicyRecord, error) {
	stub.lastTargetID = targetID
	stub.lastTargetETag = etag
	stub.lastTargetPatterns = cloneBlacklistPatterns(patterns)
	if stub.replaceTargetErr != nil {
		return nil, stub.replaceTargetErr
	}
	record := stub.targets[targetID]
	record.Patterns = cloneBlacklistPatterns(patterns)
	return cloneBlacklistPolicyRecord(record), nil
}

func (stub *blacklistPolicyStoreStub) ReadEffectivePatternsForScan(context.Context, int) ([]string, error) {
	return cloneBlacklistPatterns(stub.effective), stub.effectiveErr
}

func cloneBlacklistPolicyRecord(record *BlacklistPolicyRecord) *BlacklistPolicyRecord {
	if record == nil {
		return nil
	}
	copyRecord := *record
	copyRecord.Patterns = cloneBlacklistPatterns(record.Patterns)
	if record.TargetID != nil {
		targetID := *record.TargetID
		copyRecord.TargetID = &targetID
	}
	return &copyRecord
}

func newBlacklistPolicyFacadeForTest(t *testing.T, store *blacklistPolicyStoreStub) *BlacklistPolicyFacade {
	t.Helper()
	query, err := NewBlacklistPolicyQueryService(store)
	if err != nil {
		t.Fatal(err)
	}
	update, err := NewBlacklistPolicyUpdateService(store)
	if err != nil {
		t.Fatal(err)
	}
	facade, err := NewBlacklistPolicyFacade(query, update)
	if err != nil {
		t.Fatal(err)
	}
	return facade
}

func TestBlacklistPolicyFacadeCanonicalizesAndProjectsLocalPolicy(t *testing.T) {
	timestamp := time.Date(2026, 8, 5, 11, 0, 0, 0, time.UTC)
	store := &blacklistPolicyStoreStub{global: &BlacklistPolicyRecord{
		Scope: BlacklistPolicyScopeGlobal, Patterns: []string{}, UpdatedAt: timestamp,
	}}
	facade := newBlacklistPolicyFacadeForTest(t, store)

	initial, err := facade.GetGlobal(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if initial.Name != "blacklistPolicy" || initial.Patterns == nil || len(initial.Patterns) != 0 || initial.ETag == "" || !initial.UpdateTime.Equal(timestamp) {
		t.Fatalf("unexpected empty singleton projection: %#v", initial)
	}

	updated, err := facade.ReplaceGlobal(context.Background(), ReplaceBlacklistPolicyInput{
		ETag: initial.ETag, Patterns: []string{" EXAMPLE.com. ", "example.com", "192.0.2.19/24"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"192.0.2.0/24", "example.com"}
	if !reflect.DeepEqual(store.lastGlobalPatterns, want) || !reflect.DeepEqual(updated.Patterns, want) {
		t.Fatalf("canonical replacement = store:%v response:%v, want %v", store.lastGlobalPatterns, updated.Patterns, want)
	}
	if store.lastGlobalETag != initial.ETag || updated.Name != "blacklistPolicy" {
		t.Fatalf("replacement lost etag or resource name: %#v", updated)
	}
}

func TestBlacklistPolicyFacadeRejectsInvalidInputBeforeStore(t *testing.T) {
	store := &blacklistPolicyStoreStub{global: &BlacklistPolicyRecord{Scope: BlacklistPolicyScopeGlobal, Patterns: []string{}, UpdatedAt: time.Now().UTC()}}
	facade := newBlacklistPolicyFacadeForTest(t, store)
	if _, err := facade.ReplaceGlobal(context.Background(), ReplaceBlacklistPolicyInput{ETag: "", Patterns: []string{}}); !errors.Is(err, ErrBlacklistPolicyInvalidArgument) {
		t.Fatalf("missing etag error = %v", err)
	}
	if _, err := facade.ReplaceGlobal(context.Background(), ReplaceBlacklistPolicyInput{ETag: "current", Patterns: []string{"api.*.example.com"}}); !errors.Is(err, ErrBlacklistPolicyInvalidArgument) {
		t.Fatalf("invalid pattern error = %v", err)
	}
	if store.lastGlobalETag != "" || store.lastGlobalPatterns != nil {
		t.Fatalf("invalid input must not reach store: %#v", store)
	}
}

func TestBlacklistPolicyFacadeValidatesEffectivePatternsAndTargetIdentity(t *testing.T) {
	targetID := 42
	store := &blacklistPolicyStoreStub{
		targets:   map[int]*BlacklistPolicyRecord{targetID: {Scope: BlacklistPolicyScopeTarget, TargetID: &targetID, Patterns: []string{"*.example.com"}, UpdatedAt: time.Now().UTC()}},
		effective: []string{"*.example.com", "example.com"},
	}
	facade := newBlacklistPolicyFacadeForTest(t, store)

	local, err := facade.GetTarget(context.Background(), targetID)
	if err != nil {
		t.Fatal(err)
	}
	if local.Name != "targets/42/blacklistPolicy" || !reflect.DeepEqual(local.Patterns, []string{"*.example.com"}) {
		t.Fatalf("unexpected target projection: %#v", local)
	}
	patterns, err := facade.ResolveEffectivePatternsForScan(context.Background(), targetID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(patterns, []string{"*.example.com", "example.com"}) {
		t.Fatalf("effective patterns = %v", patterns)
	}

	store.effective = []string{"Example.com"}
	if _, err := facade.ResolveEffectivePatternsForScan(context.Background(), targetID); !errors.Is(err, ErrBlacklistPolicyDataIntegrity) {
		t.Fatalf("corrupt effective policy error = %v", err)
	}
	store.getTargetErr = ErrBlacklistPolicyNotFound
	if _, err := facade.GetTarget(context.Background(), targetID); !errors.Is(err, ErrBlacklistPolicyNotFound) {
		t.Fatalf("missing target policy error = %v", err)
	}
}

func TestBlacklistPolicyFacadeFailsFastOnMissingDependencies(t *testing.T) {
	if _, err := NewBlacklistPolicyQueryService(nil); !errors.Is(err, ErrBlacklistPolicyDependency) {
		t.Fatalf("query dependency error = %v", err)
	}
	if _, err := NewBlacklistPolicyUpdateService(nil); !errors.Is(err, ErrBlacklistPolicyDependency) {
		t.Fatalf("update dependency error = %v", err)
	}
	if _, err := NewBlacklistPolicyFacade(nil, nil); !errors.Is(err, ErrBlacklistPolicyDependency) {
		t.Fatalf("facade dependency error = %v", err)
	}
}
