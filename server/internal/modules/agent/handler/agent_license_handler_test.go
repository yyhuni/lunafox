package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
)

type agentLicenseHandlerServiceStub struct {
	license agentapp.AgentLicense
	err     error
	key     string
	calls   int
}

func (stub *agentLicenseHandlerServiceStub) Activate(_ context.Context, key string) (agentapp.AgentLicense, error) {
	stub.key = key
	stub.calls++
	return stub.license, stub.err
}

func callAgentLicenseHandler(t *testing.T, service *agentLicenseHandlerServiceStub, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/admin/agentLicenses/:licenseAction", NewAgentLicenseHandler(service).Activate)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAgentLicenseHandlerReturnsEntitlementWithoutKey(t *testing.T) {
	expiresAt := time.Date(2026, 12, 19, 12, 0, 0, 0, time.UTC)
	months := 2
	id := "71c18456-e0b0-434d-97bf-a9790176b93c"
	service := &agentLicenseHandlerServiceStub{license: agentapp.AgentLicense{State: "active", AgentLimit: 10, Months: &months, ExpiresAt: &expiresAt, LicenseID: &id}}
	recorder := callAgentLicenseHandler(t, service, "/v1/admin/agentLicenses/current:activate", `{"key":"sensitive-signed-key"}`)
	if recorder.Code != http.StatusOK || service.key != "sensitive-signed-key" || service.calls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", recorder.Code, service.calls, recorder.Body.String())
	}
	var response dto.AgentLicenseResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Name != "agentLicenses/current" || response.State != "active" || response.AgentLimit != 10 || *response.Months != 2 || !response.ExpiresAt.Equal(expiresAt) || *response.LicenseID != id {
		t.Fatalf("response=%#v", response)
	}
	if strings.Contains(recorder.Body.String(), "sensitive-signed-key") {
		t.Fatal("Key leaked in response")
	}
}

func TestAgentLicenseHandlerUsesCanonicalSafeErrors(t *testing.T) {
	for _, sample := range []struct {
		name              string
		err               error
		status            int
		rpcStatus, reason string
	}{
		{"required", agentdomain.ErrAgentLicenseKeyRequired, 400, "INVALID_ARGUMENT", "LICENSE_KEY_REQUIRED"},
		{"invalid", agentdomain.ErrAgentLicenseKeyInvalid, 400, "INVALID_ARGUMENT", "LICENSE_KEY_INVALID"},
		{"expired", agentdomain.ErrAgentLicenseKeyExpired, 400, "INVALID_ARGUMENT", "LICENSE_KEY_EXPIRED"},
		{"future", agentdomain.ErrAgentLicenseKeyNotYetValid, 400, "INVALID_ARGUMENT", "LICENSE_KEY_NOT_YET_VALID"},
		{"storage", errors.New("secret database details"), 500, "INTERNAL", "INTERNAL_ERROR"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			service := &agentLicenseHandlerServiceStub{err: sample.err}
			recorder := callAgentLicenseHandler(t, service, "/v1/admin/agentLicenses/current:activate", `{"key":"sensitive-signed-key"}`)
			var body struct {
				Error struct {
					Code    int    `json:"code"`
					Status  string `json:"status"`
					Details []struct {
						Type   string `json:"@type"`
						Reason string `json:"reason"`
						Domain string `json:"domain"`
					} `json:"details"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != sample.status || body.Error.Code != sample.status || body.Error.Status != sample.rpcStatus || len(body.Error.Details) != 1 || body.Error.Details[0].Reason != sample.reason || body.Error.Details[0].Type != "type.googleapis.com/google.rpc.ErrorInfo" {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if sample.status == 400 && body.Error.Details[0].Domain != "lunafox.agentLicense" {
				t.Fatalf("domain=%s", body.Error.Details[0].Domain)
			}
			if strings.Contains(recorder.Body.String(), "sensitive-signed-key") || strings.Contains(recorder.Body.String(), "secret database details") {
				t.Fatal("internal data exposed")
			}
		})
	}
}

func TestAgentLicenseHandlerRejectsWrongActionsAndMalformedBodies(t *testing.T) {
	for _, sample := range []struct {
		name, path, body string
		status           int
	}{
		{"unknown-verb", "/v1/admin/agentLicenses/current:revoke", `{"key":"value"}`, 404},
		{"unknown-resource", "/v1/admin/agentLicenses/another:activate", `{"key":"value"}`, 404},
		{"unknown-field", "/v1/admin/agentLicenses/current:activate", `{"key":"value","agentLimit":99}`, 400},
		{"bad-type", "/v1/admin/agentLicenses/current:activate", `{"key":123}`, 400},
		{"trailing", "/v1/admin/agentLicenses/current:activate", `{"key":"value"} {}`, 400},
		{"oversize", "/v1/admin/agentLicenses/current:activate", `{"key":"` + strings.Repeat("x", 9000) + `"}`, 400},
	} {
		t.Run(sample.name, func(t *testing.T) {
			service := &agentLicenseHandlerServiceStub{}
			recorder := callAgentLicenseHandler(t, service, sample.path, sample.body)
			if recorder.Code != sample.status || service.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", recorder.Code, service.calls, recorder.Body.String())
			}
		})
	}
}
