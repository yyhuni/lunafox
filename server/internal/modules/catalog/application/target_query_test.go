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

func (stub *targetQueryStoreStub) FindAll(page, pageSize int, targetType, filter string) ([]catalogdomain.Target, int64, error) {
	_ = targetType
	_ = filter
	stub.listPage = page
	stub.listPageSize = pageSize
	if stub.findAllErr != nil {
		return nil, 0, stub.findAllErr
	}
	items := append([]catalogdomain.Target(nil), stub.listItems...)
	return items, stub.listTotal, nil
}

func (stub *targetQueryStoreStub) GetAssetCounts(targetID int) (*catalogdomain.TargetAssetCounts, error) {
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

func (stub *targetQueryStoreStub) GetVulnerabilityCounts(targetID int) (*catalogdomain.VulnerabilityCounts, error) {
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

	items, total, err := service.ListTargets(context.Background(), 2, 20, "domain", "")
	if err != nil {
		t.Fatalf("list targets failed: %v", err)
	}
	if len(items) != 1 || total != 1 {
		t.Fatalf("unexpected list result: len=%d total=%d", len(items), total)
	}
	if store.listPage != 2 || store.listPageSize != 20 {
		t.Fatalf("unexpected pagination args: page=%d size=%d", store.listPage, store.listPageSize)
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
