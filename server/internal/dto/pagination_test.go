package dto

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type paginationQueryTest struct {
	PaginationQuery
}

func TestBindQueryRejectsInvalidPageToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/items?pageToken=not-base64", nil)

	var query paginationQueryTest
	if BindQuery(c, &query) {
		t.Fatal("expected invalid pageToken to fail binding")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPaginationQueryUsesTrimmedPageTokenConsistently(t *testing.T) {
	token := encodePageToken(2)
	query := PaginationQuery{PageToken: " " + token + " "}

	if err := query.ValidatePageToken(); err != nil {
		t.Fatalf("expected whitespace-wrapped pageToken to validate, got %v", err)
	}
	if got := query.GetPage(); got != 2 {
		t.Fatalf("expected GetPage to use trimmed pageToken, got %d", got)
	}
}
