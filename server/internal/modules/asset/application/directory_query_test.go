package application

import (
	"context"
	"errors"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	"gorm.io/gorm"
)

type directoryQueryStoreStub struct {
	items          []assetdomain.Directory
	total          int64
	count          int64
	listErr        error
	streamErr      error
	countErr       error
	scannedErr     error
	listTargetID   int
	listPage       int
	listPageSize   int
	listFilter     string
	listOrderBy    string
	optionTargetID int
	optionField    string
	streamID       int
	countID        int
}

func (stub *directoryQueryStoreStub) ListByTargetID(targetID int, page, pageSize int, filter, orderBy string) ([]assetdomain.Directory, int64, error) {
	stub.listTargetID = targetID
	stub.listPage = page
	stub.listPageSize = pageSize
	stub.listFilter = filter
	stub.listOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	return append([]assetdomain.Directory(nil), stub.items...), stub.total, nil
}

func (stub *directoryQueryStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]assetdomain.FilterOption, error) {
	stub.optionTargetID = targetID
	stub.optionField = field
	return []assetdomain.FilterOption{{Value: "200", Label: "200", Count: 2}}, nil
}

func (stub *directoryQueryStoreStub) ForEachByTargetID(ctx context.Context, targetID int, visit func(assetdomain.Directory) error) error {
	_ = ctx
	stub.streamID = targetID
	if stub.streamErr != nil {
		return stub.streamErr
	}
	if stub.scannedErr != nil {
		return stub.scannedErr
	}
	for _, item := range stub.items {
		if err := visit(item); err != nil {
			return err
		}
	}
	return nil
}

func (stub *directoryQueryStoreStub) CountByTargetID(targetID int) (int64, error) {
	stub.countID = targetID
	if stub.countErr != nil {
		return 0, stub.countErr
	}
	return stub.count, nil
}

type directoryQueryTargetLookupStub struct {
	targets map[int]*assetdomain.TargetRef
	err     error
}

func (stub *directoryQueryTargetLookupStub) GetActiveByID(id int) (*assetdomain.TargetRef, error) {
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

func TestDirectoryQueryServiceListAndCount(t *testing.T) {
	store := &directoryQueryStoreStub{
		items: []assetdomain.Directory{{ID: 1}, {ID: 2}},
		total: 2,
		count: 8,
	}
	lookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewDirectoryQueryService(store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{PageSize: 20, Filter: `url="admin"`})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(result.Directories) != 2 || result.TotalSize != 2 {
		t.Fatalf("unexpected list result len=%d total=%d", len(result.Directories), result.TotalSize)
	}
	if store.listTargetID != 7 || store.listPage != 1 || store.listPageSize != 20 || store.listFilter != `url="admin"` || store.listOrderBy != "createdAt desc" {
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

func TestDirectoryQueryServiceTargetNotFound(t *testing.T) {
	service := NewDirectoryQueryService(&directoryQueryStoreStub{}, &directoryQueryTargetLookupStub{err: gorm.ErrRecordNotFound})

	_, err := service.ListByTarget(context.Background(), 1, DirectoryListQueryInput{PageSize: 20})
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestDirectoryQueryServiceCanonicalControls(t *testing.T) {
	store := &directoryQueryStoreStub{items: []assetdomain.Directory{{ID: 1}}, total: 21}
	lookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewDirectoryQueryService(store, lookup)

	result, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{
		PageSize: 10,
		Filter:   `url="admin" && (status="200" || status="301") && contentType="text/html"`,
		OrderBy:  "status desc",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if result.TotalSize != 21 || result.NextPageToken == "" {
		t.Fatalf("unexpected result total=%d token=%q", result.TotalSize, result.NextPageToken)
	}
	if store.listPage != 1 || store.listPageSize != 10 || store.listOrderBy != "status desc" {
		t.Fatalf("unexpected list args page=%d size=%d orderBy=%q", store.listPage, store.listPageSize, store.listOrderBy)
	}

	if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{Filter: `content_type="text/html"`}); !errors.Is(err, ErrUnsupportedDirectoryFilter) {
		t.Fatalf("expected unsupported snake_case filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{Filter: `tech="react"`}); !errors.Is(err, ErrUnsupportedDirectoryFilter) {
		t.Fatalf("expected unsupported tech filter, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{OrderBy: "content_length desc"}); !errors.Is(err, ErrUnsupportedDirectoryOrderBy) {
		t.Fatalf("expected unsupported snake_case orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{OrderBy: "url desc"}); !errors.Is(err, ErrUnsupportedDirectoryOrderBy) {
		t.Fatalf("expected unsupported url orderBy, got %v", err)
	}
	if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{OrderBy: "contentType desc"}); !errors.Is(err, ErrUnsupportedDirectoryOrderBy) {
		t.Fatalf("expected unsupported contentType orderBy, got %v", err)
	}

	_, err = service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{
		PageSize:  10,
		PageToken: result.NextPageToken,
		Filter:    `url="changed"`,
		OrderBy:   "status desc",
	})
	if !errors.Is(err, ErrInvalidDirectoryPageToken) {
		t.Fatalf("expected invalid token on query shape mismatch, got %v", err)
	}
}

func TestDirectoryQueryServiceWebsiteScopeValidationAndPageBinding(t *testing.T) {
	store := &directoryQueryStoreStub{items: []assetdomain.Directory{{ID: 1}}, total: 2}
	service := NewDirectoryQueryService(store, &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}})
	filter := `websiteUrl=="https://api.acme.com/a" && status=="200"`

	first, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{PageSize: 1, Filter: filter})
	if err != nil {
		t.Fatalf("ListByTarget returned error: %v", err)
	}
	if store.listFilter != filter || first.TotalSize != 2 || first.NextPageToken == "" {
		t.Fatalf("unexpected scoped directory result=%+v filter=%q", first, store.listFilter)
	}
	if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{PageSize: 1, PageToken: first.NextPageToken, Filter: `websiteUrl=="https://api.acme.com/b" && status=="200"`}); !errors.Is(err, ErrInvalidDirectoryPageToken) {
		t.Fatalf("expected Website scope token binding, got %v", err)
	}

	for _, invalid := range []string{
		`websiteUrl="https://api.acme.com"`,
		`websiteUrl=="https://api.acme.com" || status=="200"`,
		`websiteUrl=="https://api.acme.com" && websiteUrl=="https://api.acme.com/a"`,
		`websiteUrl=="not-a-url"`,
	} {
		if _, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{Filter: invalid}); !errors.Is(err, ErrUnsupportedDirectoryFilter) {
			t.Fatalf("filter %q: expected ErrUnsupportedDirectoryFilter, got %v", invalid, err)
		}
	}
}

func TestDirectoryQueryServiceFilterOptions(t *testing.T) {
	store := &directoryQueryStoreStub{}
	lookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}
	service := NewDirectoryQueryService(store, lookup)

	options, err := service.ListFilterOptionsByTarget(context.Background(), 7, "status")
	if err != nil {
		t.Fatalf("filter options failed: %v", err)
	}
	if len(options) != 1 || options[0].Value != "200" || store.optionTargetID != 7 || store.optionField != "status" {
		t.Fatalf("unexpected options=%+v target=%d field=%q", options, store.optionTargetID, store.optionField)
	}

	if _, err := service.ListFilterOptionsByTarget(context.Background(), 7, "url"); !errors.Is(err, ErrUnsupportedDirectoryFilter) {
		t.Fatalf("expected unsupported filter option field, got %v", err)
	}

	notFoundService := NewDirectoryQueryService(store, &directoryQueryTargetLookupStub{err: gorm.ErrRecordNotFound})
	if _, err := notFoundService.ListFilterOptionsByTarget(context.Background(), 7, "status"); !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestDirectoryQueryServicePropagatesStoreErrors(t *testing.T) {
	lookup := &directoryQueryTargetLookupStub{targets: map[int]*assetdomain.TargetRef{7: {ID: 7}}}

	t.Run("list", func(t *testing.T) {
		service := NewDirectoryQueryService(&directoryQueryStoreStub{listErr: errors.New("list failed")}, lookup)

		_, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{PageSize: 20})
		if err == nil || err.Error() != "list failed" {
			t.Fatalf("expected list error, got %v", err)
		}
	})

	t.Run("stream", func(t *testing.T) {
		service := NewDirectoryQueryService(&directoryQueryStoreStub{streamErr: errors.New("stream failed")}, lookup)

		err := service.ForEachByTarget(context.Background(), 7, func(assetdomain.Directory) error { return nil })
		if err == nil || err.Error() != "stream failed" {
			t.Fatalf("expected stream error, got %v", err)
		}
	})

	t.Run("count", func(t *testing.T) {
		service := NewDirectoryQueryService(&directoryQueryStoreStub{countErr: errors.New("count failed")}, lookup)

		_, err := service.CountByTarget(context.Background(), 7)
		if err == nil || err.Error() != "count failed" {
			t.Fatalf("expected count error, got %v", err)
		}
	})

	t.Run("scan row", func(t *testing.T) {
		service := NewDirectoryQueryService(&directoryQueryStoreStub{scannedErr: errors.New("scan failed")}, lookup)

		err := service.ForEachByTarget(context.Background(), 7, func(assetdomain.Directory) error { return nil })
		if err == nil || err.Error() != "scan failed" {
			t.Fatalf("expected scan error, got %v", err)
		}
	})
}

func TestDirectoryQueryServiceLookupErrorsShortCircuitStore(t *testing.T) {
	lookupErr := errors.New("lookup failed")

	tests := []struct {
		name          string
		run           func(*DirectoryQueryService) error
		assertSkipped func(*directoryQueryStoreStub) error
	}{
		{
			name: "list",
			run: func(service *DirectoryQueryService) error {
				_, err := service.ListByTarget(context.Background(), 7, DirectoryListQueryInput{PageSize: 20, Filter: `status="200"`})
				return err
			},
			assertSkipped: func(store *directoryQueryStoreStub) error {
				if store.listTargetID != 0 {
					return errors.New("list store was called")
				}
				return nil
			},
		},
		{
			name: "count",
			run: func(service *DirectoryQueryService) error {
				_, err := service.CountByTarget(context.Background(), 7)
				return err
			},
			assertSkipped: func(store *directoryQueryStoreStub) error {
				if store.countID != 0 {
					return errors.New("count store was called")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &directoryQueryStoreStub{
				listErr:  errors.New("should not be called"),
				countErr: errors.New("should not be called"),
			}
			service := NewDirectoryQueryService(store, &directoryQueryTargetLookupStub{err: lookupErr})

			err := tt.run(service)
			if !errors.Is(err, lookupErr) {
				t.Fatalf("expected lookup error %v, got %v", lookupErr, err)
			}
			if err := tt.assertSkipped(store); err != nil {
				t.Fatalf("expected store call to be skipped: %v", err)
			}
		})
	}
}
