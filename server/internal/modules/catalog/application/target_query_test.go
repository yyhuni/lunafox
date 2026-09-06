package application

import (
	"context"
	"errors"
	"testing"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type targetQueryStoreStub struct {
	targetByID   map[int]*catalogdomain.Target
	listItems    []catalogdomain.Target
	listTotal    int64
	assetCounts  *catalogdomain.TargetAssetCounts
	vulnCounts   *catalogdomain.VulnerabilityCounts
	findByIDErr  error
	findAllErr   error
	assetErr     error
	vulnErr      error
	listPage     int
	listPageSize int
	listFilter   string
	listOrderBy  string
	listCalls    int
}

func (stub *targetQueryStoreStub) GetActiveByID(id int) (*catalogdomain.Target, error) {
	if stub.findByIDErr != nil {
		return nil, stub.findByIDErr
	}
	target, ok := stub.targetByID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copyTarget := *target
	return &copyTarget, nil
}

func (stub *targetQueryStoreStub) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error) {
	stub.listCalls++
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.findAllErr != nil {
		return nil, 0, stub.findAllErr
	}
	items := append([]catalogdomain.Target(nil), stub.listItems...)
	return items, stub.listTotal, nil
}

func (stub *targetQueryStoreStub) GetAssetCountsSummary(targetID int) (*catalogdomain.TargetAssetCounts, error) {
	_ = targetID
	if stub.assetErr != nil {
		return nil, stub.assetErr
	}
	if stub.assetCounts == nil {
		return &catalogdomain.TargetAssetCounts{}, nil
	}
	copyValue := *stub.assetCounts
	return &copyValue, nil
}

func (stub *targetQueryStoreStub) GetVulnerabilityCountsSummary(targetID int) (*catalogdomain.VulnerabilityCounts, error) {
	_ = targetID
	if stub.vulnErr != nil {
		return nil, stub.vulnErr
	}
	if stub.vulnCounts == nil {
		return &catalogdomain.VulnerabilityCounts{}, nil
	}
	copyValue := *stub.vulnCounts
	return &copyValue, nil
}

func TestTargetQueryServiceListTargets(t *testing.T) {
	store := &targetQueryStoreStub{
		listItems: []catalogdomain.Target{{ID: 1, Name: "example.com", Type: "domain"}},
		listTotal: 1,
	}
	service := NewTargetQueryService(store)

	result, err := service.ListTargets(context.Background(), TargetListQueryInput{
		PageSize: 20,
		Filter:   `type=="domain"`,
		OrderBy:  "createdAt desc",
	})
	if err != nil {
		t.Fatalf("list targets failed: %v", err)
	}
	if len(result.Targets) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result: len=%d total=%d", len(result.Targets), result.TotalSize)
	}
	if store.listPage != 1 || store.listPageSize != 20 || store.listFilter != `type=="domain"` || store.listOrderBy != "createdAt desc" {
		t.Fatalf("unexpected list args: page=%d size=%d filter=%q orderBy=%q", store.listPage, store.listPageSize, store.listFilter, store.listOrderBy)
	}
}

func TestTargetQueryServiceRejectsUnsupportedListQueryBeforeStore(t *testing.T) {
	store := &targetQueryStoreStub{}
	service := NewTargetQueryService(store)

	for _, input := range []TargetListQueryInput{
		{PageSize: 20, Filter: `type="domain"`},
		{PageSize: 20, Filter: `Type="domain"`},
		{PageSize: 20, Filter: `owner=="acme"`},
		{PageSize: 20, Filter: `type=="hostname"`},
		{PageSize: 20, Filter: `Type=="hostname"`},
		{PageSize: 20, OrderBy: "organization asc"},
		{PageSize: 20, OrderBy: "createdAt sideways"},
	} {
		if _, err := service.ListTargets(context.Background(), input); err == nil {
			t.Fatalf("expected invalid target list query for %+v", input)
		}
	}

	if store.listCalls != 0 {
		t.Fatalf("invalid query must fail before store access, got %d calls", store.listCalls)
	}
}

func TestTargetQueryServiceBindsPageTokenToQueryShape(t *testing.T) {
	store := &targetQueryStoreStub{
		listItems: []catalogdomain.Target{{ID: 1, Name: "a.example.com", Type: "domain"}},
		listTotal: 3,
	}
	service := NewTargetQueryService(store)

	first, err := service.ListTargets(context.Background(), TargetListQueryInput{
		PageSize: 1,
		Filter:   `displayName="example"`,
		OrderBy:  "displayName",
	})
	if err != nil {
		t.Fatalf("list first page failed: %v", err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected bound nextPageToken for additional target pages")
	}

	_, err = service.ListTargets(context.Background(), TargetListQueryInput{
		PageSize:  1,
		PageToken: first.NextPageToken,
		Filter:    `displayName="changed"`,
		OrderBy:   "displayName",
	})
	if !errors.Is(err, ErrInvalidTargetPageToken) {
		t.Fatalf("expected query-shape mismatch to reject pageToken, got %v", err)
	}

	second, err := service.ListTargets(context.Background(), TargetListQueryInput{
		PageSize:  1,
		PageToken: first.NextPageToken,
		Filter:    `displayName="example"`,
		OrderBy:   "displayName",
	})
	if err != nil {
		t.Fatalf("list second page failed: %v", err)
	}
	if second.Page != 2 || store.listPage != 2 {
		t.Fatalf("expected pageToken to resolve page 2, result page=%d store page=%d", second.Page, store.listPage)
	}
}

func TestTargetQueryServiceGetTargetDetailByID(t *testing.T) {
	store := &targetQueryStoreStub{
		targetByID: map[int]*catalogdomain.Target{7: {ID: 7, Name: "example.com", Type: "domain"}},
		assetCounts: &catalogdomain.TargetAssetCounts{
			Subdomains: 3,
			Websites:   2,
		},
		vulnCounts: &catalogdomain.VulnerabilityCounts{
			Total:    5,
			Critical: 1,
			High:     2,
			Medium:   1,
			Low:      1,
		},
	}
	service := NewTargetQueryService(store)

	target, summary, err := service.GetTargetDetailByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("get detail failed: %v", err)
	}
	if target.ID != 7 || summary.Subdomains != 3 || summary.Vulnerabilities.Total != 5 {
		t.Fatalf("unexpected detail: target=%+v summary=%+v", target, summary)
	}
}
