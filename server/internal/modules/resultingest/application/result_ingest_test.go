package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/yyhuni/lunafox/contracts/results"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	snapshotinfrastructure "github.com/yyhuni/lunafox/server/internal/modules/snapshot/infrastructure"
)

type subdomainMaterializerStub struct {
	items         []snapshotapp.SubdomainSnapshotItem
	scanID        int
	targetID      int
	snapshotCount int64
	assetCount    int64
	scopeFiltered int
	unsupported   int
	err           error
	ctx           context.Context
}

type endpointMaterializerStub struct {
	snapshotCount int64
	assetCount    int64
	items         []snapshotapp.EndpointSnapshotItem
	err           error
	ctx           context.Context
}

type vulnerabilityMaterializerStub struct {
	items         []snapshotapp.VulnerabilitySnapshotItem
	ctx           context.Context
	scanID        int
	targetID      int
	snapshotCount int64
	assetCount    int64
	err           error
}

func (stub *vulnerabilityMaterializerStub) SaveResultBatchContext(ctx context.Context, scanID int, targetID int, items []snapshotapp.VulnerabilitySnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.ctx = ctx
	stub.scanID = scanID
	stub.targetID = targetID
	stub.items = append([]snapshotapp.VulnerabilitySnapshotItem(nil), items...)
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: stub.snapshotCount, AssetCount: stub.assetCount}, stub.err
}

type websiteTechnologyMaterializerStub struct {
	items []assetdomain.WebsiteTechnology
	calls int
	ctx   context.Context
	err   error
}

func (stub *websiteTechnologyMaterializerStub) BatchUpsertTechnologyContext(ctx context.Context, _ int, items []assetapp.WebsiteTechnologyUpsertItem) (assetapp.WebsiteTechnologyMaterializationSummary, error) {
	stub.calls++
	stub.ctx = ctx
	stub.items = append([]assetdomain.WebsiteTechnology(nil), items...)
	return assetapp.WebsiteTechnologyMaterializationSummary{ReceivedItems: len(items), AssetCount: int64(len(items))}, stub.err
}

func (stub *endpointMaterializerStub) SaveAndSyncContext(ctx context.Context, _ int, _ int, items []snapshotapp.EndpointSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.ctx = ctx
	stub.items = append([]snapshotapp.EndpointSnapshotItem(nil), items...)
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: stub.snapshotCount, AssetCount: stub.assetCount}, stub.err
}

type resultMaterializationCoordinatorStub struct {
	calls  int
	scopes []ResultMaterializationScope
	err    error
}

func (stub *resultMaterializationCoordinatorStub) Materialize(ctx context.Context, scope ResultMaterializationScope, persist func(context.Context) error) error {
	stub.calls++
	stub.scopes = append(stub.scopes, scope)
	if stub.err != nil {
		return stub.err
	}
	return persist(ctx)
}

type dedupeHostPortSnapshotStoreStub struct {
	seen map[string]struct{}
}

func (stub *dedupeHostPortSnapshotStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
	if stub.seen == nil {
		stub.seen = map[string]struct{}{}
	}
	var affected int64
	for _, snapshot := range snapshots {
		key := fmt.Sprintf("%d|%s|%s|%d", snapshot.ScanID, snapshot.Host, snapshot.IP, snapshot.Port)
		if _, exists := stub.seen[key]; exists {
			continue
		}
		stub.seen[key] = struct{}{}
		affected++
	}
	return affected, nil
}

type dedupeHostPortAssetSyncStub struct {
	seen map[string]struct{}
}

func (stub *dedupeHostPortAssetSyncStub) BatchUpsertContext(_ context.Context, targetID int, items []snapshotapp.HostPortAssetItem) (int64, error) {
	if stub.seen == nil {
		stub.seen = map[string]struct{}{}
	}
	var affected int64
	for _, item := range items {
		key := fmt.Sprintf("%d|%s|%s|%d", targetID, item.Host, item.IP, item.Port)
		if _, exists := stub.seen[key]; exists {
			continue
		}
		stub.seen[key] = struct{}{}
		affected++
	}
	return affected, nil
}

func (stub *subdomainMaterializerStub) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []snapshotapp.SubdomainSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.ctx = ctx
	stub.scanID = scanID
	stub.targetID = targetID
	stub.items = append([]snapshotapp.SubdomainSnapshotItem(nil), items...)
	summary := snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: stub.snapshotCount, AssetCount: stub.assetCount, ScopeFilteredItems: stub.scopeFiltered, UnsupportedItems: stub.unsupported}
	if stub.err != nil {
		summary.AssetCount = 0
		return summary, stub.err
	}
	summary.DuplicateItems = completedStubDuplicateCount(len(items), stub.snapshotCount, stub.scopeFiltered, stub.unsupported)
	return summary, nil
}

type resultSummaryUpdaterStub struct {
	called   bool
	scanID   int
	targetID int
	err      error
	ctx      context.Context
}

type hostPortMaterializerStub struct {
	items         []snapshotapp.HostPortSnapshotItem
	ctx           context.Context
	scanID        int
	targetID      int
	snapshotCount int64
	assetCount    int64
	scopeFiltered int
	unsupported   int
	err           error
}

type websiteMaterializerStub struct {
	items         []snapshotapp.WebsiteSnapshotItem
	ctx           context.Context
	scanID        int
	targetID      int
	snapshotCount int64
	assetCount    int64
	scopeFiltered int
	unsupported   int
	err           error
}

type screenshotMaterializerStub struct {
	items []snapshotapp.ScreenshotSnapshotItem
	ctx   context.Context
}

func (stub *screenshotMaterializerStub) SaveResultBatchContext(ctx context.Context, _ int, _ int, items []snapshotapp.ScreenshotSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.ctx = ctx
	stub.items = append([]snapshotapp.ScreenshotSnapshotItem(nil), items...)
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: int64(len(items)), AssetCount: int64(len(items))}, nil
}

func (stub *hostPortMaterializerStub) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []snapshotapp.HostPortSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.ctx = ctx
	stub.scanID = scanID
	stub.targetID = targetID
	stub.items = append([]snapshotapp.HostPortSnapshotItem(nil), items...)
	summary := snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: stub.snapshotCount, AssetCount: stub.assetCount, ScopeFilteredItems: stub.scopeFiltered, UnsupportedItems: stub.unsupported}
	if stub.err != nil {
		summary.AssetCount = 0
		return summary, stub.err
	}
	summary.DuplicateItems = completedStubDuplicateCount(len(items), stub.snapshotCount, stub.scopeFiltered, stub.unsupported)
	return summary, nil
}

func (stub *websiteMaterializerStub) SaveAndSyncContext(ctx context.Context, scanID int, targetID int, items []snapshotapp.WebsiteSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	stub.ctx = ctx
	stub.scanID = scanID
	stub.targetID = targetID
	stub.items = append([]snapshotapp.WebsiteSnapshotItem(nil), items...)
	summary := snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: stub.snapshotCount, AssetCount: stub.assetCount, ScopeFilteredItems: stub.scopeFiltered, UnsupportedItems: stub.unsupported}
	if stub.err != nil {
		summary.AssetCount = 0
		return summary, stub.err
	}
	summary.DuplicateItems = completedStubDuplicateCount(len(items), stub.snapshotCount, stub.scopeFiltered, stub.unsupported)
	return summary, nil
}

func completedStubDuplicateCount(received int, snapshots int64, scopeFiltered, unsupported int) int {
	classified := scopeFiltered + unsupported + int(snapshots)
	if snapshots < 0 || classified > received {
		return 0
	}
	return received - classified
}

func (stub *resultSummaryUpdaterStub) RefreshScanResultSummary(ctx context.Context, scanID int, targetID int) error {
	stub.ctx = ctx
	stub.called = true
	stub.scanID = scanID
	stub.targetID = targetID
	return stub.err
}

func newTestResultIngestFacade(deps ResultIngestFacadeDependencies) *ResultIngestFacade {
	if isNilResultDependency(deps.ScanSummary) {
		deps.ScanSummary = &resultSummaryUpdaterStub{}
	}
	if isNilResultDependency(deps.Materialization) {
		deps.Materialization = &resultMaterializationCoordinatorStub{}
	}
	return NewResultIngestFacade(deps)
}

type testResultIngestInput struct {
	ScanID     int
	TargetID   int
	ResultType string
	ItemsJSON  []string
}

func ingestResultStrings(ctx context.Context, facade *ResultIngestFacade, input testResultIngestInput) (ResultIngestOutcome, error) {
	items := make([][]byte, len(input.ItemsJSON))
	for index := range input.ItemsJSON {
		items[index] = []byte(input.ItemsJSON[index])
	}
	return facade.Ingest(ctx, ResultIngestCommand{
		TaskID:       101,
		ScanID:       input.ScanID,
		TargetID:     input.TargetID,
		AgentID:      17,
		SessionID:    "session-17",
		SessionEpoch: 23,
		ResultType:   input.ResultType,
		Items:        items,
	})
}

func TestResultIngestFacadeBatchIngestTaskResultsMaterializesSubdomains(t *testing.T) {
	subdomains := &subdomainMaterializerStub{snapshotCount: 1, assetCount: 1}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON:  []string{`{"dnsName":"api.example.com"}`},
	})
	if err != nil {
		t.Fatalf("batch ingest failed: %v", err)
	}
	if output.ReceivedItems != 1 || output.DuplicateItems != 0 || output.SnapshotCount != 1 || output.AssetCount != 1 {
		t.Fatalf("unexpected output: %+v", output)
	}
	if subdomains.scanID != 12 || subdomains.targetID != 34 {
		t.Fatalf("unexpected materializer scope scan=%d target=%d", subdomains.scanID, subdomains.targetID)
	}
	if len(subdomains.items) != 1 || subdomains.items[0].DNSName != "api.example.com" {
		t.Fatalf("unexpected materialized items: %+v", subdomains.items)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsMaterializesHostPorts(t *testing.T) {
	hostPorts := &hostPortMaterializerStub{snapshotCount: 1, assetCount: 1}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: hostPorts})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetHostPort,
		ItemsJSON:  []string{`{"host":"api.example.com","ip":"192.0.2.10","port":443}`},
	})
	if err != nil {
		t.Fatalf("batch ingest failed: %v", err)
	}
	if output.ReceivedItems != 1 || output.DuplicateItems != 0 || output.ScopeFilteredItems != 0 || output.UnsupportedItems != 0 || output.SnapshotCount != 1 || output.AssetCount != 1 {
		t.Fatalf("unexpected output: %+v", output)
	}
	if hostPorts.scanID != 12 || hostPorts.targetID != 34 {
		t.Fatalf("unexpected materializer scope scan=%d target=%d", hostPorts.scanID, hostPorts.targetID)
	}
	if len(hostPorts.items) != 1 || hostPorts.items[0].Host != "api.example.com" || hostPorts.items[0].IP != "192.0.2.10" || hostPorts.items[0].Port != 443 {
		t.Fatalf("unexpected materialized items: %+v", hostPorts.items)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsMaterializesWebsites(t *testing.T) {
	websites := &websiteMaterializerStub{snapshotCount: 1, assetCount: 1}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: websites})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsite,
		ItemsJSON:  []string{`{"url":"HTTPS://Api.Example.COM:443/login?b=2&a=1#fragment","host":"api.example.com","title":"API","statusCode":200,"contentLength":42,"tech":["nginx"],"vhost":false}`},
	})
	if err != nil {
		t.Fatalf("batch ingest failed: %v", err)
	}
	if output.ReceivedItems != 1 || output.DuplicateItems != 0 || output.SnapshotCount != 1 || output.AssetCount != 1 {
		t.Fatalf("unexpected output: %+v", output)
	}
	if websites.scanID != 12 || websites.targetID != 34 {
		t.Fatalf("unexpected materializer scope scan=%d target=%d", websites.scanID, websites.targetID)
	}
	if len(websites.items) != 1 || websites.items[0].URL != "HTTPS://Api.Example.COM:443/login?b=2&a=1#fragment" || websites.items[0].Host != "api.example.com" || websites.items[0].Title != "API" {
		t.Fatalf("unexpected materialized website items: %+v", websites.items)
	}
	if websites.items[0].StatusCode == nil || *websites.items[0].StatusCode != 200 || websites.items[0].ContentLength == nil || *websites.items[0].ContentLength != 42 {
		t.Fatalf("unexpected materialized numeric fields: %+v", websites.items[0])
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsRefreshesScanSummaryForSubdomains(t *testing.T) {
	subdomains := &subdomainMaterializerStub{snapshotCount: 2, assetCount: 2}
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains, ScanSummary: summary})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON: []string{
			`{"dnsName":"api.example.com"}`,
			`{"dnsName":"admin.example.com"}`,
		},
	})
	if err != nil {
		t.Fatalf("batch ingest failed: %v", err)
	}
	if !summary.called {
		t.Fatal("expected scan summary refresh")
	}
	if summary.scanID != 12 || summary.targetID != 34 {
		t.Fatalf("unexpected summary scope scan=%d target=%d", summary.scanID, summary.targetID)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsRefreshesScanSummaryForHostPorts(t *testing.T) {
	hostPorts := &hostPortMaterializerStub{snapshotCount: 2, assetCount: 2}
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: hostPorts, ScanSummary: summary})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetHostPort,
		ItemsJSON: []string{
			`{"host":"api.example.com","ip":"192.0.2.10","port":443}`,
			`{"host":"admin.example.com","ip":"192.0.2.11","port":443}`,
		},
	})
	if err != nil {
		t.Fatalf("batch ingest failed: %v", err)
	}
	if !summary.called {
		t.Fatal("expected scan summary refresh")
	}
	if summary.scanID != 12 || summary.targetID != 34 {
		t.Fatalf("unexpected summary scope scan=%d target=%d", summary.scanID, summary.targetID)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsRefreshesScanSummaryForWebsites(t *testing.T) {
	websites := &websiteMaterializerStub{snapshotCount: 2, assetCount: 2}
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: websites, ScanSummary: summary})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsite,
		ItemsJSON: []string{
			`{"url":"https://api.example.com","host":"api.example.com"}`,
			`{"url":"https://admin.example.com","host":"admin.example.com"}`,
		},
	})
	if err != nil {
		t.Fatalf("batch ingest failed: %v", err)
	}
	if !summary.called {
		t.Fatal("expected scan summary refresh")
	}
	if summary.scanID != 12 || summary.targetID != 34 {
		t.Fatalf("unexpected summary scope scan=%d target=%d", summary.scanID, summary.targetID)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReturnsSummaryRefreshErrors(t *testing.T) {
	summaryErr := errors.New("summary refresh failed")
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{
		Subdomains:  &subdomainMaterializerStub{snapshotCount: 1, assetCount: 1},
		ScanSummary: &resultSummaryUpdaterStub{err: summaryErr},
	})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON:  []string{`{"dnsName":"api.example.com"}`},
	})
	if !errors.Is(err, summaryErr) {
		t.Fatalf("expected summary refresh error, got %v", err)
	}
	if output != (ResultIngestOutcome{}) {
		t.Fatalf("rolled-back batch returned outcome after summary failure: %+v", output)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsSkipsSummaryRefreshOnMaterializationError(t *testing.T) {
	materializeErr := errors.New("asset projection failed")
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: &subdomainMaterializerStub{snapshotCount: 1, err: materializeErr}, ScanSummary: summary})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON:  []string{`{"dnsName":"api.example.com"}`},
	})
	if !errors.Is(err, materializeErr) {
		t.Fatalf("expected materialization error, got %v", err)
	}
	if summary.called {
		t.Fatal("summary refresh must not run after failed materialization")
	}
}

func TestResultIngestFacadeEndpointUsesFinalTransactionCoordinator(t *testing.T) {
	endpoints := &endpointMaterializerStub{snapshotCount: 1, assetCount: 1}
	summary := &resultSummaryUpdaterStub{}
	coordinator := &resultMaterializationCoordinatorStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{
		Endpoints:       endpoints,
		ScanSummary:     summary,
		Materialization: coordinator,
	})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetEndpoint,
		ItemsJSON: []string{`{"url":"https://api.example.com","host":"api.example.com"}`},
	})
	if err != nil {
		t.Fatalf("endpoint ingest failed: %v", err)
	}
	if coordinator.calls != 1 {
		t.Fatalf("endpoint ingest transaction calls = %d, want 1", coordinator.calls)
	}
	if !summary.called {
		t.Fatal("endpoint ingest must refresh the summary in the final transaction")
	}
	if len(endpoints.items) != 1 || output.SnapshotCount != 1 || output.AssetCount != 1 {
		t.Fatalf("unexpected endpoint outcome: output=%+v items=%+v", output, endpoints.items)
	}
}

func TestResultIngestFacadeEndpointSummaryFailureReturnsNoCommittedOutcome(t *testing.T) {
	summaryErr := errors.New("summary refresh failed")
	endpoints := &endpointMaterializerStub{snapshotCount: 1, assetCount: 1}
	coordinator := &resultMaterializationCoordinatorStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{
		Endpoints:       endpoints,
		ScanSummary:     &resultSummaryUpdaterStub{err: summaryErr},
		Materialization: coordinator,
	})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetEndpoint,
		ItemsJSON: []string{`{"url":"https://api.example.com","host":"api.example.com"}`},
	})
	if !errors.Is(err, summaryErr) {
		t.Fatalf("expected summary error, got %v", err)
	}
	if coordinator.calls != 1 || len(endpoints.items) != 1 {
		t.Fatalf("endpoint summary failure must occur in its transaction: coordinator=%d items=%+v", coordinator.calls, endpoints.items)
	}
	if output != (ResultIngestOutcome{}) {
		t.Fatalf("rolled-back batch returned outcome: %+v", output)
	}
}

func TestResultIngestFacadeRejectsInvalidEndpointBatchWithoutMaterialization(t *testing.T) {
	endpoints := &endpointMaterializerStub{snapshotCount: 1, assetCount: 1}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Endpoints: endpoints})
	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetEndpoint,
		ItemsJSON: []string{
			`{"url":"https://api.example.com","host":"api.example.com"}`,
			`{"url":"https://api.example.com","host":"other.example.com"}`,
		},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("endpoint mixed batch error = %v, want ErrInvalidResultItems", err)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("invalid endpoint batch returned outcome = %+v", outcome)
	}
	if len(endpoints.items) != 0 {
		t.Fatalf("invalid endpoint batch reached materialization: %#v", endpoints.items)
	}
}

func TestResultIngestFacadeEndpointPreservesCallerContext(t *testing.T) {
	type contextKey struct{}
	endpoints := &endpointMaterializerStub{snapshotCount: 1, assetCount: 1}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Endpoints: endpoints})
	ctx := context.WithValue(context.Background(), contextKey{}, "request-boundary")
	_, err := ingestResultStrings(ctx, facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetEndpoint,
		ItemsJSON: []string{`{"url":"https://api.example.com","host":"api.example.com"}`},
	})
	if err != nil {
		t.Fatalf("endpoint ingest error = %v", err)
	}
	if endpoints.ctx != ctx || endpoints.ctx.Value(contextKey{}) != "request-boundary" {
		t.Fatalf("caller context was not propagated to endpoint materialization")
	}
}

func TestResultIngestFacadeRefreshesSummaryInsideFinalTransaction(t *testing.T) {
	coordinator := &resultMaterializationCoordinatorStub{}
	summary := &resultSummaryUpdaterStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{
		Subdomains:      &subdomainMaterializerStub{snapshotCount: 1, assetCount: 1},
		ScanSummary:     summary,
		Materialization: coordinator,
	})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON: []string{`{"dnsName":"api.example.com"}`},
	})
	if err != nil {
		t.Fatalf("subdomain ingest failed: %v", err)
	}
	if coordinator.calls != 1 {
		t.Fatalf("result transaction calls = %d, want 1", coordinator.calls)
	}
	if !summary.called {
		t.Fatal("summary refresh must run inside the final transaction callback")
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsRejectsUnsupportedResultType(t *testing.T) {
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: &subdomainMaterializerStub{}})

	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: "asset.unknown.v1",
		ItemsJSON:  []string{`{"dnsName":"api.example.com"}`},
	})
	if !errors.Is(err, ErrUnsupportedResultType) {
		t.Fatalf("expected ErrUnsupportedResultType, got %v", err)
	}
}

func TestResultIngestFacadeAcceptsEveryCanonicalDescriptorWithoutEngineMembership(t *testing.T) {
	for _, descriptor := range results.Descriptors() {
		descriptor := descriptor
		t.Run(descriptor.ResultType, func(t *testing.T) {
			type contextKey struct{}
			callerContext := context.WithValue(context.Background(), contextKey{}, descriptor.ResultType)
			coordinator := &resultMaterializationCoordinatorStub{}
			summary := &resultSummaryUpdaterStub{}
			var facade *ResultIngestFacade
			var item string
			var observedContext func() context.Context
			currentOnly := false
			switch descriptor.ResultType {
			case results.ResultKindAssetSubdomain:
				materializer := &subdomainMaterializerStub{snapshotCount: 1, assetCount: 1}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"dnsName":"api.example.com"}`
			case results.ResultKindAssetHostPort:
				materializer := &hostPortMaterializerStub{snapshotCount: 1, assetCount: 1}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"host":"api.example.com","ip":"192.0.2.10","port":443}`
			case results.ResultKindAssetWebsite:
				materializer := &websiteMaterializerStub{snapshotCount: 1, assetCount: 1}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"url":"https://api.example.com","host":"api.example.com"}`
			case results.ResultKindAssetWebsiteTechnology:
				materializer := &websiteTechnologyMaterializerStub{}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{WebsiteTechnologies: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				currentOnly = true
				item = `{"url":"https://api.example.com","tech":["nginx"]}`
			case results.ResultKindAssetEndpoint:
				materializer := &endpointMaterializerStub{snapshotCount: 1, assetCount: 1}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{Endpoints: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"url":"https://api.example.com","host":"api.example.com"}`
			case results.ResultKindAssetScreenshot:
				materializer := &screenshotMaterializerStub{}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{Screenshots: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"url":"https://api.example.com","image":"UklGRhYAAABXRUJQVlA4IAoAAAAAAACdASoBAAEA"}`
			case results.ResultKindAssetDirectory:
				materializer := &directoryMaterializerStub{snapshotCount: 1, assetCount: 1}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{
					Directories:     materializer,
					ScanSummary:     summary,
					Materialization: coordinator,
				})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"url":"https://api.example.com/admin","status":200,"contentLength":0,"contentType":"","duration":0}`
			case results.ResultKindAssetVulnerability:
				materializer := &vulnerabilityMaterializerStub{snapshotCount: 1, assetCount: 1}
				facade = newTestResultIngestFacade(ResultIngestFacadeDependencies{Vulnerabilities: materializer, ScanSummary: summary, Materialization: coordinator})
				observedContext = func() context.Context { return materializer.ctx }
				item = `{"url":"https://api.example.com","vulnType":"xss","severity":"high","source":"nuclei","cvssScore":7.5,"description":"reflected XSS","rawOutput":{"template-id":"xss"}}`
			default:
				t.Fatalf("canonical descriptor %q has no Server materialization path", descriptor.ResultType)
			}

			_, err := ingestResultStrings(callerContext, facade, testResultIngestInput{
				ScanID: 12, TargetID: 34, ResultType: descriptor.ResultType, ItemsJSON: []string{item},
			})
			if err != nil {
				t.Fatalf("canonical result submission failed: %v", err)
			}
			wantScope := ResultMaterializationScope{TaskID: 101, ScanID: 12, TargetID: 34, AgentID: 17, SessionID: "session-17", SessionEpoch: 23}
			if coordinator.calls != 1 || len(coordinator.scopes) != 1 || coordinator.scopes[0] != wantScope {
				t.Fatalf("result family bypassed final execution fence: calls=%d scopes=%+v", coordinator.calls, coordinator.scopes)
			}
			if got := observedContext(); got == nil || got.Value(contextKey{}) != descriptor.ResultType {
				t.Fatalf("result family lost caller context before materialization: %v", got)
			}
			if currentOnly {
				if summary.called {
					t.Fatal("current-only Website technology result must not refresh the Scan summary")
				}
				return
			}
			if !summary.called || summary.ctx == nil || summary.ctx.Value(contextKey{}) != descriptor.ResultType {
				t.Fatalf("result family lost caller context before summary refresh: %+v", summary)
			}
		})
	}
}

func TestResultIngestFacadeRejectsNonCanonicalResultTypeWhitespace(t *testing.T) {
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: &subdomainMaterializerStub{}})
	_, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: " " + results.ResultKindAssetSubdomain, ItemsJSON: []string{`{"dnsName":"api.example.com"}`},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("expected non-canonical result type rejection, got %v", err)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReportsMalformedItemRejection(t *testing.T) {
	subdomains := &subdomainMaterializerStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains})

	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON:  []string{`{"dnsName":`},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("malformed item error = %v, want ErrInvalidResultItems", err)
	}
	if subdomains.items != nil {
		t.Fatalf("malformed item reached materializer: %+v", subdomains.items)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("malformed item returned outcome = %+v", outcome)
	}
}

func TestResultIngestFacadeReportsUnknownResultItemFieldsWithoutMaterialization(t *testing.T) {
	subdomains := &subdomainMaterializerStub{}
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: subdomains})
	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON: []string{`{"dnsName":"api.example.com","engineId":"engine.other"}`},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("unknown-field item error = %v, want ErrInvalidResultItems", err)
	}
	if subdomains.items != nil {
		t.Fatalf("invalid result item must not reach materializer: %+v", subdomains.items)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("unknown-field result returned outcome = %+v", outcome)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReportsInvalidHostPortItems(t *testing.T) {
	for _, item := range []string{
		`{"host":"","ip":"192.0.2.10","port":443}`,
		`{"host":" ","ip":"192.0.2.10","port":443}`,
		`{"host":"api.example.com","ip":"not-ip","port":443}`,
		`{"host":"api.example.com","ip":"2001:db8::1","port":443}`,
		`{"host":"api.example.com","ip":"192.0.2.0/24","port":443}`,
		`{"host":"api.example.com","ip":"192.0.2.10","port":0}`,
		`{"host":"api.example.com","ip":"192.0.2.10","port":65536}`,
	} {
		hostPorts := &hostPortMaterializerStub{}
		facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: hostPorts})
		outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
			ScanID:     12,
			TargetID:   34,
			ResultType: results.ResultKindAssetHostPort,
			ItemsJSON:  []string{item},
		})
		if !errors.Is(err, ErrInvalidResultItems) {
			t.Fatalf("invalid host-port item error for %s = %v, want ErrInvalidResultItems", item, err)
		}
		if hostPorts.items != nil {
			t.Fatalf("invalid host-port item reached materialization: %+v", hostPorts.items)
		}
		if outcome != (ResultIngestOutcome{}) {
			t.Fatalf("invalid host-port returned outcome = %+v", outcome)
		}
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReportsInvalidWebsiteItems(t *testing.T) {
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: &websiteMaterializerStub{}})

	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsite,
		ItemsJSON:  []string{`{"url":"ftp://example.com"}`},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("invalid Website item error = %v, want ErrInvalidResultItems", err)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("invalid Website returned outcome = %+v", outcome)
	}
}

func TestResultIngestFacadeRejectsMalformedWebsiteTechnologyBatchWithoutMaterialization(t *testing.T) {
	technologies := &websiteTechnologyMaterializerStub{}
	coordinator := &resultMaterializationCoordinatorStub{}
	facade := NewResultIngestFacade(ResultIngestFacadeDependencies{
		WebsiteTechnologies: technologies,
		Materialization:     coordinator,
		ScanSummary:         &resultSummaryUpdaterStub{},
	})

	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsiteTechnology,
		ItemsJSON: []string{
			`{"url":"https://api.example.com","tech":["nginx"]}`,
			`{"url":"https://api.example.com","tech":null}`,
		},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("malformed Website technology batch error = %v, want ErrInvalidResultItems", err)
	}
	if technologies.calls != 0 || len(technologies.items) != 0 {
		t.Fatalf("invalid Website technology batch was materialized: calls=%d items=%#v", technologies.calls, technologies.items)
	}
	if coordinator.calls != 0 {
		t.Fatalf("invalid Website technology batch reached transaction coordinator: calls=%d", coordinator.calls)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("invalid Website technology batch returned outcome = %+v", outcome)
	}
}

func TestResultIngestFacadeWebsiteTechnologyUsesCurrentOnlyBoundary(t *testing.T) {
	technologies := &websiteTechnologyMaterializerStub{}
	coordinator := &resultMaterializationCoordinatorStub{}
	summary := &resultSummaryUpdaterStub{}
	facade := NewResultIngestFacade(ResultIngestFacadeDependencies{
		WebsiteTechnologies: technologies,
		Materialization:     coordinator,
		ScanSummary:         summary,
	})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsiteTechnology,
		ItemsJSON:  []string{`{"url":"https://api.example.com","tech":["nginx"]}`},
	})
	if err != nil {
		t.Fatalf("current-only Website technology ingest failed: %v", err)
	}
	if technologies.calls != 1 || len(technologies.items) != 1 {
		t.Fatalf("Website technology materialization = calls=%d items=%#v", technologies.calls, technologies.items)
	}
	if coordinator.calls != 1 || summary.called {
		t.Fatalf("current-only result touched scan-evidence paths: transactionCalls=%d summaryCalled=%t", coordinator.calls, summary.called)
	}
	if output.ReceivedItems != 1 || output.SnapshotCount != 0 || output.AssetCount != 1 {
		t.Fatalf("current-only result outcome = %+v", output)
	}
}

func TestResultIngestFacadeUsesObservedScreenshotURLAndRejectsInvalidURL(t *testing.T) {
	t.Run("observed URL is accepted", func(t *testing.T) {
		materializer := &screenshotMaterializerStub{}
		facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Screenshots: materializer})
		outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
			ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetScreenshot,
			ItemsJSON: []string{`{"url":"https://api.example.com/final","image":"UklGRhYAAABXRUJQVlA4IAoAAAAAAACdASoBAAEA"}`},
		})
		if err != nil {
			t.Fatalf("observed screenshot URL was rejected: %v", err)
		}
		if len(materializer.items) != 1 || materializer.items[0].URL != "https://api.example.com/final" || outcome.AssetCount != 1 {
			t.Fatalf("observed screenshot materialization = items=%#v outcome=%+v", materializer.items, outcome)
		}
	})

	for _, rawURL := range []string{"", "ftp://api.example.com/final"} {
		t.Run(rawURL, func(t *testing.T) {
			materializer := &screenshotMaterializerStub{}
			facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Screenshots: materializer})
			outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
				ScanID: 12, TargetID: 34, ResultType: results.ResultKindAssetScreenshot,
				ItemsJSON: []string{`{"url":` + strconv.Quote(rawURL) + `,"image":"UklGRhYAAABXRUJQVlA4IAoAAAAAAACdASoBAAEA"}`},
			})
			if !errors.Is(err, ErrInvalidResultItems) {
				t.Fatalf("invalid screenshot URL %q error = %v", rawURL, err)
			}
			if outcome != (ResultIngestOutcome{}) || materializer.items != nil {
				t.Fatalf("invalid screenshot URL %q reached materialization: outcome=%+v items=%#v", rawURL, outcome, materializer.items)
			}
		})
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReturnsMaterializationErrors(t *testing.T) {
	materializeErr := errors.New("asset projection failed")
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Subdomains: &subdomainMaterializerStub{snapshotCount: 1, err: materializeErr}})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetSubdomain,
		ItemsJSON:  []string{`{"dnsName":"api.example.com"}`},
	})
	if !errors.Is(err, materializeErr) {
		t.Fatalf("expected materialization error, got %v", err)
	}
	if output != (ResultIngestOutcome{}) {
		t.Fatalf("failed batch returned outcome: %+v", output)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReturnsHostPortMaterializationErrors(t *testing.T) {
	materializeErr := errors.New("asset projection failed")
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: &hostPortMaterializerStub{snapshotCount: 1, err: materializeErr}})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetHostPort,
		ItemsJSON:  []string{`{"host":"api.example.com","ip":"192.0.2.10","port":443}`},
	})
	if !errors.Is(err, materializeErr) {
		t.Fatalf("expected materialization error, got %v", err)
	}
	if output != (ResultIngestOutcome{}) {
		t.Fatalf("failed batch returned outcome: %+v", output)
	}
}

func TestResultIngestFacadeBatchIngestTaskResultsReturnsWebsiteMaterializationErrors(t *testing.T) {
	materializeErr := errors.New("asset projection failed")
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: &websiteMaterializerStub{snapshotCount: 1, err: materializeErr}})

	output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsite,
		ItemsJSON:  []string{`{"url":"https://api.example.com","host":"api.example.com"}`},
	})
	if !errors.Is(err, materializeErr) {
		t.Fatalf("expected materialization error, got %v", err)
	}
	if output != (ResultIngestOutcome{}) {
		t.Fatalf("failed batch returned outcome: %+v", output)
	}
}

type resultingestHostPortSnapshotStoreStub struct {
	snapshots []snapshotdomain.HostPortSnapshot
}

func (stub *resultingestHostPortSnapshotStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.HostPortSnapshot) (int64, error) {
	stub.snapshots = append([]snapshotdomain.HostPortSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type resultingestHostPortAssetSyncStub struct {
	items []snapshotapp.HostPortAssetItem
}

func (stub *resultingestHostPortAssetSyncStub) BatchUpsertContext(_ context.Context, _ int, items []snapshotapp.HostPortAssetItem) (int64, error) {
	stub.items = append([]snapshotapp.HostPortAssetItem(nil), items...)
	return int64(len(items)), nil
}

type resultingestWebsiteSnapshotStoreStub struct {
	snapshots []snapshotdomain.WebsiteSnapshot
}

func (stub *resultingestWebsiteSnapshotStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
	stub.snapshots = append([]snapshotdomain.WebsiteSnapshot(nil), snapshots...)
	return int64(len(snapshots)), nil
}

type resultingestWebsiteAssetSyncStub struct {
	items []snapshotapp.WebsiteAssetUpsertItem
}

func (stub *resultingestWebsiteAssetSyncStub) BatchUpsertContext(_ context.Context, _ int, items []snapshotapp.WebsiteAssetUpsertItem) (int64, error) {
	stub.items = append([]snapshotapp.WebsiteAssetUpsertItem(nil), items...)
	return int64(len(items)), nil
}

type dedupeWebsiteSnapshotStoreStub struct {
	seen map[string]struct{}
}

func (stub *dedupeWebsiteSnapshotStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.WebsiteSnapshot) (int64, error) {
	if stub.seen == nil {
		stub.seen = map[string]struct{}{}
	}
	for _, snapshot := range snapshots {
		key := fmt.Sprintf("%d|%s", snapshot.ScanID, snapshot.URL)
		stub.seen[key] = struct{}{}
	}
	return int64(len(snapshots)), nil
}

type dedupeWebsiteAssetSyncStub struct {
	seen map[string]struct{}
}

func (stub *dedupeWebsiteAssetSyncStub) BatchUpsertContext(_ context.Context, targetID int, items []snapshotapp.WebsiteAssetUpsertItem) (int64, error) {
	if stub.seen == nil {
		stub.seen = map[string]struct{}{}
	}
	for _, item := range items {
		key := fmt.Sprintf("%d|%s", targetID, item.URL)
		stub.seen[key] = struct{}{}
	}
	return int64(len(items)), nil
}

type resultingestSnapshotLookupStub struct {
	scan      *snapshotdomain.ScanRef
	target    *snapshotdomain.ScanTargetRef
	scanCtx   context.Context
	targetCtx context.Context
}

func (stub *resultingestSnapshotLookupStub) GetScanRefByID(int) (*snapshotdomain.ScanRef, error) {
	return stub.scan, nil
}

func (stub *resultingestSnapshotLookupStub) GetScanRefByIDContext(ctx context.Context, _ int) (*snapshotdomain.ScanRef, error) {
	stub.scanCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.scan, nil
}

func (stub *resultingestSnapshotLookupStub) GetTargetRefByScanID(int) (*snapshotdomain.ScanTargetRef, error) {
	return stub.target, nil
}

func (stub *resultingestSnapshotLookupStub) GetTargetRefByScanIDContext(ctx context.Context, _ int) (*snapshotdomain.ScanTargetRef, error) {
	stub.targetCtx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.target, nil
}

func TestResultIngestFacadeHostPortScopeFilteringUsesSnapshotBoundary(t *testing.T) {
	tests := []struct {
		name            string
		targetName      string
		targetType      string
		itemsJSON       []string
		wantHosts       []string
		wantScope       int
		wantUnsupported int
	}{
		{
			name:       "domain suffix",
			targetName: "example.com",
			targetType: "domain",
			itemsJSON: []string{
				`{"host":"example.com","ip":"192.0.2.10","port":443}`,
				`{"host":"api.example.com","ip":"192.0.2.11","port":443}`,
				`{"host":"evil.com","ip":"192.0.2.12","port":443}`,
			},
			wantHosts:       []string{"example.com", "api.example.com"},
			wantScope:       1,
			wantUnsupported: 0,
		},
		{
			name:       "ip by item ip",
			targetName: "192.0.2.10",
			targetType: "ip",
			itemsJSON: []string{
				`{"host":"api.example.com","ip":"192.0.2.10","port":443}`,
				`{"host":"192.0.2.11","ip":"192.0.2.11","port":443}`,
			},
			wantHosts: []string{"api.example.com"},
			wantScope: 1,
		},
		{
			name:       "cidr by parsed item ip",
			targetName: "192.0.2.0/24",
			targetType: "cidr",
			itemsJSON: []string{
				`{"host":"api.example.com","ip":"192.0.2.10","port":443}`,
				`{"host":"outside.example.com","ip":"198.51.100.10","port":443}`,
			},
			wantHosts: []string{"api.example.com"},
			wantScope: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &resultingestHostPortSnapshotStoreStub{}
			assetSync := &resultingestHostPortAssetSyncStub{}
			lookup := &resultingestSnapshotLookupStub{
				scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
				target: &snapshotdomain.ScanTargetRef{ID: 34, Name: tt.targetName, Type: tt.targetType},
			}
			cmd := snapshotapp.NewHostPortSnapshotCommandService(store, lookup, assetSync)
			hostPortFacade := snapshotapp.NewHostPortSnapshotFacade(nil, cmd)
			facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: hostPortFacade})

			output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
				ScanID:     12,
				TargetID:   34,
				ResultType: results.ResultKindAssetHostPort,
				ItemsJSON:  tt.itemsJSON,
			})
			if !errors.Is(err, ErrResultNotAuthorized) {
				t.Fatalf("batch ingest error = %v, want ErrResultNotAuthorized", err)
			}
			if output != (ResultIngestOutcome{}) {
				t.Fatalf("unauthorized batch returned outcome: %+v", output)
			}
		})
	}
}

func TestResultIngestFacadeHostPortRetryIsIdempotentThroughSnapshotBoundary(t *testing.T) {
	store := &dedupeHostPortSnapshotStoreStub{}
	assetSync := &dedupeHostPortAssetSyncStub{}
	lookup := &resultingestSnapshotLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
		target: &snapshotdomain.ScanTargetRef{ID: 34, Name: "example.com", Type: "domain"},
	}
	cmd := snapshotapp.NewHostPortSnapshotCommandService(store, lookup, assetSync)
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{HostPorts: snapshotapp.NewHostPortSnapshotFacade(nil, cmd)})
	input := testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetHostPort,
		ItemsJSON:  []string{`{"host":"api.example.com","ip":"192.0.2.10","port":443}`},
	}

	first, err := ingestResultStrings(context.Background(), facade, input)
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	second, err := ingestResultStrings(context.Background(), facade, input)
	if err != nil {
		t.Fatalf("retry ingest failed: %v", err)
	}
	if first.ReceivedItems != 1 || first.DuplicateItems != 0 || first.SnapshotCount != 1 || first.AssetCount != 1 {
		t.Fatalf("unexpected first output: %+v", first)
	}
	if second.ReceivedItems != 1 || second.DuplicateItems != 1 || second.SnapshotCount != 0 || second.AssetCount != 0 {
		t.Fatalf("unexpected retry output: %+v", second)
	}
}

func TestResultIngestFacadeWebsiteScopeFilteringUsesSnapshotBoundary(t *testing.T) {
	tests := []struct {
		name       string
		targetName string
		targetType string
		itemsJSON  []string
		wantURLs   []string
		wantScope  int
	}{
		{
			name:       "domain suffix",
			targetName: "example.com",
			targetType: "domain",
			itemsJSON: []string{
				`{"url":"https://example.com","host":"example.com"}`,
				`{"url":"https://api.example.com","host":"api.example.com"}`,
				`{"url":"https://evil.com","host":"evil.com"}`,
			},
			wantURLs:  []string{"https://example.com", "https://api.example.com"},
			wantScope: 1,
		},
		{
			name:       "ip target",
			targetName: "192.0.2.10",
			targetType: "ip",
			itemsJSON: []string{
				`{"url":"https://192.0.2.10","host":"192.0.2.10"}`,
				`{"url":"https://192.0.2.11","host":"192.0.2.11"}`,
			},
			wantURLs:  []string{"https://192.0.2.10"},
			wantScope: 1,
		},
		{
			name:       "cidr target",
			targetName: "192.0.2.0/24",
			targetType: "cidr",
			itemsJSON: []string{
				`{"url":"https://192.0.2.10","host":"192.0.2.10"}`,
				`{"url":"https://198.51.100.10","host":"198.51.100.10"}`,
			},
			wantURLs:  []string{"https://192.0.2.10"},
			wantScope: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &resultingestWebsiteSnapshotStoreStub{}
			assetSync := &resultingestWebsiteAssetSyncStub{}
			lookup := &resultingestSnapshotLookupStub{
				scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
				target: &snapshotdomain.ScanTargetRef{ID: 34, Name: tt.targetName, Type: tt.targetType},
			}
			cmd := snapshotapp.NewWebsiteSnapshotCommandService(store, lookup, assetSync)
			facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: snapshotapp.NewWebsiteSnapshotFacade(nil, cmd)})

			output, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
				ScanID:     12,
				TargetID:   34,
				ResultType: results.ResultKindAssetWebsite,
				ItemsJSON:  tt.itemsJSON,
			})
			if !errors.Is(err, ErrResultNotAuthorized) {
				t.Fatalf("batch ingest error = %v, want ErrResultNotAuthorized", err)
			}
			if output != (ResultIngestOutcome{}) {
				t.Fatalf("unauthorized batch returned outcome: %+v", output)
			}
		})
	}
}

func TestResultIngestFacadeWebsiteRetryIsIdempotentThroughSnapshotBoundary(t *testing.T) {
	store := &dedupeWebsiteSnapshotStoreStub{}
	assetSync := &dedupeWebsiteAssetSyncStub{}
	lookup := &resultingestSnapshotLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
		target: &snapshotdomain.ScanTargetRef{ID: 34, Name: "example.com", Type: "domain"},
	}
	cmd := snapshotapp.NewWebsiteSnapshotCommandService(store, lookup, assetSync)
	facade := newTestResultIngestFacade(ResultIngestFacadeDependencies{Websites: snapshotapp.NewWebsiteSnapshotFacade(nil, cmd)})
	input := testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetWebsite,
		ItemsJSON:  []string{`{"url":"https://api.example.com","host":"api.example.com"}`},
	}

	first, err := ingestResultStrings(context.Background(), facade, input)
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	second, err := ingestResultStrings(context.Background(), facade, input)
	if err != nil {
		t.Fatalf("retry ingest failed: %v", err)
	}
	if first.ReceivedItems != 1 || first.DuplicateItems != 0 || first.SnapshotCount != 1 || first.AssetCount != 1 {
		t.Fatalf("unexpected first output: %+v", first)
	}
	if second.ReceivedItems != 1 || second.DuplicateItems != 0 || second.SnapshotCount != 1 || second.AssetCount != 1 {
		t.Fatalf("unexpected retry output: %+v", second)
	}
}

type resultingestVulnerabilitySnapshotStoreStub struct {
	snapshots []snapshotdomain.VulnerabilitySnapshot
	seen      map[string]struct{}
}

func (stub *resultingestVulnerabilitySnapshotStoreStub) BatchCreate(snapshots []snapshotdomain.VulnerabilitySnapshot) (int64, error) {
	return stub.BatchCreateContext(context.Background(), snapshots)
}

func (stub *resultingestVulnerabilitySnapshotStoreStub) BatchCreateContext(_ context.Context, snapshots []snapshotdomain.VulnerabilitySnapshot) (int64, error) {
	if stub.seen == nil {
		stub.seen = make(map[string]struct{})
	}
	var created int64
	for _, snapshot := range snapshots {
		key := resultingestVulnerabilitySnapshotKey(snapshot)
		if _, exists := stub.seen[key]; exists {
			continue
		}
		stub.seen[key] = struct{}{}
		stub.snapshots = append(stub.snapshots, snapshot)
		created++
	}
	return created, nil
}

func resultingestVulnerabilitySnapshotKey(snapshot snapshotdomain.VulnerabilitySnapshot) string {
	return fmt.Sprintf("%d\x00%s\x00%s\x00%s", snapshot.ScanID, snapshot.URL, snapshot.VulnType, snapshot.Severity)
}

type resultingestVulnerabilityAssetSyncStub struct {
	items map[string]snapshotapp.VulnerabilityAssetCreateItem
}

func (stub *resultingestVulnerabilityAssetSyncStub) BatchCreate(targetID int, items []snapshotapp.VulnerabilityAssetCreateItem) (int64, error) {
	return stub.BatchCreateContext(context.Background(), targetID, items)
}

func (stub *resultingestVulnerabilityAssetSyncStub) BatchCreateContext(_ context.Context, targetID int, items []snapshotapp.VulnerabilityAssetCreateItem) (int64, error) {
	if stub.items == nil {
		stub.items = make(map[string]snapshotapp.VulnerabilityAssetCreateItem)
	}
	for _, item := range items {
		key := fmt.Sprintf("%d\x00%s\x00%s\x00%s", targetID, item.URL, item.VulnType, item.Severity)
		stub.items[key] = item
	}
	return int64(len(items)), nil
}

func newNucleiVulnerabilityResultIngestFacade(store snapshotapp.VulnerabilitySnapshotCommandStore, assets snapshotapp.VulnerabilityAssetSync) *ResultIngestFacade {
	lookup := &resultingestSnapshotLookupStub{
		scan:   &snapshotdomain.ScanRef{ID: 12, TargetID: 34},
		target: &snapshotdomain.ScanTargetRef{ID: 34, Name: "example.com", Type: "domain"},
	}
	command := snapshotapp.NewVulnerabilitySnapshotCommandService(store, lookup, assets, snapshotinfrastructure.NewVulnerabilityRawOutputCodec())
	vulnerabilities := snapshotapp.NewVulnerabilitySnapshotFacade(nil, command)
	return newTestResultIngestFacade(ResultIngestFacadeDependencies{Vulnerabilities: vulnerabilities})
}

func TestResultIngestFacadeNucleiVulnerabilityMapsCompleteParsedOutput(t *testing.T) {
	store := &resultingestVulnerabilitySnapshotStoreStub{}
	assets := &resultingestVulnerabilityAssetSyncStub{}
	facade := newNucleiVulnerabilityResultIngestFacade(store, assets)

	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetVulnerability,
		ItemsJSON: []string{
			`{"url":"https://api.example.com/admin","vulnType":"http-missing-header","severity":"high","source":"nuclei","cvssScore":7.5,"description":"Missing security header","rawOutput":{"template-id":"http-missing-header","request":"redacted-at-ingest-boundary"}}`,
		},
	})
	if err != nil {
		t.Fatalf("Nuclei vulnerability ingest failed: %v", err)
	}
	if outcome.ReceivedItems != 1 || outcome.SnapshotCount != 1 || outcome.AssetCount != 1 || outcome.RejectedItems != 0 {
		t.Fatalf("Nuclei vulnerability outcome = %+v", outcome)
	}
	if len(store.snapshots) != 1 || len(assets.items) != 1 {
		t.Fatalf("Nuclei vulnerability persistence = snapshots=%d assets=%d", len(store.snapshots), len(assets.items))
	}
	snapshot := store.snapshots[0]
	if snapshot.URL != "https://api.example.com/admin" || snapshot.VulnType != "http-missing-header" || snapshot.Severity != "high" || snapshot.Source != "nuclei" || snapshot.Description != "Missing security header" {
		t.Fatalf("mapped Nuclei snapshot = %#v", snapshot)
	}
	if snapshot.CVSSScore == nil || snapshot.CVSSScore.String() != "7.5" {
		t.Fatalf("mapped Nuclei CVSS score = %v", snapshot.CVSSScore)
	}
	var rawOutput map[string]any
	if err := json.Unmarshal(snapshot.RawOutput, &rawOutput); err != nil {
		t.Fatalf("decode persisted raw output: %v", err)
	}
	if rawOutput["template-id"] != "http-missing-header" || rawOutput["request"] != "redacted-at-ingest-boundary" {
		t.Fatalf("persisted parsed raw output = %#v", rawOutput)
	}
}

func TestResultIngestFacadeNucleiVulnerabilityRejectsMixedInvalidBatchBeforePersistence(t *testing.T) {
	store := &resultingestVulnerabilitySnapshotStoreStub{}
	assets := &resultingestVulnerabilityAssetSyncStub{}
	facade := newNucleiVulnerabilityResultIngestFacade(store, assets)

	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetVulnerability,
		ItemsJSON: []string{
			`{"url":"https://api.example.com/admin","vulnType":"valid-template","severity":"high","source":"nuclei","rawOutput":{"template-id":"valid-template"}}`,
			`{"url":"https://api.example.com/admin","vulnType":"invalid-score","severity":"high","source":"nuclei","cvssScore":10.1,"rawOutput":{"template-id":"invalid-score"}}`,
		},
	})
	if !errors.Is(err, ErrInvalidResultItems) {
		t.Fatalf("mixed Nuclei batch error = %v, want ErrInvalidResultItems", err)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("mixed invalid Nuclei batch returned outcome = %+v", outcome)
	}
	if len(store.snapshots) != 0 || len(assets.items) != 0 {
		t.Fatalf("mixed invalid Nuclei batch reached persistence: snapshots=%d assets=%d", len(store.snapshots), len(assets.items))
	}
}

func TestResultIngestFacadeNucleiVulnerabilityRejectsOutOfScopeBatchBeforePersistence(t *testing.T) {
	store := &resultingestVulnerabilitySnapshotStoreStub{}
	assets := &resultingestVulnerabilityAssetSyncStub{}
	facade := newNucleiVulnerabilityResultIngestFacade(store, assets)

	outcome, err := ingestResultStrings(context.Background(), facade, testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetVulnerability,
		ItemsJSON: []string{
			`{"url":"https://api.example.com/admin","vulnType":"in-scope","severity":"high","source":"nuclei","rawOutput":{"template-id":"in-scope"}}`,
			`{"url":"https://outside.example.net/admin","vulnType":"outside-scope","severity":"high","source":"nuclei","rawOutput":{"template-id":"outside-scope"}}`,
		},
	})
	if !errors.Is(err, ErrResultNotAuthorized) {
		t.Fatalf("out-of-scope Nuclei batch error = %v, want ErrResultNotAuthorized", err)
	}
	if outcome != (ResultIngestOutcome{}) {
		t.Fatalf("out-of-scope Nuclei batch returned outcome = %+v", outcome)
	}
	if len(store.snapshots) != 0 || len(assets.items) != 0 {
		t.Fatalf("out-of-scope Nuclei batch reached persistence: snapshots=%d assets=%d", len(store.snapshots), len(assets.items))
	}
}

func TestResultIngestFacadeNucleiVulnerabilityReplayConvergesNaturalKeys(t *testing.T) {
	store := &resultingestVulnerabilitySnapshotStoreStub{}
	assets := &resultingestVulnerabilityAssetSyncStub{}
	facade := newNucleiVulnerabilityResultIngestFacade(store, assets)
	firstInput := testResultIngestInput{
		ScanID:     12,
		TargetID:   34,
		ResultType: results.ResultKindAssetVulnerability,
		ItemsJSON:  []string{`{"url":"https://api.example.com/admin","vulnType":"replay-template","severity":"high","source":"nuclei","description":"first observation","rawOutput":{"template-id":"replay-template","sequence":1}}`},
	}
	secondInput := firstInput
	secondInput.ItemsJSON = []string{`{"url":"https://api.example.com/admin","vulnType":"replay-template","severity":"high","source":"nuclei","description":"latest observation","rawOutput":{"template-id":"replay-template","sequence":2}}`}

	first, err := ingestResultStrings(context.Background(), facade, firstInput)
	if err != nil {
		t.Fatalf("first Nuclei ingest failed: %v", err)
	}
	second, err := ingestResultStrings(context.Background(), facade, secondInput)
	if err != nil {
		t.Fatalf("replayed Nuclei ingest failed: %v", err)
	}
	if first.SnapshotCount != 1 || first.DuplicateItems != 0 || second.SnapshotCount != 0 || second.DuplicateItems != 1 {
		t.Fatalf("Nuclei replay outcomes = first=%+v second=%+v", first, second)
	}
	if len(store.snapshots) != 1 || len(assets.items) != 1 {
		t.Fatalf("Nuclei replay did not converge natural keys: snapshots=%d assets=%d", len(store.snapshots), len(assets.items))
	}
	for _, item := range assets.items {
		if item.Description != "latest observation" {
			t.Fatalf("current vulnerability projection did not retain latest observation: %#v", item)
		}
	}
}
