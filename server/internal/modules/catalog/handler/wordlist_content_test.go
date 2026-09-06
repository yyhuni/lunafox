package handler

import (
	"bytes"
	"context"
	"encoding/json"
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

type wordlistContentStoreStub struct {
	wordlist catalogdomain.Wordlist
}

const fiveMiB = 5 * 1024 * 1024

func (stub *wordlistContentStoreStub) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	_ = page
	_ = pageSize
	_ = filter
	_ = orderBy
	return []catalogdomain.Wordlist{stub.wordlist}, 1, nil
}

func (stub *wordlistContentStoreStub) ListAll() ([]catalogdomain.Wordlist, error) {
	return []catalogdomain.Wordlist{stub.wordlist}, nil
}

func (stub *wordlistContentStoreStub) ListTagSummaries(page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error) {
	_ = page
	_ = pageSize
	_ = filter
	return []catalogdomain.WordlistTagSummary{}, 0, nil
}

func (stub *wordlistContentStoreStub) GetByID(id int) (*catalogdomain.Wordlist, error) {
	if stub.wordlist.ID == id {
		item := stub.wordlist
		return &item, nil
	}
	return nil, catalogapp.ErrWordlistNotFound
}

func (stub *wordlistContentStoreStub) GetByIDContext(_ context.Context, id int) (*catalogdomain.Wordlist, error) {
	return stub.GetByID(id)
}

func (stub *wordlistContentStoreStub) ExistsByFileName(fileName string, excludeID ...int) (bool, error) {
	_ = excludeID
	return stub.wordlist.FileName == fileName, nil
}

func (stub *wordlistContentStoreStub) Create(wordlist *catalogdomain.Wordlist) error {
	stub.wordlist = *wordlist
	return nil
}

func (stub *wordlistContentStoreStub) Update(wordlist *catalogdomain.Wordlist) error {
	wordlist.UpdatedAt = time.Date(2026, 5, 13, 9, 30, 0, 0, time.UTC)
	stub.wordlist = *wordlist
	return nil
}

func (stub *wordlistContentStoreStub) Delete(id int) error {
	_ = id
	return nil
}

func newWordlistContentTestRouter(t *testing.T, initialContent string) (*gin.Engine, *wordlistContentStoreStub) {
	t.Helper()
	return newWordlistContentTestRouterWithFileSize(t, initialContent, int64(len(initialContent)))
}

func newWordlistContentTestRouterWithFileSize(t *testing.T, initialContent string, fileSize int64) (*gin.Engine, *wordlistContentStoreStub) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "words.txt")
	if err := os.WriteFile(filePath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to write wordlist fixture: %v", err)
	}

	store := &wordlistContentStoreStub{
		wordlist: catalogdomain.Wordlist{
			ID:        7,
			FileName:  "words.txt",
			FilePath:  filePath,
			FileSize:  fileSize,
			LineCount: 2,
			UpdatedAt: time.Date(2026, 5, 13, 8, 0, 0, 0, time.UTC),
		},
	}
	fileStore := catalogapp.NewLocalWordlistFileStore()
	h := NewWordlistHandler(catalogapp.NewWordlistFacade(catalogapp.NewWordlistQueryService(store, fileStore), catalogapp.NewWordlistCommandService(store, baseDir, fileStore)))
	router := gin.New()
	router.GET("/v1/wordlists/:wordlist/text", h.GetContent)
	router.PATCH("/v1/wordlists/:wordlist/text", h.UpdateContent)
	return router, store
}

func TestWordlistTextGetReturnsSingletonResource(t *testing.T) {
	router, _ := newWordlistContentTestRouter(t, "one\ntwo\n")

	req := httptest.NewRequest(http.MethodGet, "/v1/wordlists/7/text", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if payload["name"] != "wordlists/7/text" {
		t.Fatalf("expected singleton resource name, got body=%s", resp.Body.String())
	}
	if payload["content"] != "one\ntwo\n" {
		t.Fatalf("expected text content, got body=%s", resp.Body.String())
	}
	if _, exists := payload["updateTime"].(string); !exists {
		t.Fatalf("expected updateTime in body=%s", resp.Body.String())
	}
	if _, exists := payload["id"]; exists {
		t.Fatalf("wordlist text response must not expose parent id: %s", resp.Body.String())
	}
}

func TestWordlistTextGetRejectsFilesAboveFiveMiB(t *testing.T) {
	router, _ := newWordlistContentTestRouter(t, oversizedWordlistContent())

	req := httptest.NewRequest(http.MethodGet, "/v1/wordlists/7/text", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized wordlist, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "max 5MB") {
		t.Fatalf("expected 5MB limit message, got body=%s", resp.Body.String())
	}
}

func TestWordlistTextPatchRejectsExistingFilesAboveFiveMiB(t *testing.T) {
	router, _ := newWordlistContentTestRouter(t, oversizedWordlistContent())

	body := []byte(`{"name":"wordlists/7/text","content":"updated\n"}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7/text?updateMask=content", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized wordlist, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "max 5MB") {
		t.Fatalf("expected 5MB limit message, got body=%s", resp.Body.String())
	}
}

func TestWordlistTextPatchRejectsContentAboveFiveMiB(t *testing.T) {
	router, _ := newWordlistContentTestRouter(t, "one\n")
	oversizedContent := strings.Repeat("a", fiveMiB+1)
	body, err := json.Marshal(map[string]string{
		"name":    "wordlists/7/text",
		"content": oversizedContent,
	})
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7/text?updateMask=content", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized content, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "max 5MB") {
		t.Fatalf("expected 5MB limit message, got body=%s", resp.Body.String())
	}
}

func oversizedWordlistContent() string {
	return strings.Repeat("word\n", fiveMiB/len("word\n")+1)
}

func TestWordlistTextPatchRequiresContentUpdateMask(t *testing.T) {
	router, _ := newWordlistContentTestRouter(t, "one\n")

	body := []byte(`{"name":"wordlists/7/text","content":"updated\n"}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7/text", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing updateMask, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestWordlistTextPatchRejectsMismatchedName(t *testing.T) {
	router, _ := newWordlistContentTestRouter(t, "one\n")

	body := []byte(`{"name":"wordlists/8/text","content":"updated\n"}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7/text?updateMask=content", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for mismatched name, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestWordlistTextPatchReturnsSingletonResource(t *testing.T) {
	router, store := newWordlistContentTestRouter(t, "one\n")

	body := []byte(`{"name":"wordlists/7/text","content":"updated\n"}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/wordlists/7/text?updateMask=content", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.wordlist.FileSize != int64(len("updated\n")) {
		t.Fatalf("expected file metadata update, got size=%d", store.wordlist.FileSize)
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if payload["name"] != "wordlists/7/text" || payload["content"] != "updated\n" {
		t.Fatalf("unexpected wordlist text response: %s", resp.Body.String())
	}
	if _, exists := payload["updateTime"].(string); !exists {
		t.Fatalf("expected updateTime in body=%s", resp.Body.String())
	}
	if _, exists := payload["displayName"]; exists {
		t.Fatalf("update response must not return parent wordlist representation: %s", resp.Body.String())
	}
}
