package application

import (
	"context"
	"errors"
	"math"
	"testing"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type directoryMaterializerStub struct {
	items         []snapshotapp.DirectorySnapshotItem
	ctx           context.Context
	scanID        int
	targetID      int
	snapshotCount int64
	assetCount    int64
	err           error
	calls         int
}

func (stub *directoryMaterializerStub) SaveResultBatchContext(ctx context.Context, scanID int, targetID int, items []snapshotapp.DirectorySnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.calls++
	stub.ctx = ctx
	stub.scanID = scanID
	stub.targetID = targetID
	stub.items = append([]snapshotapp.DirectorySnapshotItem(nil), items...)
	return snapshotapp.MaterializationSummary{
		ReceivedItems: len(items),
		SnapshotCount: stub.snapshotCount,
		AssetCount:    stub.assetCount,
	}, stub.err
}

func TestResultIngestFacadeMaterializesDirectoryThroughClosedRegistry(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "directory-ingest")
	directories := &directoryMaterializerStub{snapshotCount: 2, assetCount: 2}
	coordinator := &resultMaterializationCoordinatorStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{
		Directories:     directories,
		Materialization: coordinator,
	})

	outcome, err := ingestResultStrings(ctx, facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: contractresults.ResultKindAssetDirectory,
		ItemsJSON: []string{
			`{"url":"https://Api.Example.com/%00?x=1#frag","status":0,"contentLength":0,"contentType":"","duration":0}`,
			`{"url":"https://api.example.com/max","status":999,"contentLength":9223372036854775807,"contentType":"application/octet-stream","duration":9223372036854775807}`,
		},
	})
	if err != nil {
		t.Fatalf("directory ingest failed: %v", err)
	}
	if coordinator.calls != 1 {
		t.Fatalf("directory ingest transaction calls = %d, want 1", coordinator.calls)
	}
	if outcome.ReceivedItems != 2 || outcome.DuplicateItems != 0 || outcome.ScopeFilteredItems != 0 || outcome.SnapshotCount != 2 || outcome.AssetCount != 2 {
		t.Fatalf("unexpected directory outcome: %+v", outcome)
	}
	if directories.ctx == nil || directories.ctx.Value(contextKey{}) != "directory-ingest" || directories.scanID != 12 || directories.targetID != 34 {
		t.Fatalf("directory materializer lost authenticated scope or context: scan=%d target=%d ctx=%v", directories.scanID, directories.targetID, directories.ctx)
	}
	if len(directories.items) != 2 {
		t.Fatalf("directory materialized item count = %d, want 2", len(directories.items))
	}
	first := directories.items[0]
	if first.URL != "https://Api.Example.com/%00?x=1#frag" || first.Status == nil || *first.Status != 0 || first.ContentLength == nil || *first.ContentLength != 0 || first.ContentType != "" || first.Duration == nil || *first.Duration != 0 {
		t.Fatalf("zero-value directory observation was not preserved: %+v", first)
	}
	last := directories.items[1]
	if last.ContentLength == nil || *last.ContentLength != math.MaxInt64 || last.Duration == nil || *last.Duration != math.MaxInt64 {
		t.Fatalf("MaxInt64 directory observation was not preserved: %+v", last)
	}
}

func TestResultIngestFacadeDirectoryRequiresTransactionCoordinatorBeforeWrite(t *testing.T) {
	directories := &directoryMaterializerStub{}
	facade := NewResultIngestFacade(ResultIngestFacadeDependencies{
		Directories: directories,
		ScanSummary: &resultSummaryUpdaterStub{},
	})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: contractresults.ResultKindAssetDirectory,
		ItemsJSON:  []string{`{"url":"https://example.com/admin","status":200,"contentLength":1,"contentType":"text/html","duration":2}`},
	})
	if !errors.Is(err, ErrResultMaterializerUnavailable) {
		t.Fatalf("missing transaction coordinator error = %v, want ErrResultMaterializerUnavailable", err)
	}
	if directories.calls != 0 {
		t.Fatalf("directory materializer called without transaction coordinator: %d", directories.calls)
	}
}
