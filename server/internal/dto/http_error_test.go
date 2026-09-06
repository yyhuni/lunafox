package dto

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteContextErrorClassifiesWrappedContextErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantRPC    string
		wantReason string
		wantWrite  bool
	}{
		{name: "wrapped cancellation", err: errors.Join(errors.New("outer"), context.Canceled), wantStatus: 499, wantRPC: "CANCELLED", wantReason: "CANCELLED", wantWrite: true},
		{name: "wrapped deadline", err: errors.Join(errors.New("outer"), context.DeadlineExceeded), wantStatus: http.StatusGatewayTimeout, wantRPC: "DEADLINE_EXCEEDED", wantReason: "DEADLINE_EXCEEDED", wantWrite: true},
		{name: "internal error", err: errors.New("boom"), wantWrite: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			if got := WriteContextError(ctx, test.err); got != test.wantWrite {
				t.Fatalf("WriteContextError() = %t, want %t", got, test.wantWrite)
			}
			if !test.wantWrite {
				if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
					t.Fatalf("unclassified error wrote a response: status=%d body=%q", recorder.Code, recorder.Body.String())
				}
				return
			}
			var body ErrorResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != test.wantStatus || body.Error.Code != test.wantStatus || body.Error.Status != test.wantRPC || len(body.Error.Details) != 1 {
				t.Fatalf("unexpected response: status=%d body=%+v", recorder.Code, body)
			}
			encoded, _ := json.Marshal(body.Error.Details[0])
			var info ErrorInfo
			if err := json.Unmarshal(encoded, &info); err != nil {
				t.Fatal(err)
			}
			if info.Reason != test.wantReason {
				t.Fatalf("reason = %q, want %q", info.Reason, test.wantReason)
			}
		})
	}
}

func TestErrorWithStatusAndTypedDetailsOverridesCanonicalStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ErrorWithStatusAndTypedDetails(
		ctx,
		http.StatusBadRequest,
		"ENGINE_CONFIG_RESOURCE_UNAVAILABLE",
		"FAILED_PRECONDITION",
		"Selected Engine configuration resource is unavailable",
		map[string]string{"field": "configuration.field", "resourceKind": "wordlist", "resourceName": "dns.txt"},
	)
	var body ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusBadRequest || body.Error.Code != http.StatusBadRequest || body.Error.Status != "FAILED_PRECONDITION" || len(body.Error.Details) != 1 {
		t.Fatalf("unexpected response: status=%d body=%+v", recorder.Code, body)
	}
	encoded, _ := json.Marshal(body.Error.Details[0])
	var info ErrorInfo
	if err := json.Unmarshal(encoded, &info); err != nil {
		t.Fatal(err)
	}
	if info.Type != "type.googleapis.com/google.rpc.ErrorInfo" || info.Reason != "ENGINE_CONFIG_RESOURCE_UNAVAILABLE" || info.Metadata["resourceName"] != "dns.txt" {
		t.Fatalf("unexpected ErrorInfo: %+v", info)
	}
}
