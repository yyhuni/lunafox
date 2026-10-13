package search

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

type exportGrantStoreStub struct {
	grant *service.GlobalAssetExportGrant
}

func (stub *exportGrantStoreStub) Issue(context.Context, service.GlobalAssetExportGrant, time.Duration) (string, error) {
	return "opaque-token", nil
}
func (stub *exportGrantStoreStub) Consume(context.Context, string) (*service.GlobalAssetExportGrant, error) {
	if stub.grant == nil {
		return nil, service.ErrGlobalAssetExportGrant
	}
	grant := stub.grant
	stub.grant = nil
	return grant, nil
}

type exportRunnerStub struct{}

func (exportRunnerStub) Export(_ context.Context, _ service.GlobalAssetSearchAST, _ service.GlobalAssetSearchAssetType, begin func(int64) (io.Writer, error)) error {
	writer, err := begin(3)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte("ok\n"))
	return err
}

type exportTokenVersionStub struct{}

func (exportTokenVersionStub) GetTokenVersion(context.Context, int) (int, error) { return 4, nil }

func TestGlobalAssetExportIssueBindsAuthenticatedGrant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &exportGrantStoreStub{}
	handler := NewGlobalAssetExportHandler(service.NewGlobalAssetExportService(store), exportRunnerStub{}, exportTokenVersionStub{})
	engine := gin.New()
	manager := auth.NewJWTManager("test-export-secret-key-32-chars!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, exportTokenVersionStub{}))
	engine.POST("/v1/assets:export", handler.Issue)
	token, _, err := manager.GenerateAccessToken(7, "alice", 4)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/assets:export", strings.NewReader(`{"q":"example","assetType":"website"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "downloadToken=opaque-token") {
		t.Fatalf("issue response = %d %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "Authorization") {
		t.Fatal("primary JWT must not be included in download URL")
	}
}

func TestGlobalAssetExportDownloadConsumesGrantAndSetsLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	grant := &service.GlobalAssetExportGrant{UserID: 7, TokenVersion: 4, AssetType: service.GlobalAssetSearchAssetTypeWebsite, AST: service.GlobalAssetSearchAST{Mode: service.GlobalAssetSearchModePlainURL, PlainURL: "example"}, ExpiresAt: time.Now().Add(time.Minute)}
	store := &exportGrantStoreStub{grant: grant}
	handler := NewGlobalAssetExportHandler(service.NewGlobalAssetExportService(store), exportRunnerStub{}, exportTokenVersionStub{})
	engine := gin.New()
	engine.GET("/v1/assets/exportFiles/current", handler.Download)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/assets/exportFiles/current?downloadToken=opaque-token", nil))
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Length") != "3" || recorder.Body.String() != "ok\n" {
		t.Fatalf("download response = %d headers=%v body=%q", recorder.Code, recorder.Header(), recorder.Body.String())
	}
	if recorder.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatal("download must disable proxy buffering")
	}
}

func TestGlobalAssetExportDownloadOverHTTPKeepsContentLengthAndBodyAligned(t *testing.T) {
	gin.SetMode(gin.TestMode)
	grant := &service.GlobalAssetExportGrant{
		UserID: 7, TokenVersion: 4,
		AssetType: service.GlobalAssetSearchAssetTypeEndpoint,
		AST:       service.GlobalAssetSearchAST{Mode: service.GlobalAssetSearchModePlainURL, PlainURL: "example"},
		ExpiresAt: time.Now().Add(time.Minute),
	}
	store := &exportGrantStoreStub{grant: grant}
	handler := NewGlobalAssetExportHandler(service.NewGlobalAssetExportService(store), exportRunnerStub{}, exportTokenVersionStub{})
	engine := gin.New()
	engine.GET("/v1/assets/exportFiles/current", handler.Download)
	server := httptest.NewServer(engine)
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/v1/assets/exportFiles/current?downloadToken=opaque-token")
	if err != nil {
		t.Fatalf("download request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read download response: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("download status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Length"); got != "3" {
		t.Fatalf("Content-Length = %q, want 3", got)
	}
	if len(body) != 3 || string(body) != "ok\n" {
		t.Fatalf("download body = %q, want %q", body, "ok\n")
	}
}

func TestGlobalAssetExportDownloadRejectsReplay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &exportGrantStoreStub{}
	handler := NewGlobalAssetExportHandler(service.NewGlobalAssetExportService(store), exportRunnerStub{}, exportTokenVersionStub{})
	engine := gin.New()
	engine.GET("/v1/assets/exportFiles/current", handler.Download)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/assets/exportFiles/current?downloadToken=opaque-token", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("replayed grant status = %d", recorder.Code)
	}
}
