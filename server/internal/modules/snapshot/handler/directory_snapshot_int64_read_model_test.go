package handler

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshotdomain "github.com/yyhuni/lunafox/server/internal/modules/snapshot/domain"
)

func TestDirectorySnapshotReadModelsPreserveNullableInt64Values(t *testing.T) {
	gin.SetMode(gin.TestMode)

	zero := int64(0)
	max := int64(math.MaxInt64)
	store := &directorySnapshotHandlerStoreStub{
		items: []snapshotdomain.DirectorySnapshot{
			{ID: 1, ScanID: 1, URL: "https://example.com/unknown", CreatedAt: time.Unix(1, 0).UTC()},
			{ID: 2, ScanID: 1, URL: "https://example.com/zero-max", ContentLength: &zero, Duration: &max, CreatedAt: time.Unix(2, 0).UTC()},
			{ID: 3, ScanID: 1, URL: "https://example.com/max-zero", ContentLength: &max, Duration: &zero, CreatedAt: time.Unix(3, 0).UTC()},
		},
		total: 3,
	}
	handler := NewDirectorySnapshotHandler(service.NewDirectorySnapshotFacade(
		service.NewDirectorySnapshotQueryService(store, &directorySnapshotHandlerLookupStub{}),
		nil,
	))

	t.Run("HTTP emits strings or null", func(t *testing.T) {
		recorder := performDirectorySnapshotListRequest(handler, "/v1/scans/1/directories")
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
		if len(response.Results) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(response.Results))
		}
		expected := [][2]string{
			{"null", "null"},
			{`"0"`, `"9223372036854775807"`},
			{`"9223372036854775807"`, `"0"`},
		}
		for index, row := range response.Results {
			if string(row.ContentLength) != expected[index][0] || string(row.Duration) != expected[index][1] {
				t.Fatalf("row %d: contentLength=%s duration=%s", index, row.ContentLength, row.Duration)
			}
		}
	})

	t.Run("CSV emits exact decimals or empty fields", func(t *testing.T) {
		recorder := performDirectorySnapshotExportRequest(handler, "/v1/scans/1/directories/exportFiles/current")
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
		expected := [][2]string{
			{"", ""},
			{"0", "9223372036854775807"},
			{"9223372036854775807", "0"},
		}
		for index, row := range rows[1:] {
			if len(row) != 8 {
				t.Fatalf("row %d: expected 8 columns, got %d", index, len(row))
			}
			if row[4] != expected[index][0] || row[6] != expected[index][1] {
				t.Fatalf("row %d: content_length=%q duration=%q", index, row[4], row[6])
			}
		}
	})
}

func performDirectorySnapshotExportRequest(handler *DirectorySnapshotHandler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Params = gin.Params{{Key: "scan", Value: "1"}}
	handler.Export(c)
	return recorder
}
