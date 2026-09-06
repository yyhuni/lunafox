package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

type subfinderAPIKeySettingsHandlerStoreStub struct {
	settings *catalogdomain.SubfinderProviderSettings
	updated  *catalogdomain.SubfinderProviderSettings
	err      error
}

func (stub *subfinderAPIKeySettingsHandlerStoreStub) GetInstance() (*catalogdomain.SubfinderProviderSettings, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return stub.settings, nil
}

func (stub *subfinderAPIKeySettingsHandlerStoreStub) Update(settings *catalogdomain.SubfinderProviderSettings) error {
	stub.updated = settings
	stub.settings = settings
	return nil
}

func newSubfinderAPIKeySettingsHandlerForTest(store *subfinderAPIKeySettingsHandlerStoreStub) *SubfinderAPIKeySettingsHandler {
	return NewSubfinderAPIKeySettingsHandler(catalogapp.NewSubfinderAPIKeySettingsService(store))
}

func performSubfinderAPIKeySettingsRequest(t *testing.T, handler gin.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/v1/settings/apiKeys", handler)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestSubfinderAPIKeySettingsHandlerGetSettingsUsesRegistryMapContract(t *testing.T) {
	handler := newSubfinderAPIKeySettingsHandlerForTest(&subfinderAPIKeySettingsHandlerStoreStub{
		settings: &catalogdomain.SubfinderProviderSettings{Providers: catalogdomain.SubfinderProviderConfigs{
			"fofa":   {Enabled: true, Status: catalogdomain.SubfinderProviderStatusConfigured, Values: map[string]string{"email": "ops@example.com", "apiKey": "fofa-secret"}},
			"censys": {Enabled: true, Status: catalogdomain.SubfinderProviderStatusConfigured, Values: map[string]string{"pat": "censys-pat", "orgId": "org-1"}},
			"gitlab": {Enabled: true, Status: catalogdomain.SubfinderProviderStatusConfigured, Values: map[string]string{"apiKey": "legacy-secret"}},
		}},
	})

	resp := performSubfinderAPIKeySettingsRequest(t, handler.GetSettings, http.MethodGet, "/v1/settings/apiKeys", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Providers   map[string]map[string]any `json:"providers"`
		Definitions []map[string]any          `json:"definitions"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Definitions) != len(catalogdomain.SubfinderProviderDefinitions()) {
		t.Fatalf("expected registry definitions in response, got %d", len(body.Definitions))
	}
	if body.Providers["fofa"]["enabled"] != true || body.Providers["fofa"]["status"] != catalogdomain.SubfinderProviderStatusConfigured {
		t.Fatalf("expected fofa provider state, got %#v", body.Providers["fofa"])
	}
	if values, ok := body.Providers["fofa"]["values"].(map[string]any); ok && values["apiKey"] == "fofa-secret" {
		t.Fatalf("response must not expose plaintext secret values: %#v", values)
	}
	if _, ok := body.Providers["hunter"]; ok {
		t.Fatalf("response must not expose legacy hunter as active provider: %#v", body.Providers["hunter"])
	}
	if _, ok := body.Providers["gitlab"]; ok {
		t.Fatalf("response must not expose unregistered GitLab provider: %#v", body.Providers["gitlab"])
	}
	if _, ok := body.Providers["zoomeyeapi"]; !ok {
		t.Fatalf("response must expose zoomeyeapi registry key")
	}
	if strings.Contains(resp.Body.String(), "apiId") || strings.Contains(resp.Body.String(), "apiSecret") || strings.Contains(resp.Body.String(), "\"zoomeye\"") {
		t.Fatalf("response must not expose old fixed provider fields: %s", resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), "docs.projectdiscovery.io/tools/subfinder/install#provider-config-") {
		t.Fatalf("response must not synthesize generic subfinder provider docs fallback: %s", resp.Body.String())
	}
}

func TestSubfinderAPIKeySettingsHandlerUpdateSettingsRejectsUnsupportedProvider(t *testing.T) {
	handler := newSubfinderAPIKeySettingsHandlerForTest(&subfinderAPIKeySettingsHandlerStoreStub{})

	resp := performSubfinderAPIKeySettingsRequest(t, handler.UpdateSettings, http.MethodPatch, "/v1/settings/apiKeys", `{"providers":{"unknown":{"enabled":true,"values":{"apiKey":"secret"}}}}`)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestSubfinderAPIKeySettingsHandlerUpdateSettingsMapsRegistryValues(t *testing.T) {
	store := &subfinderAPIKeySettingsHandlerStoreStub{
		settings: &catalogdomain.SubfinderProviderSettings{Providers: catalogdomain.SubfinderProviderConfigs{
			"fofa": {Enabled: true, Values: map[string]string{"email": "old@example.com", "apiKey": "old"}},
		}},
	}
	handler := newSubfinderAPIKeySettingsHandlerForTest(store)

	resp := performSubfinderAPIKeySettingsRequest(
		t,
		handler.UpdateSettings,
		http.MethodPatch,
		"/v1/settings/apiKeys",
		`{"providers":{"fofa":{"enabled":true,"values":{"email":"ops@example.com","apiKey":"new-fofa"}},"censys":{"enabled":true,"values":{"pat":"censys-pat","orgId":"org-1"}}}}`,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.updated == nil {
		t.Fatalf("expected store update")
	}
	if store.updated.Providers["fofa"].Values["email"] != "ops@example.com" || store.updated.Providers["fofa"].Values["apiKey"] != "new-fofa" {
		t.Fatalf("expected fofa input mapped to domain, got %+v", store.updated.Providers["fofa"])
	}
	if store.updated.Providers["censys"].Values["pat"] != "censys-pat" || store.updated.Providers["censys"].Values["orgId"] != "org-1" {
		t.Fatalf("expected censys input mapped to domain, got %+v", store.updated.Providers["censys"])
	}
}
