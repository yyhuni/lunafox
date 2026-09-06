package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/contracts/ociartifact"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	catalogdto "github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
)

func TestDecodeInstallEngineRequestIsStrict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	valid := `{"artifactRef":"registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","allowReplacement":false}`
	for _, test := range []struct {
		name, body string
		wantErr    bool
	}{
		{name: "valid", body: valid},
		{name: "missing replacement", body: `{"artifactRef":"registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`, wantErr: true},
		{name: "unknown", body: valid[:len(valid)-1] + `,"unknown":true}`, wantErr: true},
		{name: "multiple documents", body: valid + valid, wantErr: true},
		{name: "whitespace reference", body: `{"artifactRef":" registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","allowReplacement":false}`, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest(http.MethodPost, "/v1/engines:install", bytes.NewBufferString(test.body))
			context.Request.Header.Set("Content-Type", "application/json")
			request, err := decodeInstallEngineRequest(context)
			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.wantErr && (err != nil || request.AllowReplacement == nil || *request.AllowReplacement) {
				t.Fatalf("request=%#v error=%v", request, err)
			}
		})
	}
}

type engineCatalogHandlerStoreStub struct {
	items   []catalogapp.EngineCatalogItem
	listErr error
	getErr  error
}

type engineInstallerStub struct {
	record           *catalogdomain.Engine
	err              error
	allowReplacement bool
	calls            int
	context          context.Context
}

func (stub *engineInstallerStub) Install(ctx context.Context, _ ociartifact.ArtifactCandidates, allowReplacement bool) (*catalogdomain.Engine, error) {
	stub.calls++
	stub.context = ctx
	stub.allowReplacement = allowReplacement
	return stub.record, stub.err
}

func (stub *engineCatalogHandlerStoreStub) ListEngines(_ context.Context) ([]catalogapp.EngineCatalogItem, error) {
	return stub.items, stub.listErr
}

func (stub *engineCatalogHandlerStoreStub) GetEngineByID(_ context.Context, engineID string) (*catalogapp.EngineCatalogItem, error) {
	if stub.getErr != nil {
		return nil, stub.getErr
	}
	for i := range stub.items {
		if stub.items[i].EngineID == engineID {
			item := stub.items[i]
			return &item, nil
		}
	}
	return nil, catalogapp.ErrEngineNotFound
}

func TestEngineCatalogHandlerListAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &engineCatalogHandlerStoreStub{items: []catalogapp.EngineCatalogItem{{
		EngineID:             "engine.lunafox.website_discovery",
		ManifestVersion:      "engine.v5",
		Publisher:            "lunafox",
		EngineAPIMajor:       2,
		SupportedTargetTypes: []string{"domain", "ip", "cidr"},
	}}}
	handler := NewEngineCatalogHandler(catalogapp.NewEngineCatalogFacade(store))
	router := gin.New()
	router.GET("/v1/engines", handler.List)
	router.GET("/v1/engines/:engine", handler.GetByID)

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/v1/engines", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var list []catalogdto.EngineCatalogOutput
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list payload = %+v, %v", list, err)
	}

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, "/v1/engines/engine.lunafox.website_discovery", nil))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", getRecorder.Code, getRecorder.Body.String())
	}
}

func TestEngineCatalogHandlerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &engineCatalogHandlerStoreStub{listErr: errors.New("catalog unavailable")}
	handler := NewEngineCatalogHandler(catalogapp.NewEngineCatalogFacade(store))
	router := gin.New()
	router.GET("/v1/engines", handler.List)
	router.GET("/v1/engines/:engine", handler.GetByID)

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/v1/engines", nil))
	if listRecorder.Code != http.StatusInternalServerError {
		t.Fatalf("list status = %d", listRecorder.Code)
	}
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, httptest.NewRequest(http.MethodGet, "/v1/engines/missing", nil))
	if getRecorder.Code != http.StatusNotFound {
		t.Fatalf("get status = %d", getRecorder.Code)
	}
}

func TestEngineCatalogHandlerInstallMapsSuccessAndReplacementConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	item := catalogapp.EngineCatalogItem{EngineID: "engine.example.scanner", ManifestVersion: "engine.v5", Publisher: "example", PackageVersion: "1.0.0", ArtifactRef: "registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PackageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", EngineAPIMajor: 2, SupportedTargetTypes: []string{"domain"}, LocaleResources: map[string]map[string]any{}}
	store := &engineCatalogHandlerStoreStub{items: []catalogapp.EngineCatalogItem{item}}
	installer := &engineInstallerStub{record: &catalogdomain.Engine{EngineID: item.EngineID}}
	handler := NewEngineCatalogHandler(catalogapp.NewEngineCatalogFacade(store), installer)
	router := gin.New()
	router.POST("/v1/engines:install", handler.Install)
	body := `{"artifactRef":"registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","allowReplacement":true}`
	request := httptest.NewRequest(http.MethodPost, "/v1/engines:install", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !installer.allowReplacement {
		t.Fatalf("install status=%d allowReplacement=%v", response.Code, installer.allowReplacement)
	}
	installer.err = &catalogdomain.EngineReplacementConflictError{EngineID: item.EngineID, CurrentPackageDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ProposedPackageDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
	request = httptest.NewRequest(http.MethodPost, "/v1/engines:install", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", response.Code, response.Body.String())
	}
	var conflict struct {
		Error struct {
			Status  string `json:"status"`
			Details []any  `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &conflict); err != nil || conflict.Error.Status != "ALREADY_EXISTS" || len(conflict.Error.Details) != 4 {
		t.Fatalf("conflict payload=%s error=%v", response.Body.String(), err)
	}
}

func TestEngineCatalogHandlerInstallRejectsInvalidInputBeforeInstaller(t *testing.T) {
	gin.SetMode(gin.TestMode)
	item := catalogapp.EngineCatalogItem{EngineID: "engine.example.scanner"}
	store := &engineCatalogHandlerStoreStub{items: []catalogapp.EngineCatalogItem{item}}
	installer := &engineInstallerStub{record: &catalogdomain.Engine{EngineID: item.EngineID}}
	handler := NewEngineCatalogHandler(catalogapp.NewEngineCatalogFacade(store), installer)
	router := gin.New()
	router.POST("/v1/engines:install", handler.Install)

	validDigest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, test := range []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "missing artifact ref", contentType: "application/json", body: `{"allowReplacement":false}`},
		{name: "missing replacement", contentType: "application/json", body: `{"artifactRef":"registry.example/team/engine@` + validDigest + `"}`},
		{name: "unknown field", contentType: "application/json", body: `{"artifactRef":"registry.example/team/engine@` + validDigest + `","allowReplacement":false,"unknown":true}`},
		{name: "tag", contentType: "application/json", body: `{"artifactRef":"registry.example/team/engine:latest","allowReplacement":false}`},
		{name: "scheme", contentType: "application/json", body: `{"artifactRef":"https://registry.example/team/engine@` + validDigest + `","allowReplacement":false}`},
		{name: "plain HTTP", contentType: "application/json", body: `{"artifactRef":"http://registry.example/team/engine@` + validDigest + `","allowReplacement":false}`},
		{name: "wrong media type", contentType: "text/plain", body: `{"artifactRef":"registry.example/team/engine@` + validDigest + `","allowReplacement":false}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/engines:install", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	if installer.calls != 0 {
		t.Fatalf("installer calls=%d, want invalid requests to fail before installation", installer.calls)
	}
}

func TestEngineCatalogHandlerInstallPropagatesRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	item := catalogapp.EngineCatalogItem{EngineID: "engine.example.scanner"}
	store := &engineCatalogHandlerStoreStub{items: []catalogapp.EngineCatalogItem{item}}
	installer := &engineInstallerStub{record: &catalogdomain.Engine{EngineID: item.EngineID}}
	handler := NewEngineCatalogHandler(catalogapp.NewEngineCatalogFacade(store), installer)
	router := gin.New()
	router.POST("/v1/engines:install", handler.Install)

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, "/v1/engines:install", bytes.NewBufferString(`{"artifactRef":"registry.example/team/engine@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","allowReplacement":false}`)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	cancel()
	if installer.context != ctx {
		t.Fatal("installer did not receive the original request context")
	}
}
