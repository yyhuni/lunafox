package httpdto

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
)

func TestPaginated(t *testing.T) {
	t.Run("wraps paginated success responses", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)

		Paginated(ctx, []string{"alpha", "beta"}, 4, 2, 2)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var response PaginatedResponse[string]
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("unmarshal paginated response: %v", err)
		}
		if response.TotalSize != 4 || response.NextPageToken != "" {
			t.Fatalf("unexpected pagination metadata: %+v", response)
		}
		if len(response.Results) != 2 || response.Results[0] != "alpha" || response.Results[1] != "beta" {
			t.Fatalf("unexpected results: %+v", response.Results)
		}
	})

	t.Run("applies shared pagination defaults for boundary values", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)

		Paginated[string](ctx, nil, -1, 0, 5000)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var response PaginatedResponse[string]
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("unmarshal paginated response: %v", err)
		}
		if response.TotalSize != 0 || response.NextPageToken != "" {
			t.Fatalf("unexpected boundary pagination metadata: %+v", response)
		}
		if response.Results == nil || len(response.Results) != 0 {
			t.Fatalf("expected empty results slice, got %+v", response.Results)
		}
	})
}

func TestWriteWorkflowConfigurationErrorUsesStableDynamicFieldViolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	err := errors.New("outer")
	validation := &workflowconfig.ValidationError{Violation: workflowconfig.Violation{
		Path:    `configuration.steps["port-scan"].enabled`,
		Reason:  workflowconfig.ReasonFieldRequired,
		Message: "enabled is required",
	}}
	err = errors.Join(err, validation)
	if !WriteWorkflowConfigurationError(ctx, err) {
		t.Fatal("expected typed workflow configuration error to be mapped")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body struct {
		Error struct {
			Status  string            `json:"status"`
			Details []json.RawMessage `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Status != "INVALID_ARGUMENT" || len(body.Error.Details) != 2 {
		t.Fatalf("unexpected error envelope: %+v", body.Error)
	}
	var info struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body.Error.Details[0], &info); err != nil || info.Reason != "WORKFLOW_CONFIGURATION_INVALID" {
		t.Fatalf("unexpected ErrorInfo: %s", body.Error.Details[0])
	}
	var badRequest struct {
		Type            string `json:"@type"`
		FieldViolations []struct {
			Field  string `json:"field"`
			Reason string `json:"reason"`
		} `json:"fieldViolations"`
	}
	if err := json.Unmarshal(body.Error.Details[1], &badRequest); err != nil {
		t.Fatal(err)
	}
	if badRequest.Type != "type.googleapis.com/google.rpc.BadRequest" || len(badRequest.FieldViolations) != 1 || badRequest.FieldViolations[0].Field != `configuration.steps["port-scan"].enabled` || badRequest.FieldViolations[0].Reason != "FIELD_REQUIRED" {
		t.Fatalf("unexpected BadRequest detail: %+v", badRequest)
	}
}
