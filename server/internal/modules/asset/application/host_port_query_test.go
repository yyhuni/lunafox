package application

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type hostPortQueryStoreStub struct {
	ipRows        []assetdomain.IPAggregationRow
	total         int64
	count         int64
	hostsByIP     map[string][]string
	portsByIP     map[string][]int
	listErr       error
	hostPortErr   error
	streamErr     error
	streamIPsErr  error
	countErr      error
	scannedErr    error
	listTargetID  int
	listPage      int
	listPageSize  int
	listFilter    string
	hostPortCalls []string
	streamID      int
	streamIPsID   int
	streamIPs     []string
	countID       int
}

func (stub *hostPortQueryStoreStub) GetIPAggregation(targetID int, page, pageSize int, filter string) ([]assetdomain.IPAggregationRow, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	rows := append([]assetdomain.IPAggregationRow(nil), stub.ipRows...)
	if page > 0 && pageSize > 0 {
		start := (page - 1) * pageSize
		if start >= len(rows) {
			return []assetdomain.IPAggregationRow{}, stub.total, nil
		}
		end := start + pageSize
		if end > len(rows) {
			end = len(rows)
		}
		rows = rows[start:end]
	}
	return rows, stub.total, nil
}

func (stub *hostPortQueryStoreStub) GetHostsAndPortsByIP(targetID int, ip string, filter string) ([]string, []int, error) {
	_ = targetID
	_ = filter
	stub.hostPortCalls = append(stub.hostPortCalls, ip)
	if stub.hostPortErr != nil {
		return nil, nil, stub.hostPortErr
	}
	return append([]string(nil), stub.hostsByIP[ip]...), append([]int(nil), stub.portsByIP[ip]...), nil
}

func (stub *hostPortQueryStoreStub) StreamByTargetID(targetID int) (*sql.Rows, error) {
	stub.streamID = targetID
	if stub.streamErr != nil {
		return nil, stub.streamErr
	}
	return nil, nil
}

func (stub *hostPortQueryStoreStub) StreamByTargetIDAndIPs(targetID int, ips []string) (*sql.Rows, error) {
	stub.streamIPsID = targetID
	stub.streamIPs = append([]string(nil), ips...)
	if stub.streamIPsErr != nil {
		return nil, stub.streamIPsErr
	}
	return nil, nil
}

func (stub *hostPortQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	stub.countID = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

func (stub *hostPortQueryStoreStub) ScanRow(rows *sql.Rows) (*assetdomain.HostPort, error) {
	if stub.scannedErr != nil {
		return nil, stub.scannedErr
	}
	return &assetdomain.HostPort{ID: 9}, nil
}

type hostPortTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *hostPortTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	target, ok := stub.targets[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *target
	return &copyTarget, nil
}

func TestHostPortQueryServiceListAndCount(t *testing.T) {
	now := time.Now()
	store := &hostPortQueryStoreStub{
		ipRows: []assetdomain.IPAggregationRow{
			{IP: "1.1.1.1", CreatedAt: now},
			{IP: "2.2.2.2", CreatedAt: now.Add(-time.Minute)},
		},
		total:     2,
		count:     2,
		hostsByIP: map[string][]string{"1.1.1.1": {"a.example.com"}},
		portsByIP: map[string][]int{"1.1.1.1": {80, 443}},
	}
	lookup := &hostPortTargetLookupStub{targets: map[int]*assetdomain.TargetRef{3: {ID: 3}}}
	service := NewHostPortQueryService(store, lookup)

	items, total, err := service.ListByTarget(context.Background(), 3, 1, 1, "ip:1.1.1.1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 2 || len(items) != 1 {
		t.Fatalf("unexpected list result total=%d len=%d", total, len(items))
	}
	if items[0].IP != "1.1.1.1" || len(items[0].Ports) != 2 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if store.listTargetID != 3 || store.listPage != 1 || store.listPageSize != 1 || store.listFilter != "ip:1.1.1.1" {
		t.Fatalf("unexpected list args: %+v", store)
	}

	count, err := service.CountByTarget(context.Background(), 3)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestHostPortQueryServiceTargetNotFound(t *testing.T) {
	service := NewHostPortQueryService(&hostPortQueryStoreStub{}, &hostPortTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, _, err := service.ListByTarget(context.Background(), 1, 1, 20, "")
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
