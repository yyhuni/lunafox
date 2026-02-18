package pkg

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		name       string
		totalCount int64
		pageSize   int
		expected   int
	}{
		{name: "zero count", totalCount: 0, pageSize: 10, expected: 0},
		{name: "single page exact", totalCount: 10, pageSize: 10, expected: 1},
		{name: "single page partial", totalCount: 1, pageSize: 10, expected: 1},
		{name: "multiple pages", totalCount: 11, pageSize: 10, expected: 2},
		{name: "invalid page size zero", totalCount: 10, pageSize: 0, expected: 0},
		{name: "invalid page size negative", totalCount: 10, pageSize: -1, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateTotalPages(tt.totalCount, tt.pageSize); got != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestResponseHelpers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type testCase struct {
		name           string
		call           func(*gin.Context)
		expectedStatus int
		expectedOK     bool
		expectedCode   string
		checkDetails   bool
	}

	cases := []testCase{
		{
			name: "ok",
			call: func(c *gin.Context) {
				OK(c, map[string]string{"hello": "world"})
			},
			expectedStatus: 200,
			expectedOK:     true,
		},
		{
			name: "ok with meta",
			call: func(c *gin.Context) {
				OKWithMeta(c, []string{"a"}, &Meta{Page: 1, PageSize: 10, TotalCount: 11, TotalPages: 2})
			},
			expectedStatus: 200,
			expectedOK:     true,
		},
		{
			name: "created",
			call: func(c *gin.Context) {
				Created(c, "id")
			},
			expectedStatus: 201,
			expectedOK:     true,
		},
		{
			name: "bad request",
			call: func(c *gin.Context) {
				BadRequest(c, "bad")
			},
			expectedStatus: 400,
			expectedCode:   "BAD_REQUEST",
		},
		{
			name: "unauthorized",
			call: func(c *gin.Context) {
				Unauthorized(c, "nope")
			},
			expectedStatus: 401,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name: "forbidden",
			call: func(c *gin.Context) {
				Forbidden(c, "nope")
			},
			expectedStatus: 403,
			expectedCode:   "FORBIDDEN",
		},
		{
			name: "not found",
			call: func(c *gin.Context) {
				NotFound(c, "missing")
			},
			expectedStatus: 404,
			expectedCode:   "NOT_FOUND",
		},
		{
			name: "internal error",
			call: func(c *gin.Context) {
				InternalError(c, "boom")
			},
			expectedStatus: 500,
			expectedCode:   "INTERNAL_ERROR",
		},
		{
			name: "validation error",
			call: func(c *gin.Context) {
				ValidationError(c, "bad", "detail")
			},
			expectedStatus: 422,
			expectedCode:   "VALIDATION_ERROR",
			checkDetails:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tc.call(c)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			var resp Response
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("unmarshal response failed: %v", err)
			}

			if tc.expectedOK && !resp.Success {
				t.Fatalf("expected success response")
			}
			if !tc.expectedOK && resp.Success {
				t.Fatalf("expected error response")
			}
			if tc.expectedCode != "" {
				if resp.Error == nil {
					t.Fatalf("expected error payload")
				}
				if resp.Error.Code != tc.expectedCode {
					t.Fatalf("expected error code %q, got %q", tc.expectedCode, resp.Error.Code)
				}
				if tc.checkDetails && resp.Error.Details == "" {
					t.Fatalf("expected error details to be set")
				}
			}
		})
	}
}

func TestNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	NoContent(c)

	if c.Writer.Status() != 204 {
		t.Fatalf("expected status 204, got %d", c.Writer.Status())
	}
	c.Writer.WriteHeaderNow()
	if w.Code != 204 {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty response body")
	}
}
