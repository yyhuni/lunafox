package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/yyhuni/lunafox/contracts/results"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

func TestResultIngestFacadeIngestRejectsWholeBatchWhenAnyItemIsInvalid(t *testing.T) {
	for invalidIndex := 0; invalidIndex < 3; invalidIndex++ {
		t.Run(fmt.Sprintf("invalid-at-%d", invalidIndex), func(t *testing.T) {
			subdomains := &subdomainMaterializerStub{snapshotCount: 2, assetCount: 2}
			facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains})
			items := [][]byte{
				[]byte(`{"dnsName":"api.example.com"}`),
				[]byte(`{"dnsName":"www.example.com"}`),
				[]byte(`{"dnsName":"admin.example.com"}`),
			}
			items[invalidIndex] = []byte(`{"dnsName":"not a dns name"}`)
			outcome, err := facade.Ingest(context.Background(), ResultIngestCommand{
				TaskID: 101, ScanID: 12, TargetID: 34,
				ResultType: results.ResultKindAssetSubdomain, Items: items,
			})
			if !errors.Is(err, ErrInvalidResultItems) {
				t.Fatalf("mixed batch error = %v, want ErrInvalidResultItems", err)
			}
			if outcome != (ResultIngestOutcome{}) {
				t.Fatalf("rejected mixed batch returned outcome: %+v", outcome)
			}
			if len(subdomains.items) != 0 {
				t.Fatalf("invalid batch reached materializer: %+v", subdomains.items)
			}
		})
	}
}

func TestResultIngestFacadeIngestRejectsInvalidSingleItemWithoutMaterialization(t *testing.T) {
	hostPorts := &hostPortMaterializerStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: hostPorts})
	outcome, err := facade.Ingest(context.Background(), ResultIngestCommand{
		TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetHostPort,
		Items: [][]byte{[]byte(`{"host":"[2001:db8::1]","ip":"2001:db8::1","port":443}`)},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("invalid batch error = %v, want ErrInvalidResultItems", err)
	}
	if len(hostPorts.items) != 0 {
		t.Fatal("rejected IPv6 item reached the materializer")
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("rejected batch returned outcome: %+v", outcome)
	}
}

func TestResultIngestFacadeIngestRejectsMalformedBatchBeforeMaterialization(t *testing.T) {
	subdomains := &subdomainMaterializerStub{snapshotCount: 1, assetCount: 1}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains})
	outcome, err := facade.Ingest(context.Background(), ResultIngestCommand{
		TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		Items: [][]byte{
			[]byte(`{"dnsName":"api.example.com"}`),
			[]byte(`{"dnsName":`),
			{0xff},
		},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("malformed batch error = %v, want ErrInvalidResultItems", err)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("malformed batch returned outcome: %+v", outcome)
	}
	if len(subdomains.items) != 0 {
		t.Fatalf("malformed batch reached materializer: %+v", subdomains.items)
	}
}

func TestResultIngestFacadeIngestRejectsMaterializationSummaryBeforeCommit(t *testing.T) {
	subdomains := &subdomainMaterializerStub{snapshotCount: 2, assetCount: 1, scopeFiltered: 1, unsupported: 1}
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains, ScanSummary: summary})

	outcome, err := facade.Ingest(context.Background(), ResultIngestCommand{
		TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		Items: [][]byte{
			[]byte(`{"dnsName":"api.example.com"}`),
			[]byte(`{"dnsName":"www.example.com"}`),
			[]byte(`{"dnsName":"admin.example.com"}`),
			[]byte(`{"dnsName":"cdn.example.com"}`),
			[]byte(`{"dnsName":"mail.example.com"}`),
		},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("Ingest error = %v, want ErrInvalidResultItems", err)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("rejected batch returned outcome: %+v", outcome)
	}
	if summary.called {
		t.Fatal("summary refresh ran for a rejected batch")
	}
	fields := reflect.VisibleFields(reflect.TypeOf(outcome))
	for _, field := range fields {
		if field.Name == "AcceptedItems" || field.Name == "TotalItems" || field.Name == "Success" || field.Name == "Ack" {
			t.Fatalf("Server-local outcome must not expose cross-hop acknowledgement field %s", field.Name)
		}
	}
}

func TestResultIngestFacadeIngestRejectsMissingOrCancelledContext(t *testing.T) {
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: &subdomainMaterializerStub{}})
	command := ResultIngestCommand{TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain, Items: [][]byte{[]byte(`{"dnsName":"api.example.com"}`)}}
	if _, err := facade.Ingest(nil, command); !errors.Is(err, ErrResultIngestContextRequired) {
		t.Fatalf("expected nil-context error, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := facade.Ingest(ctx, command); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation to propagate before writes, got %v", err)
	}
}

func TestResultIngestFacadeRequiresSummaryUpdaterBeforeAnyWrite(t *testing.T) {
	materializer := &subdomainMaterializerStub{}
	facade := NewResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: materializer})
	_, err := facade.Ingest(context.Background(), ResultIngestCommand{
		TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		Items: [][]byte{[]byte(`{"dnsName":"api.example.com"}`)},
	})
	if !errors.Is(err, ErrResultMaterializerUnavailable) {
		t.Fatalf("missing summary updater error = %v", err)
	}
	if materializer.items != nil {
		t.Fatalf("missing summary updater reached persistence: %+v", materializer.items)
	}
}

func TestResultIngestFacadeIngestClassifiesUnknownAndMissingTypesAsPreWriteFailures(t *testing.T) {
	materializer := &subdomainMaterializerStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: materializer})
	base := ResultIngestCommand{TaskID: 101, ScanID: 12, TargetID: 34, Items: [][]byte{[]byte(`{"dnsName":"api.example.com"}`)}}

	unknown := base
	unknown.ResultType = "asset.unknown.v1"
	if _, err := facade.Ingest(context.Background(), unknown); !errors.Is(err, ErrUnsupportedResultType) {
		t.Fatalf("unknown type error = %v", err)
	}
	missing := base
	if _, err := facade.Ingest(context.Background(), missing); !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("missing type error = %v", err)
	}
	if materializer.items != nil {
		t.Fatalf("type failures reached materialization: %+v", materializer.items)
	}
}

func TestResultIngestFacadeIngestPreservesCallerContextThroughAllSynchronousEffects(t *testing.T) {
	type contextKey struct{}
	wantValue := &struct{ marker string }{marker: "caller"}
	ctx := context.WithValue(context.Background(), contextKey{}, wantValue)
	store := &contextSubdomainSnapshotStore{}
	assets := &contextSubdomainAssetSync{}
	lookup := &resultingestSnapshotLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
		target: &snapshotdomain.ScanTargetRef{ID: 34, Name: "example.com", Type: "domain"},
	}
	command := snapshotapp.NewSubdomainSnapshotCommandService(store, lookup, assets)
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{
		Subdomains: snapshotapp.NewSubdomainSnapshotFacade(nil, command), ScanSummary: summary,
	})

	_, err := facade.Ingest(ctx, ResultIngestCommand{
		TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		Items: [][]byte{[]byte(`{"dnsName":"api.example.com"}`)},
	})
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	for name, observed := range map[string]context.Context{
		"scan lookup": lookup.scanCtx, "target lookup": lookup.targetCtx,
		"snapshot": store.ctx, "asset": assets.ctx, "summary": summary.ctx,
	} {
		if observed == nil || observed.Value(contextKey{}) != wantValue {
			t.Fatalf("%s stage did not receive original caller context", name)
		}
	}
}

func TestResultIngestFacadeIngestCancellationAfterSnapshotWriteReplaysToConvergence(t *testing.T) {
	store := &cancellingDedupeSubdomainSnapshotStore{seen: map[string]struct{}{}}
	assets := &contextSubdomainAssetSync{}
	lookup := &resultingestSnapshotLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
		target: &snapshotdomain.ScanTargetRef{ID: 34, Name: "example.com", Type: "domain"},
	}
	command := snapshotapp.NewSubdomainSnapshotCommandService(store, lookup, assets)
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: snapshotapp.NewSubdomainSnapshotFacade(nil, command)})
	batch := ResultIngestCommand{
		TaskID: 101, ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		Items: [][]byte{[]byte(`{"dnsName":"api.example.com"}`)},
	}

	firstCtx, cancel := context.WithCancel(context.Background())
	store.cancel = cancel
	first, err := facade.Ingest(firstCtx, batch)
	if !errors.Is(err, context.Canceled) || first != (ResultIngestOutcome{}) {
		t.Fatalf("first attempt = %+v, %v", first, err)
	}
	store.cancel = nil
	second, err := facade.Ingest(context.Background(), batch)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if second.ReceivedItems != 1 || second.DuplicateItems != 1 || second.SnapshotCount != 0 || second.AssetCount != 1 {
		t.Fatalf("replay did not converge effects: %+v", second)
	}
}

func TestResultIngestOutcomeRejectsImpossibleMaterializerSummaries(t *testing.T) {
	for _, summary := range []materializationSummary{
		{receivedItems: 2, snapshotCount: 1},
		{receivedItems: 1, snapshotCount: 2},
		{receivedItems: 1, snapshotCount: 1, duplicateItems: 1},
	} {
		if _, err := resultIngestOutcome(1, summary, true); !errors.Is(err, ErrResultMaterializationInvariant) {
			t.Fatalf("summary %+v error = %v", summary, err)
		}
	}
}

func TestResultIngestOutcomeReportsRejectedItems(t *testing.T) {
	outcome, err := resultIngestOutcome(1, materializationSummary{receivedItems: 1, invalidItems: 1}, true)
	if err != nil {
		t.Fatalf("all-rejected summary failed: %v", err)
	}
	if outcome.RejectedItems != 1 || outcome.SnapshotCount != 0 || outcome.AssetCount != 0 {
		t.Fatalf("unexpected rejected-item outcome: %+v", outcome)
	}
}

func TestResultIngestOutcomeDoesNotInventDuplicatesForPartialFailure(t *testing.T) {
	outcome, err := resultIngestOutcome(3, materializationSummary{receivedItems: 3, snapshotCount: 1}, false)
	if err != nil {
		t.Fatalf("partial outcome error = %v", err)
	}
	if outcome.DuplicateItems != 0 || outcome.SnapshotCount != 1 {
		t.Fatalf("partial outcome invented classifications: %+v", outcome)
	}
}

type contextSubdomainSnapshotStore struct {
	ctx context.Context
}

func (store *contextSubdomainSnapshotStore) BatchCreateContext(ctx context.Context, snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	store.ctx = ctx
	return int64(len(snapshots)), nil
}

type cancellingDedupeSubdomainSnapshotStore struct {
	seen   map[string]struct{}
	cancel context.CancelFunc
}

func (store *cancellingDedupeSubdomainSnapshotStore) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.SubdomainSnapshot) (int64, error) {
	var affected int64
	for _, snapshot := range snapshots {
		key := fmt.Sprintf("%d|%s", snapshot.ScanID, snapshot.DNSName)
		if _, exists := store.seen[key]; exists {
			continue
		}
		store.seen[key] = struct{}{}
		affected++
	}
	if store.cancel != nil {
		store.cancel()
	}
	return affected, nil
}

type contextSubdomainAssetSync struct {
	ctx  context.Context
	seen map[string]struct{}
}

func (sync *contextSubdomainAssetSync) BatchCreateContext(ctx context.Context, targetID int, dnsNames []string) (int, error) {
	sync.ctx = ctx
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if sync.seen == nil {
		sync.seen = map[string]struct{}{}
	}
	affected := 0
	for _, dnsName := range dnsNames {
		key := fmt.Sprintf("%d|%s", targetID, dnsName)
		if _, exists := sync.seen[key]; exists {
			continue
		}
		sync.seen[key] = struct{}{}
		affected++
	}
	return affected, nil
}
