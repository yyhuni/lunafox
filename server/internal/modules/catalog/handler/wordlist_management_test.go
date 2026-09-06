package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type wordlistManagementStoreStub struct {
	items            []catalogdomain.Wordlist
	tagSummaries     []catalogdomain.WordlistTagSummary
	byID             map[int]*catalogdomain.Wordlist
	existsByFileName map[string]bool
	lastPage         int
	lastPageSize     int
	lastFilter       string
	lastOrderBy      string
	lastSummaryPage  int
	lastSummarySize  int
	lastSummaryQuery string
	updated          *catalogdomain.Wordlist
	deletedID        int
}

func (stub *wordlistManagementStoreStub) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return append([]catalogdomain.Wordlist(nil), stub.items...), int64(len(stub.items) + 1), nil
}

func (stub *wordlistManagementStoreStub) ListAll() ([]catalogdomain.Wordlist, error) {
	return append([]catalogdomain.Wordlist(nil), stub.items...), nil
}

func (stub *wordlistManagementStoreStub) ListTagSummaries(page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error) {
	stub.lastSummaryPage = page
	stub.lastSummarySize = pageSize
	stub.lastSummaryQuery = filter
	return append([]catalogdomain.WordlistTagSummary(nil), stub.tagSummaries...), int64(len(stub.tagSummaries)), nil
}

func (stub *wordlistManagementStoreStub) GetByID(id int) (*catalogdomain.Wordlist, error) {
	if item, ok := stub.byID[id]; ok {
		copyItem := *item
		return &copyItem, nil
	}
	return nil, catalogapp.ErrWordlistNotFound
}

func (stub *wordlistManagementStoreStub) GetByIDContext(_ context.Context, id int) (*catalogdomain.Wordlist, error) {
	return stub.GetByID(id)
}

func (stub *wordlistManagementStoreStub) ExistsByFileName(fileName string, excludeID ...int) (bool, error) {
	_ = excludeID
	return stub.existsByFileName[fileName], nil
}

func (stub *wordlistManagementStoreStub) Create(wordlist *catalogdomain.Wordlist) error {
	stub.items = append(stub.items, *wordlist)
	return nil
}

func (stub *wordlistManagementStoreStub) Update(wordlist *catalogdomain.Wordlist) error {
	copyItem := *wordlist
	stub.updated = &copyItem
	return nil
}

func (stub *wordlistManagementStoreStub) Delete(id int) error {
	stub.deletedID = id
	delete(stub.byID, id)
	return nil
}

func newWordlistManagementTestRouter(store *wordlistManagementStoreStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	fileStore := catalogapp.NewLocalWordlistFileStore()
	handler := NewWordlistHandler(catalogapp.NewWordlistFacade(
		catalogapp.NewWordlistQueryService(store, fileStore),
		catalogapp.NewWordlistCommandService(store, "", fileStore),
	))
	router.GET("/v1/wordlists", handler.List)
	router.PATCH("/v1/wordlists/:wordlist", handler.Update)
	router.DELETE("/v1/wordlists/:wordlist", handler.Delete)
	router.GET("/v1/wordlistTags", handler.ListTags)
	return router
}

func TestWordlistDeleteKeepsOrdinaryAndInstallationSeededResourcesIndependentFromConsumers(t *testing.T) {
	for _, test := range []struct {
		name         string
		wordlistName string
	}{
		{name: "ordinary", wordlistName: "custom-subdomains.txt"},
		{name: "installation seeded", wordlistName: "subdomains-top1million-110000.txt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), test.wordlistName)
			if err := os.WriteFile(filePath, []byte("example\n"), 0o644); err != nil {
				t.Fatalf("write fixture file: %v", err)
			}
			store := &wordlistManagementStoreStub{
				byID:             map[int]*catalogdomain.Wordlist{7: {ID: 7, FileName: test.wordlistName, FilePath: filePath}},
				existsByFileName: map[string]bool{},
			}
			router := newWordlistManagementTestRouter(store)

			req := httptest.NewRequest(http.MethodDelete, "/v1/wordlists/7", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusNoContent {
				t.Fatalf("DELETE status = %d body=%s, want 204", resp.Code, resp.Body.String())
			}
			if store.deletedID != 7 {
				t.Fatalf("deleted ID = %d, want 7", store.deletedID)
			}
			if _, exists := store.byID[7]; exists {
				t.Fatal("handler delete restored the Catalog row")
			}
			if store.updated != nil || len(store.items) != 0 {
				t.Fatalf("handler delete rewrote or replaced consumer state: updated=%#v items=%#v", store.updated, store.items)
			}
			if _, err := os.Stat(filePath); !os.IsNotExist(err) {
				t.Fatalf("handler delete retained wordlist file: %v", err)
			}
		})
	}
}

func newWordlistManagementCreateTestRouter(t *testing.T, store *wordlistManagementStoreStub) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	fileStore := catalogapp.NewLocalWordlistFileStore()
	handler := NewWordlistHandler(catalogapp.NewWordlistFacade(
		catalogapp.NewWordlistQueryService(store, fileStore),
		catalogapp.NewWordlistCommandService(store, t.TempDir(), fileStore),
	))
	router.POST("/v1/wordlists", handler.Create)
	return router
}

func TestWordlistCreateReturnsFileNameFromUploadedFilename(t *testing.T) {
	store := &wordlistManagementStoreStub{existsByFileName: map[string]bool{}}
	router := newWordlistManagementCreateTestRouter(t, store)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("description", "common paths"); err != nil {
		t.Fatalf("write description field: %v", err)
	}
	if err := writer.WriteField("tags", "目录扫描, fuzz"); err != nil {
		t.Fatalf("write tags field: %v", err)
	}
	part, err := writer.CreateFormFile("file", "common-paths.txt")
	if err != nil {
		t.Fatalf("create file field: %v", err)
	}
	if _, err := part.Write([]byte("admin\nbackup\n")); err != nil {
		t.Fatalf("write file field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/wordlists", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
	}
	if len(store.items) != 1 || store.items[0].FileName != "common-paths.txt" {
		t.Fatalf("expected uploaded filename as wordlist fileName, got %+v", store.items)
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload["fileName"] != "common-paths.txt" {
		t.Fatalf("expected fileName from uploaded filename, got %s", resp.Body.String())
	}
	if _, exists := payload["displayName"]; exists {
		t.Fatalf("wordlist response must not expose displayName: %s", resp.Body.String())
	}
}

func TestWordlistListReturnsPaginatedCanonicalResources(t *testing.T) {
	updatedAt := time.Date(2026, 7, 4, 8, 0, 0, 0, time.UTC)
	store := &wordlistManagementStoreStub{
		items: []catalogdomain.Wordlist{{
			ID:          7,
			FileName:    "subdomains.txt",
			Description: "subdomain scan",
			Tags:        []string{"fuzz", "subdomain"},
			FileSize:    512,
			LineCount:   12,
			FileHash:    "abc123",
			UpdatedAt:   updatedAt,
		}},
	}
	router := newWordlistManagementTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/wordlists?pageSize=1&filter=tags==\"fuzz\"&orderBy=updatedAt%20desc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.lastPageSize != 1 || store.lastFilter != `tags=="fuzz"` || store.lastOrderBy != "updatedAt desc" {
		t.Fatalf("unexpected list query: pageSize=%d filter=%q orderBy=%q", store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := body["results"].([]any); !ok {
		t.Fatalf("expected paginated results object, got %s", resp.Body.String())
	}
	if body["totalSize"] != float64(2) || body["nextPageToken"] == "" {
		t.Fatalf("expected totalSize and nextPageToken, got %s", resp.Body.String())
	}
	first := body["results"].([]any)[0].(map[string]any)
	if first["name"] != "wordlists/7" || first["fileName"] != "subdomains.txt" {
		t.Fatalf("expected canonical name and fileName, got %+v", first)
	}
	if _, exists := first["displayName"]; exists {
		t.Fatalf("wordlist response must not expose displayName: %+v", first)
	}
	if first["lineCount"] != float64(12) || first["fileSize"] != float64(512) || first["fileHash"] != "abc123" {
		t.Fatalf("missing wordlist metadata fields: %+v", first)
	}
}

func TestWordlistListRejectsUnsupportedQueryParams(t *testing.T) {
	store := &wordlistManagementStoreStub{}
	router := newWordlistManagementTestRouter(store)

	for _, path := range []string{
		"/v1/wordlists?page=2",
		"/v1/wordlists?sort=updatedAt",
		"/v1/wordlists?sortBy=updatedAt",
		"/v1/wordlists?sortOrder=desc",
		"/v1/wordlists?keyword=admin",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d body=%s", path, resp.Code, resp.Body.String())
		}
		if store.lastPage != 0 {
			t.Fatalf("unsupported query param %s must be rejected before store call, got page=%d", path, store.lastPage)
		}
	}
}

func TestWordlistListRejectsUnsupportedOrderBy(t *testing.T) {
	store := &wordlistManagementStoreStub{}
	router := newWordlistManagementTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/wordlists?orderBy=actions%20asc", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported orderBy, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.lastOrderBy != "" {
		t.Fatalf("unsupported orderBy must be rejected before store call, got %q", store.lastOrderBy)
	}
}

func TestWordlistListRejectsUnsupportedFilterField(t *testing.T) {
	store := &wordlistManagementStoreStub{}
	router := newWordlistManagementTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/wordlists?filter=actions%3D%3D%22delete%22", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported filter, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.lastFilter != "" {
		t.Fatalf("unsupported filter must be rejected before store call, got %q", store.lastFilter)
	}
}

func TestWordlistPatchUpdatesDescriptionAndTagsWithMask(t *testing.T) {
	store := &wordlistManagementStoreStub{
		byID: map[int]*catalogdomain.Wordlist{
			7: {ID: 7, FileName: "old.txt", Description: "old", Tags: []string{"old"}},
		},
	}
	router := newWordlistManagementTestRouter(store)
	body := `{"name":"wordlists/7","description":" updated ","tags":["fuzz"," custom "],"updateMask":"description,tags"}`

	req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.updated == nil {
		t.Fatalf("expected wordlist metadata update")
	}
	if store.updated.FileName != "old.txt" || store.updated.Description != "updated" {
		t.Fatalf("unexpected updated metadata: %+v", store.updated)
	}
	if strings.Join(store.updated.Tags, ",") != "custom,fuzz" {
		t.Fatalf("expected normalized tags, got %+v", store.updated.Tags)
	}
}

func TestWordlistPatchRejectsNameMismatchAndUnsupportedMask(t *testing.T) {
	store := &wordlistManagementStoreStub{byID: map[int]*catalogdomain.Wordlist{7: {ID: 7, FileName: "old.txt"}}}
	router := newWordlistManagementTestRouter(store)

	for _, body := range []string{
		`{"name":"wordlists/8","displayName":"new.txt","updateMask":"displayName"}`,
		`{"name":"wordlists/7","updateMask":"name"}`,
		`{"name":"wordlists/7","displayName":"new.txt","updateMask":"displayName"}`,
		`{"name":"wordlists/7","fileName":"new.txt","updateMask":"fileName"}`,
		`{"name":"wordlists/7","displayName":"new.txt","updateMask":"fileSize"}`,
		`{"name":"wordlists/7","displayName":"new.txt"}`,
	} {
		req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for body %s, got %d body=%s", body, resp.Code, resp.Body.String())
		}
	}
}

func TestWordlistTagSummaryListReturnsDerivedPaginatedResources(t *testing.T) {
	store := &wordlistManagementStoreStub{
		tagSummaries: []catalogdomain.WordlistTagSummary{{DisplayName: "fuzz", WordlistCount: 3}},
	}
	router := newWordlistManagementTestRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/wordlistTags?pageSize=10&filter=fu", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.lastSummarySize != 10 || store.lastSummaryQuery != "fu" {
		t.Fatalf("unexpected tag summary query: size=%d filter=%q", store.lastSummarySize, store.lastSummaryQuery)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	results := body["results"].([]any)
	first := results[0].(map[string]any)
	if first["name"] != "wordlistTags/fuzz" || first["displayName"] != "fuzz" || first["wordlistCount"] != float64(3) {
		t.Fatalf("unexpected tag summary response: %s", resp.Body.String())
	}
}
