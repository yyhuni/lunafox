package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

type globalAssetSearchWebsiteStoreStub struct {
	items           []assetdomain.Website
	err             error
	query           GlobalAssetSearchStoreQuery
	calls           int
	totalSize       int64
	totalSizeCapped bool
}

func (stub *globalAssetSearchWebsiteStoreStub) SearchGlobalWebsites(_ context.Context, query GlobalAssetSearchStoreQuery) ([]assetdomain.Website, int64, bool, error) {
	stub.calls++
	stub.query = query
	if stub.err != nil {
		return nil, 0, false, stub.err
	}
	return append([]assetdomain.Website(nil), stub.items...), stub.totalSize, stub.totalSizeCapped, nil
}

type globalAssetSearchEndpointStoreStub struct {
	items           []assetdomain.Endpoint
	err             error
	query           GlobalAssetSearchStoreQuery
	calls           int
	totalSize       int64
	totalSizeCapped bool
}

type globalAssetSearchScreenshotStoreStub struct {
	items     []assetdomain.Screenshot
	err       error
	calls     int
	targetIDs []int
	urlGroups [][]string
}

func (stub *globalAssetSearchScreenshotStoreStub) ListSummariesByTargetAndURLs(targetID int, urls []string) ([]assetdomain.Screenshot, error) {
	stub.calls++
	stub.targetIDs = append(stub.targetIDs, targetID)
	stub.urlGroups = append(stub.urlGroups, append([]string(nil), urls...))
	if stub.err != nil {
		return nil, stub.err
	}
	return append([]assetdomain.Screenshot(nil), stub.items...), nil
}

func (stub *globalAssetSearchEndpointStoreStub) SearchGlobalEndpoints(_ context.Context, query GlobalAssetSearchStoreQuery) ([]assetdomain.Endpoint, int64, bool, error) {
	stub.calls++
	stub.query = query
	if stub.err != nil {
		return nil, 0, false, stub.err
	}
	return append([]assetdomain.Endpoint(nil), stub.items...), stub.totalSize, stub.totalSizeCapped, nil
}

func TestGlobalAssetSearchServiceDefaultsAndSingleTableDispatch(t *testing.T) {
	now := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	websiteStore := &globalAssetSearchWebsiteStoreStub{items: []assetdomain.Website{{ID: 3, CreatedAt: now}, {ID: 2, CreatedAt: now}, {ID: 1, CreatedAt: now}}}
	endpointStore := &globalAssetSearchEndpointStoreStub{}
	service := NewGlobalAssetSearchService(websiteStore, endpointStore, &globalAssetSearchScreenshotStoreStub{})

	pageSize := 2
	result, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: "example", AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if endpointStore.calls != 0 || websiteStore.calls != 1 {
		t.Fatalf("search must select one store, website=%d endpoint=%d", websiteStore.calls, endpointStore.calls)
	}
	if websiteStore.query.PageSize != 2 || websiteStore.query.AST.Mode != GlobalAssetSearchModePlainURL {
		t.Fatalf("unexpected repository query: %+v", websiteStore.query)
	}
	if len(result.Websites) != 2 || result.NextPageToken == "" {
		t.Fatalf("expected keyset page, got %+v", result)
	}
}

func TestGlobalAssetSearchServiceEnrichesWebsiteScreenshotsByTargetAndURL(t *testing.T) {
	now := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	status := int16(200)
	websiteStore := &globalAssetSearchWebsiteStoreStub{items: []assetdomain.Website{
		{ID: 1, TargetID: 7, URL: "https://one.example.test"},
		{ID: 2, TargetID: 7, URL: "https://two.example.test"},
		{ID: 3, TargetID: 8, URL: "https://one.example.test"},
	}}
	screenshotStore := &globalAssetSearchScreenshotStoreStub{items: []assetdomain.Screenshot{
		{ID: 41, TargetID: 7, URL: "https://one.example.test", StatusCode: &status, CreatedAt: now, UpdatedAt: now},
	}}
	service := NewGlobalAssetSearchService(websiteStore, &globalAssetSearchEndpointStoreStub{}, screenshotStore)

	result, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: "example", AssetType: GlobalAssetSearchAssetTypeWebsite})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if screenshotStore.calls != 2 {
		t.Fatalf("expected one screenshot summary query per target, got %d", screenshotStore.calls)
	}
	if result.Websites[0].Screenshot == nil || result.Websites[0].Screenshot.ID != 41 {
		t.Fatalf("expected matching screenshot summary on first website, got %+v", result.Websites[0].Screenshot)
	}
	if result.Websites[1].Screenshot != nil || result.Websites[2].Screenshot != nil {
		t.Fatalf("screenshot summaries must remain scoped to target and URL: %+v", result.Websites)
	}
}

func TestGlobalAssetSearchServicePropagatesScreenshotSummaryErrors(t *testing.T) {
	service := NewGlobalAssetSearchService(
		&globalAssetSearchWebsiteStoreStub{items: []assetdomain.Website{{ID: 1, TargetID: 7, URL: "https://example.test"}}},
		&globalAssetSearchEndpointStoreStub{},
		&globalAssetSearchScreenshotStoreStub{err: errors.New("screenshot lookup failed")},
	)
	if _, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: "example", AssetType: GlobalAssetSearchAssetTypeWebsite}); err == nil || err.Error() != "screenshot lookup failed" {
		t.Fatalf("screenshot lookup errors must not become an empty screenshot state, got %v", err)
	}
}

func TestGlobalAssetSearchServiceBindsPageTokenToShapeAndCursor(t *testing.T) {
	now := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	websiteStore := &globalAssetSearchWebsiteStoreStub{items: []assetdomain.Website{{ID: 3, CreatedAt: now}, {ID: 2, CreatedAt: now}}}
	service := NewGlobalAssetSearchService(websiteStore, &globalAssetSearchEndpointStoreStub{}, &globalAssetSearchScreenshotStoreStub{})

	pageSize := 1
	first, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: `host="api"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize})
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	websiteStore.items = []assetdomain.Website{{ID: 1, CreatedAt: now.Add(-time.Second)}}
	second, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: `host="api"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize, PageToken: first.NextPageToken})
	if err != nil {
		t.Fatalf("second search: %v", err)
	}
	if websiteStore.query.Cursor == nil || websiteStore.query.Cursor.ID != 3 || len(second.Websites) != 1 || second.Websites[0].ID != 1 {
		t.Fatalf("cursor or page result lost: query=%+v result=%+v", websiteStore.query, second)
	}
	mismatchedPageSize := 2
	for _, mismatched := range []GlobalAssetSearchInput{
		{Query: `host="web"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize, PageToken: first.NextPageToken},
		{Query: `host="api"`, AssetType: GlobalAssetSearchAssetTypeEndpoint, PageSize: &pageSize, PageToken: first.NextPageToken},
		{Query: `host="api"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &mismatchedPageSize, PageToken: first.NextPageToken},
	} {
		if _, err := service.Search(context.Background(), mismatched); !errors.Is(err, ErrInvalidGlobalAssetSearchPageToken) {
			t.Fatalf("expected bound token rejection for %+v, got %v", mismatched, err)
		}
	}
}

func TestGlobalAssetSearchPageTokenBindsHasScreenshotPolarity(t *testing.T) {
	now := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	websiteStore := &globalAssetSearchWebsiteStoreStub{items: []assetdomain.Website{{ID: 3, CreatedAt: now}, {ID: 2, CreatedAt: now}}}
	service := NewGlobalAssetSearchService(websiteStore, &globalAssetSearchEndpointStoreStub{}, &globalAssetSearchScreenshotStoreStub{})

	pageSize := 1
	first, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: `hasScreenshot=="true"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize})
	if err != nil {
		t.Fatalf("first search: %v", err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected next page token")
	}
	websiteStore.items = []assetdomain.Website{{ID: 2, CreatedAt: now}}
	second, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: `hasScreenshot=="true"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize, PageToken: first.NextPageToken})
	if err != nil {
		t.Fatalf("same-polarity token must stay valid: %v", err)
	}
	if len(second.Websites) != 1 || second.Websites[0].ID != 2 {
		t.Fatalf("token must page within the same boolean query, got %+v", second.Websites)
	}
	if _, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: `hasScreenshot=="false"`, AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &pageSize, PageToken: first.NextPageToken}); !errors.Is(err, ErrInvalidGlobalAssetSearchPageToken) {
		t.Fatalf("token issued for true must be rejected for false, got %v", err)
	}
}

func TestGlobalAssetSearchPageTokenRejectsPreContainsVersion(t *testing.T) {
	now := time.Date(2026, 8, 7, 8, 0, 0, 0, time.UTC)
	ast, err := ParseGlobalAssetSearchQuery("example")
	if err != nil {
		t.Fatalf("parse plain URL query: %v", err)
	}
	legacyPayload := globalAssetSearchPageToken{
		Version:     globalAssetSearchTokenVersion - 1,
		AssetType:   string(GlobalAssetSearchAssetTypeWebsite),
		QueryDigest: globalAssetSearchQueryDigest(ast),
		PageSize:    10,
		CreatedAt:   now.Format(time.RFC3339Nano),
		ID:          1,
	}
	encoded, err := json.Marshal(legacyPayload)
	if err != nil {
		t.Fatalf("marshal legacy page token: %v", err)
	}
	legacyToken := base64.RawURLEncoding.EncodeToString(encoded)
	if _, err := decodeAndValidateGlobalAssetSearchPageToken(legacyToken, GlobalAssetSearchAssetTypeWebsite, ast, 10); !errors.Is(err, ErrInvalidGlobalAssetSearchPageToken) {
		t.Fatalf("pre-contains page token must be rejected, got %v", err)
	}
}

func TestGlobalAssetSearchServiceFastFailsBeforeRepository(t *testing.T) {
	websiteStore := &globalAssetSearchWebsiteStoreStub{}
	endpointStore := &globalAssetSearchEndpointStoreStub{}
	service := NewGlobalAssetSearchService(websiteStore, endpointStore, &globalAssetSearchScreenshotStoreStub{})
	zero := 0
	overLimit := 101
	for _, input := range []GlobalAssetSearchInput{
		{Query: "", AssetType: GlobalAssetSearchAssetTypeWebsite},
		{Query: "valid", AssetType: "snapshot"},
		{Query: "valid", AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &zero},
		{Query: "valid", AssetType: GlobalAssetSearchAssetTypeWebsite, PageSize: &overLimit},
		{Query: "valid", AssetType: GlobalAssetSearchAssetTypeWebsite, PageToken: "not-a-token"},
	} {
		if _, err := service.Search(context.Background(), input); err == nil {
			t.Fatalf("expected failure for %+v", input)
		}
	}
	if websiteStore.calls != 0 || endpointStore.calls != 0 {
		t.Fatalf("invalid inputs must not reach repositories: website=%d endpoint=%d", websiteStore.calls, endpointStore.calls)
	}
}

func TestGlobalAssetSearchServiceMapsRepositoryTimeout(t *testing.T) {
	service := NewGlobalAssetSearchService(&globalAssetSearchWebsiteStoreStub{err: ErrGlobalAssetSearchTimeout}, &globalAssetSearchEndpointStoreStub{}, &globalAssetSearchScreenshotStoreStub{})
	if _, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: "example", AssetType: GlobalAssetSearchAssetTypeWebsite}); !errors.Is(err, ErrGlobalAssetSearchTimeout) {
		t.Fatalf("expected timeout sentinel, got %v", err)
	}
}

func TestGlobalAssetSearchServicePassesThroughCappedTotal(t *testing.T) {
	websiteStore := &globalAssetSearchWebsiteStoreStub{totalSize: GlobalAssetSearchTotalSizeCap, totalSizeCapped: true}
	endpointStore := &globalAssetSearchEndpointStoreStub{totalSize: 7}
	service := NewGlobalAssetSearchService(websiteStore, endpointStore, &globalAssetSearchScreenshotStoreStub{})

	capped, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: "example", AssetType: GlobalAssetSearchAssetTypeWebsite})
	if err != nil {
		t.Fatalf("website search: %v", err)
	}
	if capped.TotalSize != GlobalAssetSearchTotalSizeCap || !capped.TotalSizeCapped {
		t.Fatalf("capped total must pass through unchanged, got total=%d capped=%v", capped.TotalSize, capped.TotalSizeCapped)
	}

	exact, err := service.Search(context.Background(), GlobalAssetSearchInput{Query: "example", AssetType: GlobalAssetSearchAssetTypeEndpoint})
	if err != nil {
		t.Fatalf("endpoint search: %v", err)
	}
	if exact.TotalSize != 7 || exact.TotalSizeCapped {
		t.Fatalf("exact total must pass through unchanged, got total=%d capped=%v", exact.TotalSize, exact.TotalSizeCapped)
	}
}
