package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type wordlistQueryStoreStub struct {
	listItems        []catalogdomain.Wordlist
	tagSummaries     []catalogdomain.WordlistTagSummary
	findByID         map[int]*catalogdomain.Wordlist
	listErr          error
	findErr          error
	updated          *catalogdomain.Wordlist
	lastPage         int
	lastPageSize     int
	lastFilter       string
	lastOrderBy      string
	lastSummaryPage  int
	lastSummarySize  int
	lastSummaryQuery string
	findContext      context.Context
}

func (stub *wordlistQueryStoreStub) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := append([]catalogdomain.Wordlist(nil), stub.listItems...)
	return result, int64(len(result)), nil
}

func (stub *wordlistQueryStoreStub) ListTagSummaries(page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error) {
	stub.lastSummaryPage = page
	stub.lastSummarySize = pageSize
	stub.lastSummaryQuery = filter
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := append([]catalogdomain.WordlistTagSummary(nil), stub.tagSummaries...)
	return result, int64(len(result)), nil
}

func (stub *wordlistQueryStoreStub) ListAll() ([]catalogdomain.Wordlist, error) {
	if stub.listErr != nil {
		return nil, stub.listErr
	}
	return append([]catalogdomain.Wordlist(nil), stub.listItems...), nil
}

func (stub *wordlistQueryStoreStub) GetByID(id int) (*catalogdomain.Wordlist, error) {
	if stub.findErr != nil {
		return nil, stub.findErr
	}
	item, ok := stub.findByID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copyItem := *item
	return &copyItem, nil
}

func (stub *wordlistQueryStoreStub) GetByIDContext(ctx context.Context, id int) (*catalogdomain.Wordlist, error) {
	stub.findContext = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return stub.GetByID(id)
}

func (stub *wordlistQueryStoreStub) Update(wordlist *catalogdomain.Wordlist) error {
	copyItem := *wordlist
	stub.updated = &copyItem
	return nil
}

func TestWordlistQueryServiceGetAndPath(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "dict.txt")
	if err := os.WriteFile(filePath, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatalf("write fixture file failed: %v", err)
	}

	store := &wordlistQueryStoreStub{
		findByID: map[int]*catalogdomain.Wordlist{
			1: {ID: 1, FileName: "dict.txt", FilePath: filePath},
		},
	}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	wordlist, err := service.GetWordlistByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if wordlist.ID != 1 {
		t.Fatalf("unexpected wordlist: %+v", wordlist)
	}

	path, err := service.GetWordlistFilePathByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("get file path failed: %v", err)
	}
	if path != filePath {
		t.Fatalf("unexpected file path: %s", path)
	}
}

func TestExecutionWordlistSourceForwardsContextWithoutRefreshingMetadata(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "dict.txt")
	if err := os.WriteFile(filePath, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &wordlistQueryStoreStub{findByID: map[int]*catalogdomain.Wordlist{
		7: {ID: 7, FileName: "dict.txt", FilePath: filePath, FileSize: 8, LineCount: 2},
	}}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())
	source := NewExecutionWordlistSource(service)
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request"), "artifact-1")

	resourceName := resourcenames.Wordlist(7)
	wordlist, err := source.GetByResourceName(ctx, resourceName)
	if err != nil {
		t.Fatalf("GetByResourceName failed: %v", err)
	}
	if wordlist.ID != 7 || store.findContext != ctx {
		t.Fatalf("metadata read did not receive the original context: wordlist=%+v ctx=%v", wordlist, store.findContext)
	}
	if store.updated != nil {
		t.Fatalf("execution metadata read must not refresh or write catalog state: %+v", store.updated)
	}

	path, err := source.GetFilePathByResourceName(ctx, resourceName)
	if err != nil {
		t.Fatalf("GetFilePathByResourceName failed: %v", err)
	}
	if path != filePath || store.findContext != ctx {
		t.Fatalf("path read = %q with ctx=%v, want %q with original context", path, store.findContext, filePath)
	}
}

func TestExecutionWordlistSourcePropagatesCancellation(t *testing.T) {
	store := &wordlistQueryStoreStub{findByID: map[int]*catalogdomain.Wordlist{
		7: {ID: 7, FileName: "dict.txt", FilePath: filepath.Join(t.TempDir(), "dict.txt")},
	}}
	source := NewExecutionWordlistSource(NewWordlistQueryService(store, newWordlistFileStoreTestStub()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resourceName := resourcenames.Wordlist(7)
	if _, err := source.GetByResourceName(ctx, resourceName); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByResourceName error = %v, want context.Canceled", err)
	}
	if _, err := source.GetFilePathByResourceName(ctx, resourceName); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetFilePathByResourceName error = %v, want context.Canceled", err)
	}
}

func TestWordlistQueryServiceListAll(t *testing.T) {
	store := &wordlistQueryStoreStub{
		listItems: []catalogdomain.Wordlist{{ID: 1, FileName: "a.txt"}, {ID: 2, FileName: "b.txt"}},
	}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	items, err := service.ListAllWordlists(context.Background())
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestWordlistQueryServiceListWordlistsPassesFilterAndOrder(t *testing.T) {
	store := &wordlistQueryStoreStub{listItems: []catalogdomain.Wordlist{{ID: 1, FileName: "a.txt"}, {ID: 2, FileName: "b.txt"}}}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	result, err := service.ListWordlists(context.Background(), WordlistListQueryInput{
		PageSize: 1,
		Filter:   `tags=="fuzz"`,
		OrderBy:  "updatedAt desc",
	})
	if err != nil {
		t.Fatalf("list wordlists failed: %v", err)
	}
	if len(result.Wordlists) != 2 || result.TotalSize != 2 || result.NextPageToken == "" {
		t.Fatalf("unexpected list result: %+v", result)
	}
	if store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `tags=="fuzz"` || store.lastOrderBy != "updatedAt desc" {
		t.Fatalf("unexpected list args: page=%d size=%d filter=%q order=%q", store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
}

func TestWordlistQueryServiceBindsPageTokenToQueryShape(t *testing.T) {
	store := &wordlistQueryStoreStub{
		listItems: []catalogdomain.Wordlist{{ID: 1, FileName: "a.txt"}, {ID: 2, FileName: "b.txt"}},
	}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	first, err := service.ListWordlists(context.Background(), WordlistListQueryInput{
		PageSize: 1,
		Filter:   `fileName="common.txt"`,
		OrderBy:  "fileName asc",
	})
	if err != nil {
		t.Fatalf("list first page failed: %v", err)
	}
	if first.NextPageToken == "" {
		t.Fatal("expected bound nextPageToken for additional wordlist pages")
	}

	_, err = service.ListWordlists(context.Background(), WordlistListQueryInput{
		PageSize:  1,
		PageToken: first.NextPageToken,
		Filter:    `fileName="changed.txt"`,
		OrderBy:   "fileName asc",
	})
	if !errors.Is(err, ErrInvalidWordlistPageToken) {
		t.Fatalf("expected query-shape mismatch to reject pageToken, got %v", err)
	}

	second, err := service.ListWordlists(context.Background(), WordlistListQueryInput{
		PageSize:  1,
		PageToken: first.NextPageToken,
		Filter:    `fileName="common.txt"`,
		OrderBy:   "fileName asc",
	})
	if err != nil {
		t.Fatalf("list second page failed: %v", err)
	}
	if second.Page != 2 || store.lastPage != 2 {
		t.Fatalf("expected pageToken to resolve page 2, result page=%d store page=%d", second.Page, store.lastPage)
	}
}

func TestWordlistQueryServiceRejectsMalformedPageToken(t *testing.T) {
	store := &wordlistQueryStoreStub{}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	_, err := service.ListWordlists(context.Background(), WordlistListQueryInput{PageToken: "not-a-token"})
	if !errors.Is(err, ErrInvalidWordlistPageToken) {
		t.Fatalf("expected malformed pageToken to be rejected, got %v", err)
	}
	if store.lastPage != 0 {
		t.Fatalf("malformed pageToken must be rejected before store call, got page=%d", store.lastPage)
	}
}

func TestWordlistQueryServiceListWordlistTagSummaries(t *testing.T) {
	store := &wordlistQueryStoreStub{tagSummaries: []catalogdomain.WordlistTagSummary{{DisplayName: "fuzz", WordlistCount: 2}}}
	service := NewWordlistQueryService(store, newWordlistFileStoreTestStub())

	items, total, err := service.ListWordlistTagSummaries(context.Background(), 1, 20, "fu")
	if err != nil {
		t.Fatalf("list tag summaries failed: %v", err)
	}
	if len(items) != 1 || items[0].DisplayName != "fuzz" || total != 1 {
		t.Fatalf("unexpected tag summaries: items=%+v total=%d", items, total)
	}
	if store.lastSummaryPage != 1 || store.lastSummarySize != 20 || store.lastSummaryQuery != "fu" {
		t.Fatalf("unexpected summary args: page=%d size=%d filter=%q", store.lastSummaryPage, store.lastSummarySize, store.lastSummaryQuery)
	}
}
