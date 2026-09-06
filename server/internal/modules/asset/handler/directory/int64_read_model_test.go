package directory

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
)

func TestDirectoryReadModelsPreserveNullableInt64Values(t *testing.T) {
	gin.SetMode(gin.TestMode)

	zero := int64(0)
	max := int64(math.MaxInt64)
	store := &directoryStoreStub{
		items: []assetdomain.Directory{
			{ID: 1, TargetID: 1, URL: "https://example.com/unknown", CreatedAt: time.Unix(1, 0).UTC()},
			{ID: 2, TargetID: 1, URL: "https://example.com/zero-max", ContentLength: &zero, Duration: &max, CreatedAt: time.Unix(2, 0).UTC()},
			{ID: 3, TargetID: 1, URL: "https://example.com/max-zero", ContentLength: &max, Duration: &zero, CreatedAt: time.Unix(3, 0).UTC()},
		},
		total: 3,
		count: 3,
	}
	lookup := &directoryLookupStub{target: &assetdomain.TargetRef{ID: 1, Name: "example.com", Type: assetdomain.TargetTypeDomain}}
	handler := NewDirectoryHandler(service.NewDirectoryFacade(
		service.NewDirectoryQueryService(store, lookup),
		service.NewDirectoryCommandService(store, lookup),
	))

	t.Run("HTTP emits strings or null", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.List, http.MethodGet, "/v1/targets/1/directories", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var response struct {
			Results []struct {
				ContentLength json.RawMessage `json:"contentLength"`
				Duration      json.RawMessage `json:"duration"`
			} `json:"results"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		assertDirectoryRawInt64Fields(t, response.Results, [][2]string{
			{"null", "null"},
			{`"0"`, `"9223372036854775807"`},
			{`"9223372036854775807"`, `"0"`},
		})
	})

	t.Run("CSV emits exact decimals or empty fields", func(t *testing.T) {
		recorder := performDirectoryRequest(t, handler.Export, http.MethodGet, "/v1/targets/1/directories/exportFiles/current", gin.Params{{Key: "target", Value: "1"}}, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(recorder.Body.String(), "\ufeff"))).ReadAll()
		if err != nil {
			t.Fatalf("decode CSV: %v", err)
		}
		if len(rows) != 4 {
			t.Fatalf("expected header plus 3 rows, got %d", len(rows))
		}
		assertDirectoryCSVInt64Fields(t, rows[1:], [][2]string{
			{"", ""},
			{"0", "9223372036854775807"},
			{"9223372036854775807", "0"},
		})
	})
}

func assertDirectoryRawInt64Fields(t *testing.T, rows []struct {
	ContentLength json.RawMessage `json:"contentLength"`
	Duration      json.RawMessage `json:"duration"`
}, expected [][2]string) {
	t.Helper()
	if len(rows) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(rows))
	}
	for index, row := range rows {
		if string(row.ContentLength) != expected[index][0] || string(row.Duration) != expected[index][1] {
			t.Fatalf("row %d: contentLength=%s duration=%s", index, row.ContentLength, row.Duration)
		}
	}
}

func assertDirectoryCSVInt64Fields(t *testing.T, rows [][]string, expected [][2]string) {
	t.Helper()
	if len(rows) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(rows))
	}
	for index, row := range rows {
		if len(row) != 8 {
			t.Fatalf("row %d: expected 8 columns, got %d", index, len(row))
		}
		if row[4] != expected[index][0] || row[6] != expected[index][1] {
			t.Fatalf("row %d: content_length=%q duration=%q", index, row[4], row[6])
		}
	}
}
