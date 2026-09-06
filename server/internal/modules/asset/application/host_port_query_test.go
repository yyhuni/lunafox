package application

import (
	"context"
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
	listOrderBy   string
	hostPortCalls []string
	streamID      int
	streamIPsID   int
	streamIPs     []string
	countID       int
}

func (stub *hostPortQueryStoreStub) GetIPAggregation(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.IPAggregationRow, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
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

func (stub *hostPortQueryStoreStub) ListPortOptionsByTargetID(targetID int) ([]assetdomain.FilterOption, error) {
	stub.listTargetID = targetID
	return []assetdomain.FilterOption{{Value: "443", Label: "443", Count: 2}}, nil
}

func (stub *hostPortQueryStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.HostPort) error) error {
	_ = ctx
	stub.streamID = targetID
	if stub.streamErr != nil {
		return stub.streamErr
	}
	if stub.scannedErr != nil {
		return stub.scannedErr
	}
	for _, item := range stub.hostPortItems(nil) {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *hostPortQueryStoreStub) ForEachByTargetIDAndIPs(ctx context.Context, targetID int, ips []string, visit func(assetdomain.HostPort) error) error {
	_ = ctx
	stub.streamIPsID = targetID
	stub.streamIPs = append([]string(nil), ips...)
	if stub.streamIPsErr != nil {
		return stub.streamIPsErr
	}
	if stub.scannedErr != nil {
		return stub.scannedErr
	}
	for _, item := range stub.hostPortItems(ips) {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *hostPortQueryStoreStub) hostPortItems(ips []string) []assetdomain.HostPort {
	items := []assetdomain.HostPort{{ID: 9, TargetID: 1, IP: "1.1.1.1", Host: "a.example.com", Port: 443}}
	if len(ips) == 0 {
		return items
	}
	filtered := make([]assetdomain.HostPort, 0, len(items))
	allowed := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		allowed[ip] = struct{}{}
	}
	for _, item := range items {
		if _, ok := allowed[item.IP]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (stub *hostPortQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	stub.countID = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
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

	result, err := service.ListByTarget(context.Background(), 3, HostPortListQueryInput{PageSize: 1, Filter: `ip="1.1.1.1"`, OrderBy: "ip"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	items := result.HostPorts
	total := result.TotalSize
	if total != 2 || len(items) != 1 {
		t.Fatalf("unexpected list result total=%d len=%d", total, len(items))
	}
	if items[0].IP != "1.1.1.1" || len(items[0].Ports) != 2 {
		t.Fatalf("unexpected first item: %+v", items[0])
	}
	if store.listTargetID != 3 || store.listPage != 1 || store.listPageSize != 1 || store.listFilter != `ip="1.1.1.1"` || store.listOrderBy != "ip asc" {
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

func TestHostPortQueryServicePreservesExactHostScope(t *testing.T) {
	now := time.Now()
	store := &hostPortQueryStoreStub{
		ipRows:    []assetdomain.IPAggregationRow{{IP: "1.1.1.1", CreatedAt: now}},
		total:     1,
		hostsByIP: map[string][]string{"1.1.1.1": {"api.acme.com"}},
		portsByIP: map[string][]int{"1.1.1.1": {443}},
	}
	service := NewHostPortQueryService(store, &hostPortTargetLookupStub{targets: map[int]*assetdomain.TargetRef{3: {ID: 3}}})

	filter := `host=="API.Acme.COM"`
	if _, err := service.ListByTarget(context.Background(), 3, HostPortListQueryInput{PageSize: 20, Filter: filter}); err != nil {
		t.Fatalf("ListByTarget returned error: %v", err)
	}
	if store.listFilter != filter {
		t.Fatalf("expected exact host filter to reach store, got %q", store.listFilter)
	}

	if _, err := service.ListByTarget(context.Background(), 3, HostPortListQueryInput{PageSize: 20, Filter: `host="api"`}); err != nil {
		t.Fatalf("ordinary host contains filter must remain supported: %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 3, HostPortListQueryInput{PageSize: 20, Filter: `host!="api.acme.com"`}); !errors.Is(err, ErrUnsupportedHostPortFilter) {
		t.Fatalf("expected unsupported host operator, got %v", err)
	}
}

func TestHostPortQueryServicePortOptions(t *testing.T) {
	store := &hostPortQueryStoreStub{}
	lookup := &hostPortTargetLookupStub{targets: map[int]*assetdomain.TargetRef{4: {ID: 4}}}
	service := NewHostPortQueryService(store, lookup)

	options, err := service.ListPortOptionsByTarget(context.Background(), 4)
	if err != nil {
		t.Fatalf("port options failed: %v", err)
	}
	if store.listTargetID != 4 || len(options) != 1 || options[0].Value != "443" {
		t.Fatalf("unexpected port options target=%d options=%+v", store.listTargetID, options)
	}

	lookup.err = gorm.ErrRecordNotFound
	if _, err := service.ListPortOptionsByTarget(context.Background(), 4); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected target not found, got %v", err)
	}
}

func TestHostPortQueryServiceTargetNotFound(t *testing.T) {
	service := NewHostPortQueryService(&hostPortQueryStoreStub{}, &hostPortTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.ListByTarget(context.Background(), 1, HostPortListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}
