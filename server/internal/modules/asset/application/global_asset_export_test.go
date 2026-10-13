package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type exportSnapshotStub struct {
	websites  []assetdomain.Website
	endpoints []assetdomain.Endpoint
}

func (stub *exportSnapshotStub) ForEachWebsite(ctx context.Context, visit func(assetdomain.Website) error) error {
	for _, item := range stub.websites {
		if err := visit(item); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (stub *exportSnapshotStub) ForEachEndpoint(ctx context.Context, visit func(assetdomain.Endpoint) error) error {
	for _, item := range stub.endpoints {
		if err := visit(item); err != nil {
			return err
		}
	}
	return ctx.Err()
}

type exportStoreStub struct {
	snapshot GlobalAssetExportSnapshot
	after    func()
}

func (stub *exportStoreStub) WithSnapshot(ctx context.Context, _ GlobalAssetSearchAST, _ GlobalAssetSearchAssetType, visit func(GlobalAssetExportSnapshot) error) error {
	if err := visit(stub.snapshot); err != nil {
		return err
	}
	if stub.after != nil {
		stub.after()
	}
	return ctx.Err()
}

func TestGlobalAssetExportRunnerEmitsUnifiedCSVAfterPreflight(t *testing.T) {
	stub := &exportStoreStub{snapshot: &exportSnapshotStub{websites: []assetdomain.Website{{ID: 7, TargetID: 3, URL: "https://example.test/a,b", Tech: []string{"nginx", "tls"}, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}}}}
	runner := NewGlobalAssetExportServiceRunner(stub)
	var output bytes.Buffer
	if err := runner.Export(context.Background(), GlobalAssetSearchAST{Mode: GlobalAssetSearchModePlainURL, PlainURL: "example"}, GlobalAssetSearchAssetTypeWebsite, func(size int64) (io.Writer, error) {
		if size <= 0 {
			t.Fatal("expected positive CSV size")
		}
		return &output, nil
	}); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	data := output.String()
	if !strings.HasPrefix(data, "\ufeffid,target_id,url") {
		t.Fatalf("CSV header = %q", data[:min(len(data), 40)])
	}
	if !strings.Contains(data, `"https://example.test/a,b"`) {
		t.Fatalf("CSV did not escape comma field: %q", data)
	}
}

func TestGlobalAssetExportRunnerRejectsRowsBeforeBegin(t *testing.T) {
	items := make([]assetdomain.Website, GlobalAssetExportMaxRows+1)
	for index := range items {
		items[index].ID = index + 1
		items[index].TargetID = 1
		items[index].URL = "https://example.test"
	}
	runner := NewGlobalAssetExportServiceRunner(&exportStoreStub{snapshot: &exportSnapshotStub{websites: items}})
	called := false
	err := runner.Export(context.Background(), GlobalAssetSearchAST{Mode: GlobalAssetSearchModePlainURL, PlainURL: "example"}, GlobalAssetSearchAssetTypeWebsite, func(int64) (io.Writer, error) { called = true; return io.Discard, nil })
	if !errors.Is(err, ErrGlobalAssetExportRows) {
		t.Fatalf("Export() error = %v, want row limit", err)
	}
	if called {
		t.Fatal("begin called for over-budget export")
	}
}

func TestGlobalAssetExportRunnerDoesNotReleaseTailAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stub := &exportStoreStub{snapshot: &exportSnapshotStub{websites: []assetdomain.Website{{ID: 1, TargetID: 1, URL: "https://example.test"}}}}
	stub.after = cancel
	runner := NewGlobalAssetExportServiceRunner(stub)
	var output bytes.Buffer
	err := runner.Export(ctx, GlobalAssetSearchAST{Mode: GlobalAssetSearchModePlainURL, PlainURL: "example"}, GlobalAssetSearchAssetTypeWebsite, func(int64) (io.Writer, error) { return &output, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Export() error = %v, want cancellation", err)
	}
	if output.Len() != 0 {
		t.Fatalf("tail released after cancellation: %d bytes", output.Len())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
