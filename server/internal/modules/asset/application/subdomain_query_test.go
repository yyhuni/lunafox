package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type subdomainQueryStoreStub struct {
	items        []assetdomain.Subdomain
	total        int64
	count        int64
	listErr      error
	forEachErr   error
	countErr     error
	listTargetID int
	listPage     int
	listPageSize int
	listFilter   string
	listOrderBy  string
	forEachID    int
	countID      int
}

func (stub *subdomainQueryStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Subdomain, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Subdomain(nil), stub.items...), stub.total, nil
}

func (stub *subdomainQueryStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Subdomain) error) error {
	_ = ctx
	stub.forEachID = targetID
	if stub.forEachErr != nil {
		return stub.forEachErr
	}
	for _, item := range stub.items {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *subdomainQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	stub.countID = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

type subdomainQueryTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func TestSubdomainQueryServiceForEachByTarget(t *testing.T) {
	store := &subdomainQueryStoreStub{items: []assetdomain.Subdomain{{ID: 1}, {ID: 2}}}
	lookup := &subdomainQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewSubdomainQueryService(store, lookup)

	var ids []int
	err := service.ForEachByTarget(context.Background(), 7, func(item assetdomain.Subdomain) error {
		ids = append(ids, item.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if store.forEachID != 7 {
		t.Fatalf("expected target 7, got %d", store.forEachID)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("unexpected streamed ids: %v", ids)
	}
}

func (stub *subdomainQueryTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func (stub *subdomainQueryTargetLookupStub) GetActiveByIDContext(ctx context.Context, id int) (*assetdomain.TargetRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetActiveByID(id)
}

func TestSubdomainQueryServiceListAndCount(t *testing.T) {
	store := &subdomainQueryStoreStub{
		items: []assetdomain.Subdomain{{ID: 1}, {ID: 2}},
		total: 2,
		count: 8,
	}
	lookup := &subdomainQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewSubdomainQueryService(store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, SubdomainListQueryInput{
		PageSize: 20,
		Filter:   `dnsName="api"`,
		OrderBy:  "dnsName desc",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Subdomains) != 2 || result.TotalSize != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Subdomains), result.TotalSize)
	}
	if store.listTargetID != 7 || store.listPage != 1 || store.listPageSize != 20 || store.listFilter != `dnsName="api"` || store.listOrderBy != "dnsName desc" {
		t.Fatalf("unexpected list args: %+v", store)
	}

	count, err := service.CountByTarget(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 8 {
		t.Fatalf("expected count 8, got %d", count)
	}
}

func TestSubdomainQueryServiceTargetNotFound(t *testing.T) {
	service := NewSubdomainQueryService(&subdomainQueryStoreStub{}, &subdomainQueryTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.ListByTarget(context.Background(), 1, SubdomainListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestSubdomainQueryServiceRejectsUnsupportedFilterOrderAndMismatchedToken(t *testing.T) {
	store := &subdomainQueryStoreStub{}
	lookup := &subdomainQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewSubdomainQueryService(store, lookup)

	if _, err := service.ListByTarget(context.Background(), 7, SubdomainListQueryInput{Filter: `name="api"`}); !errors.Is(err, ErrUnsupportedSubdomainFilter) {
		t.Fatalf("expected unsupported filter, got %v", err)
	}
	if store.listTargetID != 0 {
		t.Fatal("unsupported filter must fail before store access")
	}

	if _, err := service.ListByTarget(context.Background(), 7, SubdomainListQueryInput{OrderBy: "targetCount desc"}); !errors.Is(err, ErrUnsupportedSubdomainOrderBy) {
		t.Fatalf("expected unsupported orderBy, got %v", err)
	}

	token, err := encodeSubdomainListPageToken(subdomainListPageTokenPayload{
		Version:  subdomainListTokenVersion,
		Page:     2,
		PageSize: 10,
		TargetID: 8,
		Filter:   `dnsName="api"`,
		OrderBy:  "createdAt desc",
	})
	if err != nil {
		t.Fatalf("encode token: %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, SubdomainListQueryInput{PageSize: 10, PageToken: token, Filter: `dnsName="api"`}); !errors.Is(err, ErrInvalidSubdomainPageToken) {
		t.Fatalf("expected target-bound token mismatch, got %v", err)
	}
}
