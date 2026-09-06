package httpdto

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type configResourceErrorViewStub struct {
	cause string
	field string
	kind  string
	name  string
	err   error
}

func (stub *configResourceErrorViewStub) Error() string                         { return "safe config resource error" }
func (stub *configResourceErrorViewStub) Unwrap() error                         { return stub.err }
func (stub *configResourceErrorViewStub) ConfigResourceValidationCause() string { return stub.cause }
func (stub *configResourceErrorViewStub) ConfigResourceValidationField() string { return stub.field }
func (stub *configResourceErrorViewStub) ConfigResourceValidationKind() string  { return stub.kind }
func (stub *configResourceErrorViewStub) ConfigResourceValidationName() string  { return stub.name }

func TestWriteConfigResourceValidationErrorUsesExactSafeContract(t *testing.T) {
	tests := []struct {
		name       string
		cause      string
		statusCode int
		rpcStatus  string
		reason     string
	}{
		{name: "unavailable", cause: configResourceUnavailableCause, statusCode: http.StatusBadRequest, rpcStatus: "FAILED_PRECONDITION", reason: "ENGINE_CONFIG_RESOURCE_UNAVAILABLE"},
		{name: "validation unavailable", cause: configResourceValidationUnavailableCause, statusCode: http.StatusServiceUnavailable, rpcStatus: "UNAVAILABLE", reason: "ENGINE_CONFIG_RESOURCE_VALIDATION_UNAVAILABLE"},
		{name: "internal", cause: configResourceInternalCause, statusCode: http.StatusInternalServerError, rpcStatus: "INTERNAL", reason: "INTERNAL_ERROR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			private := errors.New("open /srv/private/wordlists/dns.txt: permission denied")
			view := &configResourceErrorViewStub{
				cause: test.cause,
				field: `configuration.steps["step"].engineConfig.scan.wordlist`,
				kind:  "wordlist", name: "dns.txt", err: private,
			}
			if !WriteConfigResourceValidationError(ctx, fmt.Errorf("wrapped: %w", view)) {
				t.Fatal("expected config resource error to be handled")
			}
			var body struct {
				Error struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Status  string `json:"status"`
					Details []struct {
						Type     string            `json:"@type"`
						Reason   string            `json:"reason"`
						Metadata map[string]string `json:"metadata"`
					} `json:"details"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != test.statusCode || body.Error.Code != test.statusCode || body.Error.Status != test.rpcStatus || len(body.Error.Details) != 1 || body.Error.Details[0].Reason != test.reason {
				t.Fatalf("unexpected response: status=%d body=%+v", recorder.Code, body)
			}
			if strings.Contains(recorder.Body.String(), "/srv/private") || strings.Contains(recorder.Body.String(), "permission denied") {
				t.Fatalf("response leaked private diagnostic: %s", recorder.Body.String())
			}
			if test.cause == configResourceInternalCause {
				if len(body.Error.Details[0].Metadata) != 0 || strings.Contains(recorder.Body.String(), "wordlist") {
					t.Fatalf("internal response exposed resource metadata: %s", recorder.Body.String())
				}
				return
			}
			wantMetadata := map[string]string{
				"field":        `configuration.steps["step"].engineConfig.scan.wordlist`,
				"resourceKind": "wordlist", "resourceName": "dns.txt",
			}
			if fmt.Sprint(body.Error.Details[0].Metadata) != fmt.Sprint(wantMetadata) {
				t.Fatalf("metadata = %#v, want %#v", body.Error.Details[0].Metadata, wantMetadata)
			}
			if test.cause == configResourceValidationUnavailableCause && (strings.Contains(strings.ToLower(body.Error.Message), "absent") || strings.Contains(strings.ToLower(body.Error.Message), "another")) {
				t.Fatalf("transient response gave misleading replacement advice: %q", body.Error.Message)
			}
		})
	}
}

func TestWriteConfigResourceValidationErrorIgnoresUnrelatedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	if WriteConfigResourceValidationError(ctx, errors.New("ordinary failure")) {
		t.Fatal("unrelated error was handled")
	}
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("writer mutated response for unrelated error: %d %s", recorder.Code, recorder.Body.String())
	}
}
