package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type directoryFacadeStoreStub struct {
	*directoryQueryStoreStub
	*directoryCommandStoreStub
}

type endpointFacadeStoreStub struct {
	*endpointQueryStoreStub
	*endpointCommandStoreStub
}

func (stub *endpointFacadeStoreStub) GetByID(id int) (*assetdomain.Endpoint, error) {
	if stub.endpointCommandStoreStub != nil {
		return stub.endpointCommandStoreStub.GetByID(id)
	}
	return stub.endpointQueryStoreStub.GetByID(id)
}

type websiteFacadeStoreStub struct {
	*websiteQueryStoreStub
	*websiteCommandStoreStub
}

func (stub *websiteFacadeStoreStub) GetByID(id int) (*assetdomain.Website, error) {
	if stub.websiteCommandStoreStub != nil {
		return stub.websiteCommandStoreStub.GetByID(id)
	}
	return stub.websiteQueryStoreStub.GetByID(id)
}

type subdomainFacadeStoreStub struct {
	*subdomainQueryStoreStub
	*subdomainCommandStoreStub
}

type hostPortFacadeStoreStub struct {
	*hostPortQueryStoreStub
	*hostPortCommandStoreStub
}

type screenshotFacadeStoreStub struct {
	*screenshotQueryStoreStub
	*screenshotCommandStoreStub
}

func TestDirectoryQueryServiceForEachAndCount(t *testing.T) {
	store := &directoryQueryStoreStub{items: []assetdomain.Directory{{ID: 11}}, count: 6}
	lookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{9: {ID: 9}}}
	service := NewDirectoryQueryService(store, lookup)

	var streamed []assetdomain.Directory
	if err := service.ForEachByTarget(context.Background(), 9, func(item assetdomain.Directory) error {
		streamed = append(streamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if store.streamID != 9 {
		t.Fatalf("expected for-each target id 9, got %d", store.streamID)
	}
	if len(streamed) != 1 || streamed[0].ID != 11 {
		t.Fatalf("unexpected streamed items: %+v", streamed)
	}

	count, err := service.CountByTarget(context.Background(), 9)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 6 {
		t.Fatalf("expected count 6, got %d", count)
	}

	lookup.err = gorm.ErrRecordNotFound
	if err := service.ForEachByTarget(context.Background(), 9, func(assetdomain.Directory) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
	if _, err := service.CountByTarget(context.Background(), 9); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from count, got %v", err)
	}

	lookup.err = errors.New("lookup failed")
	if err := service.ForEachByTarget(context.Background(), 9, func(assetdomain.Directory) error { return nil }); err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestEndpointQueryServiceForEachAndCount(t *testing.T) {
	store := &endpointQueryStoreStub{items: []assetdomain.Endpoint{{ID: 10}}, count: 7}
	lookup := &endpointQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{5: {ID: 5}}}
	service := NewEndpointQueryService(store, lookup)

	var streamed []assetdomain.Endpoint
	if err := service.ForEachByTarget(context.Background(), 5, func(item assetdomain.Endpoint) error {
		streamed = append(streamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if len(streamed) != 1 || streamed[0].ID != 10 {
		t.Fatalf("unexpected streamed items: %+v", streamed)
	}

	count, err := service.CountByTarget(context.Background(), 5)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 7 {
		t.Fatalf("expected count 7, got %d", count)
	}

	lookup.err = gorm.ErrRecordNotFound
	if err := service.ForEachByTarget(context.Background(), 5, func(assetdomain.Endpoint) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}

	lookup.err = errors.New("lookup failed")
	if _, err := service.CountByTarget(context.Background(), 5); err == nil || err.Error() != "lookup failed" {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func TestWebsiteQueryServiceForEachAndCount(t *testing.T) {
	store := &websiteQueryStoreStub{items: []assetdomain.Website{{ID: 13}}, count: 4}
	lookup := &websiteTargetLookupQueryStub{target: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewWebsiteQueryService(store, store, lookup)

	var streamed []assetdomain.Website
	if err := service.ForEachByTarget(context.Background(), 7, func(item assetdomain.Website) error {
		streamed = append(streamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if len(streamed) != 1 || streamed[0].ID != 13 {
		t.Fatalf("unexpected streamed items: %+v", streamed)
	}

	count, err := service.CountByTarget(context.Background(), 7)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}

	lookup.err = gorm.ErrRecordNotFound
	if _, err := service.CountByTarget(context.Background(), 7); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestSubdomainQueryServiceForEachAndCount(t *testing.T) {
	store := &subdomainQueryStoreStub{items: []assetdomain.Subdomain{{ID: 12}}, count: 9}
	lookup := &subdomainQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{11: {ID: 11}}}
	service := NewSubdomainQueryService(store, lookup)

	var streamed []assetdomain.Subdomain
	if err := service.ForEachByTarget(context.Background(), 11, func(item assetdomain.Subdomain) error {
		streamed = append(streamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if store.forEachID != 11 {
		t.Fatalf("expected for-each target id 11, got %d", store.forEachID)
	}
	if len(streamed) != 1 || streamed[0].ID != 12 {
		t.Fatalf("unexpected streamed items: %+v", streamed)
	}

	count, err := service.CountByTarget(context.Background(), 11)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 9 {
		t.Fatalf("expected count 9, got %d", count)
	}

	lookup.err = gorm.ErrRecordNotFound
	if err := service.ForEachByTarget(context.Background(), 11, func(assetdomain.Subdomain) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestHostPortQueryServiceForEachCountAndIPFilter(t *testing.T) {
	store := &hostPortQueryStoreStub{count: 3}
	lookup := &hostPortTargetLookupStub{targets: map[int]*assetdomain.TargetRef{4: {ID: 4}}}
	service := NewHostPortQueryService(store, lookup)

	var streamed []assetdomain.HostPort
	if err := service.ForEachByTarget(context.Background(), 4, func(item assetdomain.HostPort) error {
		streamed = append(streamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if store.streamID != 4 {
		t.Fatalf("expected for-each target id 4, got %d", store.streamID)
	}
	if len(streamed) != 1 || streamed[0].ID != 9 {
		t.Fatalf("unexpected streamed items: %+v", streamed)
	}

	streamed = nil
	if err := service.ForEachByTargetAndIPs(context.Background(), 4, []string{"1.1.1.1"}, func(item assetdomain.HostPort) error {
		streamed = append(streamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each by ips failed: %v", err)
	}
	if store.streamIPsID != 4 || len(store.streamIPs) != 1 || store.streamIPs[0] != "1.1.1.1" {
		t.Fatalf("unexpected for-each by ips args: %+v", store)
	}

	count, err := service.CountByTarget(context.Background(), 4)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	lookup.err = gorm.ErrRecordNotFound
	if err := service.ForEachByTargetAndIPs(context.Background(), 4, []string{"1.1.1.1"}, func(assetdomain.HostPort) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestCommandServicesEmptyBatchDeleteAndResolveTarget(t *testing.T) {
	t.Run("directory batch delete empty", func(t *testing.T) {
		service := NewDirectoryCommandService(&directoryCommandStoreStub{}, &directoryTargetLookupStub{})
		deleted, err := service.BatchDelete(context.Background(), nil)
		if err != nil {
			t.Fatalf("batch delete failed: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("expected deleted count 0, got %d", deleted)
		}
	})

	t.Run("subdomain batch delete empty", func(t *testing.T) {
		service := NewSubdomainCommandService(&subdomainCommandStoreStub{}, &subdomainTargetLookupStub{})
		deleted, err := service.BatchDelete(context.Background(), nil)
		if err != nil {
			t.Fatalf("batch delete failed: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("expected deleted count 0, got %d", deleted)
		}
	})

	t.Run("endpoint batch delete coverage", func(t *testing.T) {
		store := &endpointCommandStoreStub{}
		service := NewEndpointCommandService(store, &endpointTargetLookupStub{})

		deleted, err := service.BatchDelete(context.Background(), nil)
		if err != nil {
			t.Fatalf("empty batch delete failed: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("expected deleted count 0, got %d", deleted)
		}

		deleted, err = service.BatchDelete(context.Background(), []int{1, 2})
		if err != nil {
			t.Fatalf("batch delete failed: %v", err)
		}
		if deleted != 2 || len(store.batchDeletedIDs) != 2 {
			t.Fatalf("unexpected delete result deleted=%d ids=%v", deleted, store.batchDeletedIDs)
		}

		store.batchDeleteErr = errors.New("delete failed")
		if _, err := service.BatchDelete(context.Background(), []int{1}); err == nil || err.Error() != "delete failed" {
			t.Fatalf("expected delete error, got %v", err)
		}
	})

	t.Run("website batch delete and resolve target", func(t *testing.T) {
		store := &websiteCommandStoreStub{}
		lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{8: {ID: 8}}}
		service := NewWebsiteCommandService(store, lookup)

		deleted, err := service.BatchDelete(context.Background(), nil)
		if err != nil {
			t.Fatalf("empty batch delete failed: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("expected deleted count 0, got %d", deleted)
		}

		deleted, err = service.BatchDelete(context.Background(), []int{2, 3})
		if err != nil {
			t.Fatalf("batch delete failed: %v", err)
		}
		if deleted != 2 || len(store.batchDeletedID) != 2 {
			t.Fatalf("unexpected delete result deleted=%d ids=%v", deleted, store.batchDeletedID)
		}

		target, err := service.ResolveTarget(context.Background(), 8)
		if err != nil {
			t.Fatalf("resolve target failed: %v", err)
		}
		if target.ID != 8 {
			t.Fatalf("unexpected target: %+v", target)
		}

		lookup.err = gorm.ErrRecordNotFound
		if _, err := service.ResolveTarget(context.Background(), 8); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound, got %v", err)
		}
	})

	t.Run("screenshot batch delete empty", func(t *testing.T) {
		service := NewScreenshotCommandService(&screenshotCommandStoreStub{}, &screenshotTargetLookupStub{})
		deleted, err := service.BatchDelete(context.Background(), nil)
		if err != nil {
			t.Fatalf("batch delete failed: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("expected deleted count 0, got %d", deleted)
		}
	})
}

func TestDirectoryFacadeCoverage(t *testing.T) {
	store := &directoryFacadeStoreStub{
		directoryQueryStoreStub:   &directoryQueryStoreStub{items: []assetdomain.Directory{{ID: 1}}, total: 1, count: 2},
		directoryCommandStoreStub: &directoryCommandStoreStub{},
	}
	lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	facade := NewDirectoryFacade(NewDirectoryQueryService(store, lookup), NewDirectoryCommandService(store, lookup))

	if facade == nil || facade.queryService == nil || facade.cmdService == nil {
		t.Fatal("expected facade services to be initialized")
	}

	result, err := facade.ListByTarget(1, DirectoryListQueryInput{PageSize: 20})
	if err != nil || len(result.Directories) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result items=%v total=%d err=%v", result.Directories, result.TotalSize, err)
	}

	created, err := facade.BatchCreate(1, []string{"https://example.com/a"})
	if err != nil || created != 1 {
		t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
	}

	deleted, err := facade.BatchDelete([]int{1, 2})
	if err != nil || deleted != 2 {
		t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
	}

	var directoryStreamed []assetdomain.Directory
	if err := facade.ForEachByTarget(1, func(item assetdomain.Directory) error {
		directoryStreamed = append(directoryStreamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if len(directoryStreamed) != 1 || directoryStreamed[0].ID != 1 {
		t.Fatalf("unexpected streamed directories: %+v", directoryStreamed)
	}

	count, err := facade.CountByTarget(1)
	if err != nil || count != 2 {
		t.Fatalf("unexpected count result count=%d err=%v", count, err)
	}

	affected, err := facade.BatchUpsert(1, []DirectoryUpsertItem{{URL: "https://example.com/b"}})
	if err != nil || affected != 1 {
		t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
	}

	lookup.err = gorm.ErrRecordNotFound
	if _, err := facade.ListByTarget(1, DirectoryListQueryInput{PageSize: 20}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from list, got %v", err)
	}
	if _, err := facade.BatchCreate(1, []string{"https://example.com/a"}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from batch create, got %v", err)
	}
	if err := facade.ForEachByTarget(1, func(Directory) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from for-each, got %v", err)
	}
	if _, err := facade.CountByTarget(1); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from count, got %v", err)
	}
	if _, err := facade.BatchUpsert(1, []DirectoryUpsertItem{{URL: "https://example.com/b"}}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from batch upsert, got %v", err)
	}

	store.directoryQueryStoreStub.scannedErr = errors.New("scan failed")
	lookup.err = nil
	if err := facade.ForEachByTarget(1, func(Directory) error { return nil }); err == nil || err.Error() != "scan failed" {
		t.Fatalf("expected for-each error, got %v", err)
	}
}

func TestEndpointFacadeCoverage(t *testing.T) {
	store := &endpointFacadeStoreStub{
		endpointQueryStoreStub:   &endpointQueryStoreStub{items: []assetdomain.Endpoint{{ID: 1}}, total: 1, count: 2, itemByID: map[int]*assetdomain.Endpoint{1: {ID: 1}}},
		endpointCommandStoreStub: &endpointCommandStoreStub{endpointByID: map[int]*assetdomain.Endpoint{1: {ID: 1}}},
	}
	lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	facade := NewEndpointFacade(NewEndpointQueryService(store, lookup), NewEndpointCommandService(store, lookup))

	if facade == nil {
		t.Fatal("expected facade")
	}

	result, err := facade.ListByTarget(1, EndpointListQueryInput{PageSize: 20})
	if err != nil || len(result.Endpoints) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result result=%+v err=%v", result, err)
	}
	item, err := facade.GetByID(1)
	if err != nil || item.ID != 1 {
		t.Fatalf("unexpected get result item=%+v err=%v", item, err)
	}
	created, err := facade.BatchCreate(1, []string{"https://example.com/a"})
	if err != nil || created != 1 {
		t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
	}
	if err := facade.Delete(1); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	deleted, err := facade.BatchDelete([]int{1, 2})
	if err != nil || deleted != 2 {
		t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
	}
	var endpointStreamed []assetdomain.Endpoint
	if err := facade.ForEachByTarget(1, func(item assetdomain.Endpoint) error {
		endpointStreamed = append(endpointStreamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if len(endpointStreamed) != 1 || endpointStreamed[0].ID != 1 {
		t.Fatalf("unexpected streamed endpoints: %+v", endpointStreamed)
	}
	count, err := facade.CountByTarget(1)
	if err != nil || count != 2 {
		t.Fatalf("unexpected count result count=%d err=%v", count, err)
	}
	affected, err := facade.BatchUpsert(1, []EndpointUpsertItem{{URL: "https://example.com/b", Host: "example.com"}})
	if err != nil || affected != 1 {
		t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
	}

	store.endpointQueryStoreStub.itemByID = map[int]*assetdomain.Endpoint{}
	if _, err := facade.GetByID(404); !errors.Is(err, ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound from get, got %v", err)
	}

	store.endpointCommandStoreStub.endpointByID = map[int]*assetdomain.Endpoint{}
	if err := facade.Delete(404); !errors.Is(err, ErrEndpointNotFound) {
		t.Fatalf("expected ErrEndpointNotFound from delete, got %v", err)
	}

	lookup.err = gorm.ErrRecordNotFound
	if _, err := facade.ListByTarget(1, EndpointListQueryInput{PageSize: 20}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from list, got %v", err)
	}
	if _, err := facade.BatchCreate(1, []string{"https://example.com/a"}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from batch create, got %v", err)
	}
	if err := facade.ForEachByTarget(1, func(Endpoint) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from for-each, got %v", err)
	}
	if _, err := facade.CountByTarget(1); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from count, got %v", err)
	}
	if _, err := facade.BatchUpsert(1, []EndpointUpsertItem{{URL: "https://example.com/b", Host: "example.com"}}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from batch upsert, got %v", err)
	}

	store.endpointQueryStoreStub.scannedErr = errors.New("scan failed")
	lookup.err = nil
	if err := facade.ForEachByTarget(1, func(Endpoint) error { return nil }); err == nil || err.Error() != "scan failed" {
		t.Fatalf("expected for-each error, got %v", err)
	}
}

func TestWebsiteFacadeCoverage(t *testing.T) {
	store := &websiteFacadeStoreStub{
		websiteQueryStoreStub:   &websiteQueryStoreStub{items: []assetdomain.Website{{ID: 1}}, total: 1, count: 3},
		websiteCommandStoreStub: &websiteCommandStoreStub{websiteByID: map[int]*assetdomain.Website{1: {ID: 1}}},
	}
	lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
	facade := NewWebsiteFacade(NewWebsiteQueryService(store, store, lookup), NewWebsiteCommandService(store, lookup))

	result, err := facade.ListByTarget(1, WebsiteListQueryInput{PageSize: 20})
	if err != nil || len(result.Websites) != 1 || result.TotalSize != 1 {
		t.Fatalf("unexpected list result items=%v total=%d err=%v", result.Websites, result.TotalSize, err)
	}
	created, err := facade.BatchCreate(1, []string{"https://example.com"})
	if err != nil || created != 1 {
		t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
	}
	if err := facade.Delete(1); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	deleted, err := facade.BatchDelete([]int{1, 2})
	if err != nil || deleted != 2 {
		t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
	}
	var websiteStreamed []assetdomain.Website
	if err := facade.ForEachByTarget(1, func(item assetdomain.Website) error {
		websiteStreamed = append(websiteStreamed, item)
		return nil
	}); err != nil {
		t.Fatalf("for-each failed: %v", err)
	}
	if len(websiteStreamed) != 1 || websiteStreamed[0].ID != 1 {
		t.Fatalf("unexpected streamed websites: %+v", websiteStreamed)
	}
	count, err := facade.CountByTarget(1)
	if err != nil || count != 3 {
		t.Fatalf("unexpected count result count=%d err=%v", count, err)
	}
	affected, err := facade.BatchUpsert(1, []WebsiteUpsertItem{{URL: "https://example.com/a", Host: "example.com"}})
	if err != nil || affected != 1 {
		t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
	}

	store.websiteCommandStoreStub.websiteByID = map[int]*assetdomain.Website{}
	if err := facade.Delete(404); !errors.Is(err, ErrWebsiteNotFound) {
		t.Fatalf("expected ErrWebsiteNotFound, got %v", err)
	}

	lookup.err = gorm.ErrRecordNotFound
	if _, err := facade.BatchCreate(1, []string{"https://example.com"}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from batch create, got %v", err)
	}
	if err := facade.ForEachByTarget(1, func(Website) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from for-each, got %v", err)
	}
	if _, err := facade.CountByTarget(1); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from count, got %v", err)
	}
	if _, err := facade.BatchUpsert(1, []WebsiteUpsertItem{{URL: "https://example.com/a", Host: "example.com"}}); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound from batch upsert, got %v", err)
	}
}

func TestSubdomainHostPortAndScreenshotFacadeCoverage(t *testing.T) {
	t.Run("subdomain", func(t *testing.T) {
		store := &subdomainFacadeStoreStub{
			subdomainQueryStoreStub:   &subdomainQueryStoreStub{items: []assetdomain.Subdomain{{ID: 1}}, total: 1, count: 2},
			subdomainCommandStoreStub: &subdomainCommandStoreStub{},
		}
		lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		facade := NewSubdomainFacade(NewSubdomainQueryService(store, lookup), NewSubdomainCommandService(store, lookup))

		result, err := facade.ListByTarget(1, SubdomainListQueryInput{PageSize: 20})
		if err != nil || len(result.Subdomains) != 1 || result.TotalSize != 1 {
			t.Fatalf("unexpected list result result=%+v err=%v", result, err)
		}
		created, err := facade.BatchCreate(1, []string{"api.example.com"})
		if err != nil || created != 1 {
			t.Fatalf("unexpected batch create result created=%d err=%v", created, err)
		}
		deleted, err := facade.BatchDelete([]int{1})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}
		var subdomainStreamed []assetdomain.Subdomain
		if err := facade.ForEachByTarget(1, func(item assetdomain.Subdomain) error {
			subdomainStreamed = append(subdomainStreamed, item)
			return nil
		}); err != nil {
			t.Fatalf("for-each failed: %v", err)
		}
		if len(subdomainStreamed) != 1 || subdomainStreamed[0].ID != 1 {
			t.Fatalf("unexpected streamed subdomains: %+v", subdomainStreamed)
		}
		count, err := facade.CountByTarget(1)
		if err != nil || count != 2 {
			t.Fatalf("unexpected count result count=%d err=%v", count, err)
		}
		lookup.targets[1].Type = assetdomain.TargetTypeIP
		if _, err := facade.BatchCreate(1, []string{"api.example.com"}); !errors.Is(err, ErrInvalidTargetType) {
			t.Fatalf("expected ErrInvalidTargetType, got %v", err)
		}

		lookup.err = gorm.ErrRecordNotFound
		if _, err := facade.ListByTarget(1, SubdomainListQueryInput{PageSize: 20}); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from list, got %v", err)
		}
		if err := facade.ForEachByTarget(1, func(Subdomain) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from for-each, got %v", err)
		}
		if _, err := facade.CountByTarget(1); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from count, got %v", err)
		}
	})

	t.Run("host port", func(t *testing.T) {
		store := &hostPortFacadeStoreStub{
			hostPortQueryStoreStub:   &hostPortQueryStoreStub{ipRows: []assetdomain.IPAggregationRow{{IP: "1.1.1.1"}}, total: 1, count: 1, hostsByIP: map[string][]string{"1.1.1.1": {"a.example.com"}}, portsByIP: map[string][]int{"1.1.1.1": {443}}},
			hostPortCommandStoreStub: &hostPortCommandStoreStub{},
		}
		lookup := &hostPortCommandTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
		facade := NewHostPortFacade(NewHostPortQueryService(store, lookup), NewHostPortCommandService(store, lookup))

		result, err := facade.ListByTarget(1, HostPortListQueryInput{PageSize: 20})
		if err != nil || len(result.HostPorts) != 1 || result.TotalSize != 1 {
			t.Fatalf("unexpected list result result=%v err=%v", result, err)
		}
		var hostPortStreamed []assetdomain.HostPort
		if err := facade.ForEachByTarget(1, func(item assetdomain.HostPort) error {
			hostPortStreamed = append(hostPortStreamed, item)
			return nil
		}); err != nil {
			t.Fatalf("for-each failed: %v", err)
		}
		if len(hostPortStreamed) != 1 || hostPortStreamed[0].ID != 9 {
			t.Fatalf("unexpected streamed host ports: %+v", hostPortStreamed)
		}
		hostPortStreamed = nil
		if err := facade.ForEachByTargetAndIPs(1, []string{"1.1.1.1"}, func(item assetdomain.HostPort) error {
			hostPortStreamed = append(hostPortStreamed, item)
			return nil
		}); err != nil {
			t.Fatalf("for-each by ips failed: %v", err)
		}
		if len(hostPortStreamed) != 1 || hostPortStreamed[0].IP != "1.1.1.1" {
			t.Fatalf("unexpected filtered host ports: %+v", hostPortStreamed)
		}
		count, err := facade.CountByTarget(1)
		if err != nil || count != 1 {
			t.Fatalf("unexpected count result count=%d err=%v", count, err)
		}
		affected, err := facade.BatchUpsert(1, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}})
		if err != nil || affected != 1 {
			t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
		}
		deleted, err := facade.BatchDeleteByIPs([]string{"1.1.1.1"})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}

		lookup.err = gorm.ErrRecordNotFound
		if _, err := facade.ListByTarget(1, HostPortListQueryInput{PageSize: 20}); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from list, got %v", err)
		}
		if err := facade.ForEachByTarget(1, func(HostPort) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from for-each, got %v", err)
		}
		if err := facade.ForEachByTargetAndIPs(1, []string{"1.1.1.1"}, func(HostPort) error { return nil }); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from for-each by ips, got %v", err)
		}
		if _, err := facade.CountByTarget(1); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from count, got %v", err)
		}
		if _, err := facade.BatchUpsert(1, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}}); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from batch upsert, got %v", err)
		}
	})

	t.Run("screenshot", func(t *testing.T) {
		status := int16(200)
		store := &screenshotFacadeStoreStub{
			screenshotQueryStoreStub:   &screenshotQueryStoreStub{items: []assetdomain.Screenshot{{ID: 1}}, total: 1, itemByID: map[int]*assetdomain.Screenshot{1: {ID: 1}}},
			screenshotCommandStoreStub: &screenshotCommandStoreStub{},
		}
		lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		facade := NewScreenshotFacade(NewScreenshotQueryService(store, lookup), NewScreenshotCommandService(store, lookup))

		result, err := facade.ListByTarget(1, ScreenshotListQueryInput{PageSize: 20})
		if err != nil || len(result.Screenshots) != 1 || result.TotalSize != 1 {
			t.Fatalf("unexpected list result result=%+v err=%v", result, err)
		}
		item, err := facade.GetByID(1)
		if err != nil || item.ID != 1 {
			t.Fatalf("unexpected get result item=%+v err=%v", item, err)
		}
		deleted, err := facade.BatchDelete([]int{1})
		if err != nil || deleted != 1 {
			t.Fatalf("unexpected batch delete result deleted=%d err=%v", deleted, err)
		}
		affected, err := facade.BatchUpsert(1, &BatchUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://example.com", StatusCode: &status}}})
		if err != nil || affected != 1 {
			t.Fatalf("unexpected batch upsert result affected=%d err=%v", affected, err)
		}

		store.screenshotQueryStoreStub.itemByID = map[int]*assetdomain.Screenshot{}
		if _, err := facade.GetByID(404); !errors.Is(err, ErrScreenshotNotFound) {
			t.Fatalf("expected ErrScreenshotNotFound, got %v", err)
		}

		lookup.err = gorm.ErrRecordNotFound
		if _, err := facade.ListByTarget(1, ScreenshotListQueryInput{PageSize: 20}); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from list, got %v", err)
		}
		if _, err := facade.BatchUpsert(1, &BatchUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://example.com", StatusCode: &status}}}); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected ErrTargetNotFound from batch upsert, got %v", err)
		}
	})
}

func TestApplicationGenericErrorPassThrough(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("services", func(t *testing.T) {
		directoryStore := &directoryQueryStoreStub{listErr: errBoom}
		directoryLookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
		if _, err := NewDirectoryQueryService(directoryStore, directoryLookup).ListByTarget(context.Background(), 1, DirectoryListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected directory query error, got %v", err)
		}

		endpointStore := &endpointCommandStoreStub{findByIDErr: errBoom}
		if err := NewEndpointCommandService(endpointStore, &endpointTargetLookupStub{}).Delete(context.Background(), 1); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint delete error, got %v", err)
		}

		websiteLookup := &websiteTargetLookupStub{err: errBoom}
		if _, err := NewWebsiteCommandService(&websiteCommandStoreStub{}, websiteLookup).ResolveTarget(context.Background(), 1); !errors.Is(err, errBoom) {
			t.Fatalf("expected website resolve error, got %v", err)
		}

		hostPortStore := &hostPortCommandStoreStub{deleteErr: errBoom}
		if _, err := NewHostPortCommandService(hostPortStore, &hostPortCommandTargetLookupStub{}).BatchDeleteByIPs(context.Background(), []string{"1.1.1.1"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected host-port delete error, got %v", err)
		}

		screenshotStore := &screenshotCommandStoreStub{upsertErr: errBoom}
		req := &BatchUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://example.com"}}}
		if _, err := NewScreenshotCommandService(screenshotStore, &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}).BatchUpsert(context.Background(), 1, req); !errors.Is(err, errBoom) {
			t.Fatalf("expected screenshot upsert error, got %v", err)
		}
	})

	t.Run("facades", func(t *testing.T) {
		directoryStore := &directoryFacadeStoreStub{
			directoryQueryStoreStub:   &directoryQueryStoreStub{listErr: errBoom, streamErr: errBoom, countErr: errBoom, scannedErr: errBoom},
			directoryCommandStoreStub: &directoryCommandStoreStub{batchCreateErr: errBoom, batchUpsertErr: errBoom},
		}
		directoryLookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		directoryFacade := NewDirectoryFacade(NewDirectoryQueryService(directoryStore, directoryLookup), NewDirectoryCommandService(directoryStore, directoryLookup))
		if _, err := directoryFacade.ListByTarget(1, DirectoryListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected directory facade list error, got %v", err)
		}
		if _, err := directoryFacade.BatchCreate(1, []string{"https://example.com"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected directory facade create error, got %v", err)
		}
		if err := directoryFacade.ForEachByTarget(1, func(Directory) error { return nil }); !errors.Is(err, errBoom) {
			t.Fatalf("expected directory facade for-each error, got %v", err)
		}
		if _, err := directoryFacade.CountByTarget(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected directory facade count error, got %v", err)
		}
		if _, err := directoryFacade.BatchUpsert(1, []DirectoryUpsertItem{{URL: "https://example.com"}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected directory facade upsert error, got %v", err)
		}

		endpointStore := &endpointFacadeStoreStub{
			endpointQueryStoreStub:   &endpointQueryStoreStub{listErr: errBoom, findErr: errBoom, streamErr: errBoom, countErr: errBoom, scannedErr: errBoom},
			endpointCommandStoreStub: &endpointCommandStoreStub{findByIDErr: errBoom, batchCreateErr: errBoom, batchUpsertErr: errBoom},
		}
		endpointLookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		endpointFacade := NewEndpointFacade(NewEndpointQueryService(endpointStore, endpointLookup), NewEndpointCommandService(endpointStore, endpointLookup))
		if _, err := endpointFacade.ListByTarget(1, EndpointListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade list error, got %v", err)
		}
		if _, err := endpointFacade.GetByID(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade get error, got %v", err)
		}
		if _, err := endpointFacade.BatchCreate(1, []string{"https://example.com"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade create error, got %v", err)
		}
		if err := endpointFacade.Delete(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade delete error, got %v", err)
		}
		if err := endpointFacade.ForEachByTarget(1, func(Endpoint) error { return nil }); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade for-each error, got %v", err)
		}
		if _, err := endpointFacade.CountByTarget(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade count error, got %v", err)
		}
		if _, err := endpointFacade.BatchUpsert(1, []EndpointUpsertItem{{URL: "https://example.com", Host: "example.com"}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected endpoint facade upsert error, got %v", err)
		}

		websiteStore := &websiteFacadeStoreStub{
			websiteQueryStoreStub:   &websiteQueryStoreStub{findErr: errBoom, streamErr: errBoom, countErr: errBoom},
			websiteCommandStoreStub: &websiteCommandStoreStub{findByIDErr: errBoom, batchCreateErr: errBoom, batchUpsertErr: errBoom},
		}
		websiteLookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		websiteFacade := NewWebsiteFacade(NewWebsiteQueryService(websiteStore, websiteStore, websiteLookup), NewWebsiteCommandService(websiteStore, websiteLookup))
		if _, err := websiteFacade.ListByTarget(1, WebsiteListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected website facade list error, got %v", err)
		}
		if err := websiteFacade.Delete(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected website facade delete error, got %v", err)
		}
		if err := websiteFacade.ForEachByTarget(1, func(Website) error { return nil }); !errors.Is(err, errBoom) {
			t.Fatalf("expected website facade for-each error, got %v", err)
		}
		if _, err := websiteFacade.CountByTarget(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected website facade count error, got %v", err)
		}
		if _, err := websiteFacade.BatchUpsert(1, []WebsiteUpsertItem{{URL: "https://example.com", Host: "example.com"}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected website facade upsert error, got %v", err)
		}

		subdomainStore := &subdomainFacadeStoreStub{
			subdomainQueryStoreStub:   &subdomainQueryStoreStub{forEachErr: errBoom, countErr: errBoom},
			subdomainCommandStoreStub: &subdomainCommandStoreStub{batchCreateErr: errBoom},
		}
		subdomainLookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		subdomainFacade := NewSubdomainFacade(NewSubdomainQueryService(subdomainStore, subdomainLookup), NewSubdomainCommandService(subdomainStore, subdomainLookup))
		if _, err := subdomainFacade.BatchCreate(1, []string{"api.example.com"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected subdomain facade create error, got %v", err)
		}
		if err := subdomainFacade.ForEachByTarget(1, func(Subdomain) error { return nil }); !errors.Is(err, errBoom) {
			t.Fatalf("expected subdomain facade for-each error, got %v", err)
		}
		if _, err := subdomainFacade.CountByTarget(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected subdomain facade count error, got %v", err)
		}

		hostPortStore := &hostPortFacadeStoreStub{
			hostPortQueryStoreStub:   &hostPortQueryStoreStub{listErr: errBoom, streamErr: errBoom, streamIPsErr: errBoom, countErr: errBoom, scannedErr: errBoom},
			hostPortCommandStoreStub: &hostPortCommandStoreStub{upsertErr: errBoom},
		}
		hostPortLookup := &hostPortCommandTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1}}}
		hostPortFacade := NewHostPortFacade(NewHostPortQueryService(hostPortStore, hostPortLookup), NewHostPortCommandService(hostPortStore, hostPortLookup))
		if _, err := hostPortFacade.ListByTarget(1, HostPortListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected host-port facade list error, got %v", err)
		}
		if err := hostPortFacade.ForEachByTarget(1, func(HostPort) error { return nil }); !errors.Is(err, errBoom) {
			t.Fatalf("expected host-port facade for-each error, got %v", err)
		}
		if err := hostPortFacade.ForEachByTargetAndIPs(1, []string{"1.1.1.1"}, func(HostPort) error { return nil }); !errors.Is(err, errBoom) {
			t.Fatalf("expected host-port facade for-each by ips error, got %v", err)
		}
		if _, err := hostPortFacade.CountByTarget(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected host-port facade count error, got %v", err)
		}
		if _, err := hostPortFacade.BatchUpsert(1, []HostPortItem{{Host: "a.example.com", IP: "1.1.1.1", Port: 443}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected host-port facade upsert error, got %v", err)
		}

		screenshotStore := &screenshotFacadeStoreStub{
			screenshotQueryStoreStub:   &screenshotQueryStoreStub{listErr: errBoom, findByIDErr: errBoom},
			screenshotCommandStoreStub: &screenshotCommandStoreStub{upsertErr: errBoom},
		}
		screenshotLookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}}
		screenshotFacade := NewScreenshotFacade(NewScreenshotQueryService(screenshotStore, screenshotLookup), NewScreenshotCommandService(screenshotStore, screenshotLookup))
		if _, err := screenshotFacade.ListByTarget(1, ScreenshotListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected screenshot facade list error, got %v", err)
		}
		if _, err := screenshotFacade.GetByID(1); !errors.Is(err, errBoom) {
			t.Fatalf("expected screenshot facade get error, got %v", err)
		}
		if _, err := screenshotFacade.BatchUpsert(1, &BatchUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://example.com"}}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected screenshot facade upsert error, got %v", err)
		}
	})
}

func TestApplicationLookupErrorPropagation(t *testing.T) {
	errLookup := errors.New("lookup failed")

	t.Run("directory query", func(t *testing.T) {
		service := NewDirectoryQueryService(&directoryQueryStoreStub{}, &directoryQueryTargetLookupStub{err: errLookup})
		if _, err := service.ListByTarget(context.Background(), 1, DirectoryListQueryInput{PageSize: 20}); !errors.Is(err, errLookup) {
			t.Fatalf("expected list lookup error, got %v", err)
		}
		if _, err := service.CountByTarget(context.Background(), 1); !errors.Is(err, errLookup) {
			t.Fatalf("expected count lookup error, got %v", err)
		}
	})

	t.Run("endpoint query", func(t *testing.T) {
		service := NewEndpointQueryService(&endpointQueryStoreStub{}, &endpointQueryTargetLookupStub{err: errLookup})
		if _, err := service.ListByTarget(context.Background(), 1, EndpointListQueryInput{PageSize: 20}); !errors.Is(err, errLookup) {
			t.Fatalf("expected list lookup error, got %v", err)
		}
		if err := service.ForEachByTarget(context.Background(), 1, func(assetdomain.Endpoint) error { return nil }); !errors.Is(err, errLookup) {
			t.Fatalf("expected for-each lookup error, got %v", err)
		}
	})

	t.Run("subdomain query", func(t *testing.T) {
		service := NewSubdomainQueryService(&subdomainQueryStoreStub{}, &subdomainQueryTargetLookupStub{err: errLookup})
		if _, err := service.ListByTarget(context.Background(), 1, SubdomainListQueryInput{PageSize: 20}); !errors.Is(err, errLookup) {
			t.Fatalf("expected list lookup error, got %v", err)
		}
		if err := service.ForEachByTarget(context.Background(), 1, func(assetdomain.Subdomain) error { return nil }); !errors.Is(err, errLookup) {
			t.Fatalf("expected for-each lookup error, got %v", err)
		}
		if _, err := service.CountByTarget(context.Background(), 1); !errors.Is(err, errLookup) {
			t.Fatalf("expected count lookup error, got %v", err)
		}
	})

	t.Run("website query", func(t *testing.T) {
		store := &websiteQueryStoreStub{}
		service := NewWebsiteQueryService(store, store, &websiteTargetLookupQueryStub{err: errLookup})
		if _, err := service.ListByTarget(context.Background(), 1, WebsiteListQueryInput{PageSize: 20}); !errors.Is(err, errLookup) {
			t.Fatalf("expected list lookup error, got %v", err)
		}
		if err := service.ForEachByTarget(context.Background(), 1, func(assetdomain.Website) error { return nil }); !errors.Is(err, errLookup) {
			t.Fatalf("expected for-each lookup error, got %v", err)
		}
		if _, err := service.CountByTarget(context.Background(), 1); !errors.Is(err, errLookup) {
			t.Fatalf("expected count lookup error, got %v", err)
		}
	})

	t.Run("host port query", func(t *testing.T) {
		service := NewHostPortQueryService(&hostPortQueryStoreStub{}, &hostPortTargetLookupStub{err: errLookup})
		if err := service.ForEachByTarget(context.Background(), 1, func(assetdomain.HostPort) error { return nil }); !errors.Is(err, errLookup) {
			t.Fatalf("expected for-each lookup error, got %v", err)
		}
		if err := service.ForEachByTargetAndIPs(context.Background(), 1, []string{"1.1.1.1"}, func(assetdomain.HostPort) error { return nil }); !errors.Is(err, errLookup) {
			t.Fatalf("expected for-each-by-ips lookup error, got %v", err)
		}
		if _, err := service.CountByTarget(context.Background(), 1); !errors.Is(err, errLookup) {
			t.Fatalf("expected count lookup error, got %v", err)
		}
	})

	t.Run("screenshot query", func(t *testing.T) {
		service := NewScreenshotQueryService(&screenshotQueryStoreStub{}, &screenshotTargetLookupStub{err: errLookup})
		if _, err := service.ListByTarget(context.Background(), 1, ScreenshotListQueryInput{PageSize: 20}); !errors.Is(err, errLookup) {
			t.Fatalf("expected list lookup error, got %v", err)
		}
	})
}

func TestApplicationCommandServicesAdditionalBranches(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("directory command", func(t *testing.T) {
		store := &directoryCommandStoreStub{}
		lookup := &directoryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		service := NewDirectoryCommandService(store, lookup)

		lookup.err = errBoom
		if _, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch create lookup error, got %v", err)
		}

		lookup.err = nil
		created, err := service.BatchCreate(context.Background(), 1, []string{"https://other.com/a"})
		if err != nil || created != 0 || store.batchCreateCalls != 0 {
			t.Fatalf("expected unmatched batch create to be skipped, created=%d calls=%d err=%v", created, store.batchCreateCalls, err)
		}

		lookup.err = errBoom
		if _, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://example.com/a"}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch upsert lookup error, got %v", err)
		}

		lookup.err = nil
		affected, err := service.BatchUpsert(context.Background(), 1, []DirectoryUpsertItem{{URL: "https://other.com/a"}})
		if err != nil || affected != 0 || store.batchUpsertCalls != 0 {
			t.Fatalf("expected unmatched batch upsert to be skipped, affected=%d calls=%d err=%v", affected, store.batchUpsertCalls, err)
		}
	})

	t.Run("endpoint command", func(t *testing.T) {
		store := &endpointCommandStoreStub{}
		lookup := &endpointTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		service := NewEndpointCommandService(store, lookup)

		lookup.err = errBoom
		if _, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com/a"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch create lookup error, got %v", err)
		}

		lookup.err = nil
		created, err := service.BatchCreate(context.Background(), 1, []string{"https://other.com/a"})
		if err != nil || created != 0 || store.batchCreateCalls != 0 {
			t.Fatalf("expected unmatched batch create to be skipped, created=%d calls=%d err=%v", created, store.batchCreateCalls, err)
		}

		lookup.err = errBoom
		if _, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{{URL: "https://example.com/a", Host: "example.com"}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch upsert lookup error, got %v", err)
		}

		lookup.err = nil
		affected, err := service.BatchUpsert(context.Background(), 1, []EndpointUpsertItem{{URL: "https://other.com/a", Host: "other.com"}})
		if err != nil || affected != 0 || store.batchUpsertCalls != 0 {
			t.Fatalf("expected unmatched batch upsert to be skipped, affected=%d calls=%d err=%v", affected, store.batchUpsertCalls, err)
		}
	})

	t.Run("screenshot command", func(t *testing.T) {
		store := &screenshotCommandStoreStub{}
		lookup := &screenshotTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		service := NewScreenshotCommandService(store, lookup)

		lookup.err = errBoom
		if _, err := service.BatchUpsert(context.Background(), 1, &BatchUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://example.com/a"}}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch upsert lookup error, got %v", err)
		}

		lookup.err = nil
		affected, err := service.BatchUpsert(context.Background(), 1, &BatchUpsertScreenshotRequest{Screenshots: []ScreenshotItem{{URL: "https://other.com/a"}}})
		if err != nil || affected != 0 || len(store.upsertIn) != 0 {
			t.Fatalf("expected unmatched screenshot upsert to be skipped, affected=%d len=%d err=%v", affected, len(store.upsertIn), err)
		}
	})

	t.Run("subdomain command", func(t *testing.T) {
		store := &subdomainCommandStoreStub{}
		lookup := &subdomainTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		service := NewSubdomainCommandService(store, lookup)

		lookup.err = errBoom
		if _, err := service.BatchCreate(context.Background(), 1, []string{"api.example.com"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch create lookup error, got %v", err)
		}

		lookup.err = nil
		created, err := service.BatchCreate(context.Background(), 1, []string{"foo.com"})
		if err != nil || created != 0 || len(store.batchCreateIn) != 0 {
			t.Fatalf("expected unmatched subdomain create to be skipped, created=%d len=%d err=%v", created, len(store.batchCreateIn), err)
		}
	})

	t.Run("website command", func(t *testing.T) {
		store := &websiteCommandStoreStub{}
		lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		service := NewWebsiteCommandService(store, lookup)

		lookup.err = errBoom
		if _, err := service.BatchCreate(context.Background(), 1, []string{"https://example.com"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch create lookup error, got %v", err)
		}

		lookup.err = nil
		created, err := service.BatchCreate(context.Background(), 1, []string{"https://other.com"})
		if err != nil || created != 0 || len(store.batchCreateIn) != 0 {
			t.Fatalf("expected unmatched batch create to be skipped, created=%d len=%d err=%v", created, len(store.batchCreateIn), err)
		}

		lookup.err = errBoom
		if _, err := service.BatchUpsert(context.Background(), 1, []WebsiteUpsertItem{{URL: "https://example.com", Host: "example.com"}}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch upsert lookup error, got %v", err)
		}

		lookup.err = nil
		affected, err := service.BatchUpsert(context.Background(), 1, []WebsiteUpsertItem{{URL: "https://other.com", Host: "other.com"}})
		if err != nil || affected != 0 || len(store.batchUpsertIn) != 0 {
			t.Fatalf("expected unmatched batch upsert to be skipped, affected=%d len=%d err=%v", affected, len(store.batchUpsertIn), err)
		}
	})
}

func TestApplicationFacadeAndQueryAdditionalBranches(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("subdomain facade mappings", func(t *testing.T) {
		store := &subdomainFacadeStoreStub{
			subdomainQueryStoreStub: &subdomainQueryStoreStub{listErr: errBoom},
			subdomainCommandStoreStub: &subdomainCommandStoreStub{
				batchCreateErr: gorm.ErrRecordNotFound,
			},
		}
		lookup := &subdomainQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		facade := NewSubdomainFacade(NewSubdomainQueryService(store, lookup), NewSubdomainCommandService(store, lookup))

		if _, err := facade.ListByTarget(1, SubdomainListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected list error to propagate, got %v", err)
		}
		if _, err := facade.BatchCreate(1, []string{"api.example.com"}); !errors.Is(err, ErrTargetNotFound) {
			t.Fatalf("expected batch create record-not-found to map to ErrTargetNotFound, got %v", err)
		}
	})

	t.Run("website facade batch create generic error", func(t *testing.T) {
		store := &websiteFacadeStoreStub{
			websiteQueryStoreStub:   &websiteQueryStoreStub{},
			websiteCommandStoreStub: &websiteCommandStoreStub{batchCreateErr: errBoom},
		}
		lookup := &websiteTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		facade := NewWebsiteFacade(NewWebsiteQueryService(store, store, lookup), NewWebsiteCommandService(store, lookup))

		if _, err := facade.BatchCreate(1, []string{"https://example.com"}); !errors.Is(err, errBoom) {
			t.Fatalf("expected batch create error to propagate, got %v", err)
		}
	})

	t.Run("host port query list generic branches", func(t *testing.T) {
		store := &hostPortQueryStoreStub{
			ipRows: []assetdomain.IPAggregationRow{{IP: "1.1.1.1"}},
		}
		lookup := &hostPortTargetLookupStub{targets: map[int]*assetdomain.TargetRef{
			1: {ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain},
		}}
		service := NewHostPortQueryService(store, lookup)

		lookup.err = errBoom
		if _, err := service.ListByTarget(context.Background(), 1, HostPortListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected list lookup error, got %v", err)
		}

		lookup.err = nil
		store.hostPortErr = errBoom
		if _, err := service.ListByTarget(context.Background(), 1, HostPortListQueryInput{PageSize: 20}); !errors.Is(err, errBoom) {
			t.Fatalf("expected hosts/ports error, got %v", err)
		}
	})
}
