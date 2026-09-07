package application

import (
	"context"
	"errors"
	"testing"

	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
	"gorm.io/gorm"
)

type hostPortSnapshotQueryStoreStub struct {
	items      []snapshotdomain.HostPortSnapshot
	ipRows     []snapshotdomain.HostPortIPAggregationRow
	total      int64
	count      int64
	hostsByIP  map[string][]string
	portsByIP  map[string][]int
	listErr    error
	forEachErr error
	countErr   error
	visitErr   error
}

func (stub *hostPortSnapshotQueryStoreStub) GetIPAggregation(scanID int, page, pageSize int, filter, orderBy string) ([]snapshotdomain.HostPortIPAggregationRow, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	_ = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.HostPortIPAggregationRow(nil), stub.ipRows...), stub.total, nil
}

func (stub *hostPortSnapshotQueryStoreStub) GetHostsAndPortsByIP(scanID int, ip string, filter string) ([]string, []int, error) {
	_ = scanID
	_ = filter
	return append([]string(nil), stub.hostsByIP[ip]...), append([]int(nil), stub.portsByIP[ip]...), nil
}

func (stub *hostPortSnapshotQueryStoreStub) ListPortOptionsByScanID(scanID int) ([]snapshotdomain.FilterOption, error) {
	_ = scanID
	return []snapshotdomain.FilterOption{{Value: "443", Label: "443", Count: 2}}, nil
}

func (stub *hostPortSnapshotQueryStoreStub) ListByScanID(scanID int, page, pageSize int, filter string) ([]snapshotdomain.HostPortSnapshot, int64, error) {
	_ = scanID
	_ = page
	_ = pageSize
	_ = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]snapshotdomain.HostPortSnapshot(nil), stub.items...), stub.total, nil
}

func (stub *hostPortSnapshotQueryStoreStub) ForEachByScanID(ctx context.Context, scanID int, visit func(snapshotdomain.HostPortSnapshot) error) error {
	_ = ctx
	_ = scanID
	if stub.forEachErr != nil {
		return stub.forEachErr
	}
	for _, item := range stub.items {
		if stub.visitErr != nil {
			return stub.visitErr
		}
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *hostPortSnapshotQueryStoreStub) CountByScanID(scanID int) (int64, error) {
	_ = scanID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func TestHostPortSnapshotQueryServiceListAndCount(t *testing.T) {
	store := &hostPortSnapshotQueryStoreStub{
		items:     []snapshotdomain.HostPortSnapshot{{ID: 1}},
		ipRows:    []snapshotdomain.HostPortIPAggregationRow{{IP: "192.0.2.1"}},
		total:     1,
		count:     3,
		hostsByIP: map[string][]string{"192.0.2.1": {"a.example.com"}},
		portsByIP: map[string][]int{"192.0.2.1": {443}},
	}
	lookup := &snapshotScanLookupStub{scan: &snapshotdomain.ScanRef{ID: 2}}
	service := NewHostPortSnapshotQueryService(store, lookup)

	result, err := service.ListByScan(context.Background(), 2, HostPortSnapshotListQueryInput{PageSize: 20})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.HostPorts) != 1 || result.TotalSize != 1 || result.HostPorts[0].IP != "192.0.2.1" {
		t.Fatalf("unexpected list result: %+v", result)
	}

	count, err := service.CountByScan(context.Background(), 2)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestHostPortSnapshotQueryServiceScanNotFound(t *testing.T) {
	service := NewHostPortSnapshotQueryService(&hostPortSnapshotQueryStoreStub{}, &snapshotScanLookupStub{findErr: gorm.ErrRecordNotFound})

	_, err := service.ListByScan(context.Background(), 1, HostPortSnapshotListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrSnapshotScanNotFound) {
		t.Fatalf("expected ErrSnapshotScanNotFound, got %v", err)
	}
}
