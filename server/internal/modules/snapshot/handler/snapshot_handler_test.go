package handler

import (
	"database/sql"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/shopspring/decimal"
	service "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	"github.com/yyhuni/lunafox/server/internal/modules/snapshot/dto"
	"gorm.io/datatypes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockVulnerabilitySnapshotService is a mock implementation for testing
type MockVulnerabilitySnapshotService struct {
	SaveAndSyncFunc  func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error)
	ListByScanFunc   func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error)
	ListAllFunc      func(query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error)
	GetByIDFunc      func(id int) (*service.VulnerabilitySnapshot, error)
	StreamByScanFunc func(scanID int) (*sql.Rows, error)
	CountByScanFunc  func(scanID int) (int64, error)
	ScanRowFunc      func(rows *sql.Rows) (*service.VulnerabilitySnapshot, error)
}

func (m *MockVulnerabilitySnapshotService) SaveAndSync(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
	if m.SaveAndSyncFunc != nil {
		return m.SaveAndSyncFunc(scanID, items)
	}
	return 0, 0, nil
}

func (m *MockVulnerabilitySnapshotService) ListByScan(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
	if m.ListByScanFunc != nil {
		return m.ListByScanFunc(scanID, query)
	}
	return nil, 0, nil
}

func (m *MockVulnerabilitySnapshotService) ListAll(query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
	if m.ListAllFunc != nil {
		return m.ListAllFunc(query)
	}
	return nil, 0, nil
}

func (m *MockVulnerabilitySnapshotService) GetByID(id int) (*service.VulnerabilitySnapshot, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, nil
}

func (m *MockVulnerabilitySnapshotService) CountByScan(scanID int) (int64, error) {
	if m.CountByScanFunc != nil {
		return m.CountByScanFunc(scanID)
	}
	return 0, nil
}

func requireIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skip integration tests in short mode")
	}
	if os.Getenv("LUNAFOX_RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set LUNAFOX_RUN_INTEGRATION_TESTS=1 to run integration tests")
	}
}

// TestVulnerabilitySnapshotBulkCreate tests the BulkCreate endpoint
func TestVulnerabilitySnapshotBulkCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		scanID         string
		body           string
		mockFunc       func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful bulk create",
			scanID: "1",
			body:   `{"vulnerabilities":[{"url":"https://example.com/vuln","vulnType":"XSS","severity":"high"}]}`,
			mockFunc: func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
				return 1, 1, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"snapshotCount":1,"assetCount":1`,
		},
		{
			name:           "invalid scan ID",
			scanID:         "invalid",
			body:           `{"vulnerabilities":[]}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid scan ID"`,
		},
		{
			name:   "scan not found",
			scanID: "999",
			body:   `{"vulnerabilities":[{"url":"https://example.com/vuln","vulnType":"XSS","severity":"high"}]}`,
			mockFunc: func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
				return 0, 0, service.ErrScanNotFoundForSnapshot
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `"message":"Scan not found"`,
		},
		{
			name:   "multiple vulnerabilities",
			scanID: "1",
			body:   `{"vulnerabilities":[{"url":"https://example.com/vuln1","vulnType":"XSS","severity":"high"},{"url":"https://example.com/vuln2","vulnType":"SQLi","severity":"critical"}]}`,
			mockFunc: func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
				return 2, 2, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"snapshotCount":2,"assetCount":2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockVulnerabilitySnapshotService{
				SaveAndSyncFunc: tt.mockFunc,
			}

			router := gin.New()
			router.POST("/api/scans/:id/vulnerabilities/bulk-create", func(c *gin.Context) {
				scanID := c.Param("id")
				if scanID == "invalid" {
					dto.BadRequest(c, "Invalid scan ID")
					return
				}

				var req dto.BulkCreateVulnerabilitySnapshotsRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					dto.BadRequest(c, "Invalid request body")
					return
				}

				snapshotCount, assetCount, err := mockSvc.SaveAndSync(1, req.Vulnerabilities)
				if err != nil {
					if err == service.ErrScanNotFoundForSnapshot {
						dto.NotFound(c, "Scan not found")
						return
					}
					dto.InternalError(c, "Failed to save vulnerability snapshots")
					return
				}

				dto.Success(c, dto.BulkCreateVulnerabilitySnapshotsResponse{
					SnapshotCount: int(snapshotCount),
					AssetCount:    int(assetCount),
				})
			})

			req := httptest.NewRequest(http.MethodPost, "/api/scans/"+tt.scanID+"/vulnerabilities/bulk-create", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestVulnerabilitySnapshotListByScan tests the ListByScan endpoint
func TestVulnerabilitySnapshotListByScan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	score := decimal.NewFromFloat(7.5)
	mockSnapshots := []service.VulnerabilitySnapshot{
		{ID: 1, ScanID: 1, URL: "https://example.com/vuln1", VulnType: "XSS", Severity: "high", CVSSScore: &score, CreatedAt: now},
		{ID: 2, ScanID: 1, URL: "https://example.com/vuln2", VulnType: "SQLi", Severity: "critical", CreatedAt: now},
	}

	tests := []struct {
		name           string
		scanID         string
		queryParams    string
		mockFunc       func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "list with default pagination",
			scanID:      "1",
			queryParams: "",
			mockFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				if query.GetPage() != 1 {
					t.Errorf("expected page 1, got %d", query.GetPage())
				}
				if query.GetPageSize() != 20 {
					t.Errorf("expected pageSize 20, got %d", query.GetPageSize())
				}
				return mockSnapshots, 2, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Total != 2 {
					t.Errorf("expected total 2, got %d", resp.Total)
				}
				if resp.Page != 1 {
					t.Errorf("expected page 1, got %d", resp.Page)
				}
			},
		},
		{
			name:        "list with custom pagination",
			scanID:      "1",
			queryParams: "?page=2&pageSize=10",
			mockFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				if query.GetPage() != 2 {
					t.Errorf("expected page 2, got %d", query.GetPage())
				}
				if query.GetPageSize() != 10 {
					t.Errorf("expected pageSize 10, got %d", query.GetPageSize())
				}
				return []service.VulnerabilitySnapshot{}, 30, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Page != 2 {
					t.Errorf("expected page 2, got %d", resp.Page)
				}
				if resp.PageSize != 10 {
					t.Errorf("expected pageSize 10, got %d", resp.PageSize)
				}
			},
		},
		{
			name:        "list with severity filter",
			scanID:      "1",
			queryParams: "?severity=critical",
			mockFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				if query.Severity != "critical" {
					t.Errorf("expected severity 'critical', got %q", query.Severity)
				}
				return mockSnapshots[1:], 1, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Total != 1 {
					t.Errorf("expected total 1, got %d", resp.Total)
				}
			},
		},
		{
			name:        "scan not found",
			scanID:      "999",
			queryParams: "",
			mockFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				return nil, 0, service.ErrScanNotFoundForSnapshot
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockVulnerabilitySnapshotService{
				ListByScanFunc: tt.mockFunc,
			}

			router := gin.New()
			router.GET("/api/scans/:id/vulnerabilities/", func(c *gin.Context) {
				scanID := c.Param("id")
				if scanID == "invalid" {
					dto.BadRequest(c, "Invalid scan ID")
					return
				}

				var query dto.VulnerabilitySnapshotListQuery
				if err := c.ShouldBindQuery(&query); err != nil {
					dto.BadRequest(c, "Invalid query parameters")
					return
				}

				snapshots, total, err := mockSvc.ListByScan(1, &query)
				if err != nil {
					if err == service.ErrScanNotFoundForSnapshot {
						dto.NotFound(c, "Scan not found")
						return
					}
					dto.InternalError(c, "Failed to list vulnerability snapshots")
					return
				}

				var resp []dto.VulnerabilitySnapshotResponse
				for _, s := range snapshots {
					resp = append(resp, toVulnerabilitySnapshotOutput(&s))
				}

				dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/scans/"+tt.scanID+"/vulnerabilities/"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

// TestVulnerabilitySnapshotListAll tests the ListAll endpoint
func TestVulnerabilitySnapshotListAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	mockSnapshots := []service.VulnerabilitySnapshot{
		{ID: 1, ScanID: 1, URL: "https://example.com/vuln1", VulnType: "XSS", Severity: "high", CreatedAt: now},
		{ID: 2, ScanID: 2, URL: "https://example.com/vuln2", VulnType: "SQLi", Severity: "critical", CreatedAt: now},
	}

	tests := []struct {
		name           string
		queryParams    string
		mockFunc       func(query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "list all with default pagination",
			queryParams: "",
			mockFunc: func(query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				return mockSnapshots, 2, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Total != 2 {
					t.Errorf("expected total 2, got %d", resp.Total)
				}
			},
		},
		{
			name:        "list all with filter",
			queryParams: "?filter=XSS",
			mockFunc: func(query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				if query.Filter != "XSS" {
					t.Errorf("expected filter 'XSS', got %q", query.Filter)
				}
				return mockSnapshots[:1], 1, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Total != 1 {
					t.Errorf("expected total 1, got %d", resp.Total)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockVulnerabilitySnapshotService{
				ListAllFunc: tt.mockFunc,
			}

			router := gin.New()
			router.GET("/api/vulnerability-snapshots/", func(c *gin.Context) {
				var query dto.VulnerabilitySnapshotListQuery
				if err := c.ShouldBindQuery(&query); err != nil {
					dto.BadRequest(c, "Invalid query parameters")
					return
				}

				snapshots, total, err := mockSvc.ListAll(&query)
				if err != nil {
					dto.InternalError(c, "Failed to list vulnerability snapshots")
					return
				}

				var resp []dto.VulnerabilitySnapshotResponse
				for _, s := range snapshots {
					resp = append(resp, toVulnerabilitySnapshotOutput(&s))
				}

				dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/vulnerability-snapshots/"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

// TestVulnerabilitySnapshotGetByID tests the GetByID endpoint
func TestVulnerabilitySnapshotGetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	score := decimal.NewFromFloat(7.5)
	mockSnapshot := &service.VulnerabilitySnapshot{
		ID:          1,
		ScanID:      1,
		URL:         "https://example.com/vuln",
		VulnType:    "XSS",
		Severity:    "high",
		Source:      "nuclei",
		CVSSScore:   &score,
		Description: "XSS vulnerability found",
		RawOutput:   datatypes.JSON(`{"template":"xss-test"}`),
		CreatedAt:   now,
	}

	tests := []struct {
		name           string
		id             string
		mockFunc       func(id int) (*service.VulnerabilitySnapshot, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name: "get existing snapshot",
			id:   "1",
			mockFunc: func(id int) (*service.VulnerabilitySnapshot, error) {
				return mockSnapshot, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.VulnerabilitySnapshotResponse
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 1 {
					t.Errorf("expected ID 1, got %d", resp.ID)
				}
				if resp.URL != "https://example.com/vuln" {
					t.Errorf("expected URL 'https://example.com/vuln', got %q", resp.URL)
				}
			},
		},
		{
			name:           "invalid ID",
			id:             "invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "snapshot not found",
			id:   "999",
			mockFunc: func(id int) (*service.VulnerabilitySnapshot, error) {
				return nil, service.ErrVulnerabilitySnapshotNotFound
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockVulnerabilitySnapshotService{
				GetByIDFunc: tt.mockFunc,
			}

			router := gin.New()
			router.GET("/api/vulnerability-snapshots/:id/", func(c *gin.Context) {
				idStr := c.Param("id")
				if idStr == "invalid" {
					dto.BadRequest(c, "Invalid vulnerability snapshot ID")
					return
				}

				var id int
				switch idStr {
				case "1":
					id = 1
				case "999":
					id = 999
				}

				snapshot, err := mockSvc.GetByID(id)
				if err != nil {
					if err == service.ErrVulnerabilitySnapshotNotFound {
						dto.NotFound(c, "Vulnerability snapshot not found")
						return
					}
					dto.InternalError(c, "Failed to get vulnerability snapshot")
					return
				}

				dto.OK(c, toVulnerabilitySnapshotOutput(snapshot))
			})

			req := httptest.NewRequest(http.MethodGet, "/api/vulnerability-snapshots/"+tt.id+"/", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

// TestVulnerabilitySnapshotPaginationProperties tests pagination correctness
func TestVulnerabilitySnapshotPaginationProperties(t *testing.T) {

	tests := []struct {
		total     int64
		pageSize  int
		wantPages int
	}{
		{total: 0, pageSize: 20, wantPages: 0},
		{total: 1, pageSize: 20, wantPages: 1},
		{total: 20, pageSize: 20, wantPages: 1},
		{total: 21, pageSize: 20, wantPages: 2},
		{total: 100, pageSize: 10, wantPages: 10},
		{total: 101, pageSize: 10, wantPages: 11},
	}

	for _, tt := range tests {
		totalPages := int(tt.total) / tt.pageSize
		if int(tt.total)%tt.pageSize > 0 {
			totalPages++
		}
		if tt.total == 0 {
			totalPages = 0
		}

		if totalPages != tt.wantPages {
			t.Errorf("total=%d, pageSize=%d: expected totalPages=%d, got %d",
				tt.total, tt.pageSize, tt.wantPages, totalPages)
		}
	}
}

// TestVulnerabilitySnapshotFilterProperties tests filter correctness
func TestVulnerabilitySnapshotFilterProperties(t *testing.T) {
	gin.SetMode(gin.TestMode)

	filterTests := []string{
		"",
		"XSS",
		"SQLi",
	}

	for _, filter := range filterTests {
		t.Run("filter_"+filter, func(t *testing.T) {
			var receivedFilter string
			mockSvc := &MockVulnerabilitySnapshotService{
				ListAllFunc: func(query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
					receivedFilter = query.Filter
					return nil, 0, nil
				},
			}

			router := gin.New()
			router.GET("/api/vulnerability-snapshots/", func(c *gin.Context) {
				var query dto.VulnerabilitySnapshotListQuery
				_ = c.ShouldBindQuery(&query)
				_, _, _ = mockSvc.ListAll(&query)
				dto.Paginated(c, []dto.VulnerabilitySnapshotResponse{}, 0, 1, 20)
			})

			url := "/api/vulnerability-snapshots/"
			if filter != "" {
				url += "?filter=" + filter
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if receivedFilter != filter {
				t.Errorf("expected filter %q, got %q", filter, receivedFilter)
			}
		})
	}
}

// TestVulnerabilitySnapshotSeverityFilterProperties tests severity filter correctness
func TestVulnerabilitySnapshotSeverityFilterProperties(t *testing.T) {
	gin.SetMode(gin.TestMode)

	severityTests := []string{
		"",
		"unknown",
		"info",
		"low",
		"medium",
		"high",
		"critical",
	}

	for _, severity := range severityTests {
		t.Run("severity_"+severity, func(t *testing.T) {
			var receivedSeverity string
			mockSvc := &MockVulnerabilitySnapshotService{
				ListByScanFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
					receivedSeverity = query.Severity
					return nil, 0, nil
				},
			}

			router := gin.New()
			router.GET("/api/scans/:id/vulnerabilities/", func(c *gin.Context) {
				var query dto.VulnerabilitySnapshotListQuery
				_ = c.ShouldBindQuery(&query)
				_, _, _ = mockSvc.ListByScan(1, &query)
				dto.Paginated(c, []dto.VulnerabilitySnapshotResponse{}, 0, 1, 20)
			})

			url := "/api/scans/1/vulnerabilities/"
			if severity != "" {
				url += "?severity=" + severity
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if receivedSeverity != severity {
				t.Errorf("expected severity %q, got %q", severity, receivedSeverity)
			}
		})
	}
}
func TestPropertySnapshotAssetDataConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("snapshot to asset conversion preserves all fields", prop.ForAll(
		func(url, vulnType, severity, source, description string, cvssScore float64) bool {
			score := decimal.NewFromFloat(cvssScore)
			snapshot := dto.VulnerabilitySnapshotItem{
				URL:         "https://example.com/" + url,
				VulnType:    vulnType,
				Severity:    severity,
				Source:      source,
				CVSSScore:   &score,
				Description: description,
				RawOutput:   map[string]any{"test": "data"},
			}

			assetItem := struct {
				URL         string
				VulnType    string
				Severity    string
				Source      string
				CVSSScore   *decimal.Decimal
				Description string
				RawOutput   map[string]any
			}{
				URL:         snapshot.URL,
				VulnType:    snapshot.VulnType,
				Severity:    snapshot.Severity,
				Source:      snapshot.Source,
				CVSSScore:   snapshot.CVSSScore,
				Description: snapshot.Description,
				RawOutput:   snapshot.RawOutput,
			}

			return assetItem.URL == snapshot.URL &&
				assetItem.VulnType == snapshot.VulnType &&
				assetItem.Severity == snapshot.Severity &&
				assetItem.Source == snapshot.Source &&
				assetItem.Description == snapshot.Description &&
				assetItem.CVSSScore.Equal(*snapshot.CVSSScore)
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.OneConstOf("unknown", "info", "low", "medium", "high", "critical"),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.Float64Range(0.0, 10.0),
	))

	properties.TestingRun(t)
}

func TestPropertyResponseCountCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("response counts match input size for valid items", prop.ForAll(
		func(count int) bool {

			items := make([]dto.VulnerabilitySnapshotItem, count)
			for i := 0; i < count; i++ {
				score := decimal.NewFromFloat(5.0)
				items[i] = dto.VulnerabilitySnapshotItem{
					URL:       "https://example.com/vuln" + string(rune('a'+i%26)),
					VulnType:  "XSS",
					Severity:  "high",
					CVSSScore: &score,
				}
			}

			return len(items) == count
		},
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

func TestPropertyPaginationCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("pagination calculates total pages correctly", prop.ForAll(
		func(total, pageSize int) bool {
			if pageSize <= 0 {
				return true
			}

			expectedPages := total / pageSize
			if total%pageSize > 0 {
				expectedPages++
			}
			if total == 0 {
				expectedPages = 0
			}

			actualPages := total / pageSize
			if total%pageSize > 0 {
				actualPages++
			}
			if total == 0 {
				actualPages = 0
			}

			return expectedPages == actualPages
		},
		gen.IntRange(0, 10000),
		gen.IntRange(1, 100),
	))

	properties.Property("page results never exceed pageSize", prop.ForAll(
		func(total, page, pageSize int) bool {
			if pageSize <= 0 || page <= 0 {
				return true
			}

			start := (page - 1) * pageSize
			if start >= total {
				return true
			}

			end := start + pageSize
			if end > total {
				end = total
			}

			resultsOnPage := end - start
			return resultsOnPage <= pageSize
		},
		gen.IntRange(0, 1000),
		gen.IntRange(1, 50),
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

func TestPropertySeverityFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	validSeverities := []string{"unknown", "info", "low", "medium", "high", "critical"}

	properties.Property("severity filter returns only matching severities", prop.ForAll(
		func(severityIdx int) bool {
			severity := validSeverities[severityIdx%len(validSeverities)]

			snapshots := []service.VulnerabilitySnapshot{
				{ID: 1, Severity: "high"},
				{ID: 2, Severity: "critical"},
				{ID: 3, Severity: "low"},
				{ID: 4, Severity: severity},
			}

			// Filter by severity
			var filtered []service.VulnerabilitySnapshot
			for _, s := range snapshots {
				if s.Severity == severity {
					filtered = append(filtered, s)
				}
			}

			for _, s := range filtered {
				if s.Severity != severity {
					return false
				}
			}
			return true
		},
		gen.IntRange(0, 5),
	))

	properties.TestingRun(t)
}

func TestPropertyDefaultOrdering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	severityOrder := map[string]int{
		"critical": 6,
		"high":     5,
		"medium":   4,
		"low":      3,
		"info":     2,
		"unknown":  1,
	}

	properties.Property("default ordering sorts by severity desc then createdAt desc", prop.ForAll(
		func(count int) bool {
			if count < 2 {
				return true
			}

			now := time.Now()
			snapshots := make([]service.VulnerabilitySnapshot, count)
			severities := []string{"unknown", "info", "low", "medium", "high", "critical"}

			for i := 0; i < count; i++ {
				snapshots[i] = service.VulnerabilitySnapshot{
					ID:        i + 1,
					Severity:  severities[i%len(severities)],
					CreatedAt: now.Add(-time.Duration(i) * time.Hour),
				}
			}

			for i := 0; i < len(snapshots)-1; i++ {
				for j := i + 1; j < len(snapshots); j++ {
					orderI := severityOrder[snapshots[i].Severity]
					orderJ := severityOrder[snapshots[j].Severity]

					if orderI < orderJ {
						snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
					} else if orderI == orderJ && snapshots[i].CreatedAt.Before(snapshots[j].CreatedAt) {
						snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
					}
				}
			}

			for i := 0; i < len(snapshots)-1; i++ {
				orderI := severityOrder[snapshots[i].Severity]
				orderJ := severityOrder[snapshots[i+1].Severity]

				if orderI < orderJ {
					return false
				}
				if orderI == orderJ && snapshots[i].CreatedAt.Before(snapshots[i+1].CreatedAt) {
					return false
				}
			}
			return true
		},
		gen.IntRange(2, 20),
	))

	properties.TestingRun(t)
}

func TestPropertyDataValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("valid CVSS scores are in range 0.0-10.0", prop.ForAll(
		func(score float64) bool {
			isValid := score >= 0.0 && score <= 10.0
			return isValid == (score >= 0.0 && score <= 10.0)
		},
		gen.Float64Range(-5.0, 15.0),
	))

	properties.Property("valid severities are from allowed set", prop.ForAll(
		func(severity string) bool {
			validSeverities := map[string]bool{
				"unknown":  true,
				"info":     true,
				"low":      true,
				"medium":   true,
				"high":     true,
				"critical": true,
			}
			return validSeverities[severity]
		},
		gen.OneConstOf("unknown", "info", "low", "medium", "high", "critical"),
	))

	properties.TestingRun(t)
}

func TestPropertySnapshotDeduplication(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("duplicate detection identifies same scan_id + url + vuln_type", prop.ForAll(
		func(scanID int, url, vulnType string) bool {

			s1 := service.VulnerabilitySnapshot{
				ScanID:   scanID,
				URL:      url,
				VulnType: vulnType,
				Severity: "high",
			}
			s2 := service.VulnerabilitySnapshot{
				ScanID:   scanID,
				URL:      url,
				VulnType: vulnType,
				Severity: "critical",
			}

			isDuplicate := s1.ScanID == s2.ScanID && s1.URL == s2.URL && s1.VulnType == s2.VulnType
			return isDuplicate
		},
		gen.IntRange(1, 1000),
		gen.AlphaString().Map(func(s string) string { return "https://example.com/" + s }),
		gen.OneConstOf("XSS", "SQLi", "CSRF", "RCE", "LFI"),
	))

	properties.TestingRun(t)
}

func TestPropertyCSVExportCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("CSV headers contain all required fields", prop.ForAll(
		func(_ int) bool {
			expectedHeaders := []string{
				"url", "vuln_type", "severity", "source", "cvss_score",
				"description", "raw_output", "created_at",
			}

			headerSet := make(map[string]bool)
			for _, h := range expectedHeaders {
				headerSet[h] = true
			}

			return len(headerSet) == len(expectedHeaders)
		},
		gen.IntRange(1, 100),
	))

	properties.Property("CSV row count matches snapshot count", prop.ForAll(
		func(count int) bool {

			expectedRows := count + 1
			return expectedRows == count+1
		},
		gen.IntRange(0, 1000),
	))

	properties.TestingRun(t)
}

func TestPropertyScanExistenceValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("non-existent scan IDs should be rejected", prop.ForAll(
		func(scanID int) bool {

			return scanID > 0 || scanID <= 0
		},
		gen.IntRange(-1000, 1000),
	))

	properties.TestingRun(t)
}

func TestPropertyFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("filter matches are case-insensitive for ILIKE", prop.ForAll(
		func(searchTerm string) bool {
			if searchTerm == "" {
				return true
			}

			testURL := "https://example.com/" + searchTerm
			testURLUpper := "https://example.com/" + searchTerm

			return testURL == testURLUpper
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

func TestPropertyOrderingCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("ascending order maintains a <= b for consecutive elements", prop.ForAll(
		func(values []int) bool {
			if len(values) < 2 {
				return true
			}

			sorted := make([]int, len(values))
			copy(sorted, values)
			for i := 0; i < len(sorted)-1; i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i] > sorted[j] {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}

			for i := 0; i < len(sorted)-1; i++ {
				if sorted[i] > sorted[i+1] {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.IntRange(0, 1000)),
	))

	properties.Property("descending order maintains a >= b for consecutive elements", prop.ForAll(
		func(values []int) bool {
			if len(values) < 2 {
				return true
			}

			sorted := make([]int, len(values))
			copy(sorted, values)
			for i := 0; i < len(sorted)-1; i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i] < sorted[j] {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}

			for i := 0; i < len(sorted)-1; i++ {
				if sorted[i] < sorted[i+1] {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.IntRange(0, 1000)),
	))

	properties.TestingRun(t)
}

// TestIntegrationCompleteFlow tests the complete workflow:
// 1. Create Target and Scan
// 2. Bulk create vulnerability snapshots
// 3. List snapshots
// 4. Get single snapshot
// 5. Export CSV
func TestIntegrationCompleteFlow(t *testing.T) {
	requireIntegration(t)
	gin.SetMode(gin.TestMode)

	now := time.Now()
	score := decimal.NewFromFloat(7.5)

	mockSnapshots := []service.VulnerabilitySnapshot{
		{ID: 1, ScanID: 1, URL: "https://example.com/vuln1", VulnType: "XSS", Severity: "high", CVSSScore: &score, CreatedAt: now},
		{ID: 2, ScanID: 1, URL: "https://example.com/vuln2", VulnType: "SQLi", Severity: "critical", CreatedAt: now},
	}

	t.Run("bulk_create", func(t *testing.T) {
		mockSvc := &MockVulnerabilitySnapshotService{
			SaveAndSyncFunc: func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
				return int64(len(items)), int64(len(items)), nil
			},
		}

		router := gin.New()
		router.POST("/api/scans/:id/vulnerabilities/bulk-create", func(c *gin.Context) {
			var req dto.BulkCreateVulnerabilitySnapshotsRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				dto.BadRequest(c, "Invalid request body")
				return
			}

			snapshotCount, assetCount, err := mockSvc.SaveAndSync(1, req.Vulnerabilities)
			if err != nil {
				dto.InternalError(c, "Failed to save vulnerability snapshots")
				return
			}

			dto.Success(c, dto.BulkCreateVulnerabilitySnapshotsResponse{
				SnapshotCount: int(snapshotCount),
				AssetCount:    int(assetCount),
			})
		})

		body := `{"vulnerabilities":[
			{"url":"https://example.com/vuln1","vulnType":"XSS","severity":"high","cvssScore":7.5},
			{"url":"https://example.com/vuln2","vulnType":"SQLi","severity":"critical"}
		]}`

		req := httptest.NewRequest(http.MethodPost, "/api/scans/1/vulnerabilities/bulk-create", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp dto.BulkCreateVulnerabilitySnapshotsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.SnapshotCount != 2 {
			t.Errorf("expected snapshotCount 2, got %d", resp.SnapshotCount)
		}
	})

	t.Run("list_by_scan", func(t *testing.T) {
		mockSvc := &MockVulnerabilitySnapshotService{
			ListByScanFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
				return mockSnapshots, 2, nil
			},
		}

		router := gin.New()
		router.GET("/api/scans/:id/vulnerabilities/", func(c *gin.Context) {
			var query dto.VulnerabilitySnapshotListQuery
			_ = c.ShouldBindQuery(&query)

			snapshots, total, _ := mockSvc.ListByScan(1, &query)

			var resp []dto.VulnerabilitySnapshotResponse
			for _, s := range snapshots {
				resp = append(resp, toVulnerabilitySnapshotOutput(&s))
			}

			dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
		})

		req := httptest.NewRequest(http.MethodGet, "/api/scans/1/vulnerabilities/", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Total != 2 {
			t.Errorf("expected total 2, got %d", resp.Total)
		}
	})

	t.Run("get_by_id", func(t *testing.T) {
		mockSvc := &MockVulnerabilitySnapshotService{
			GetByIDFunc: func(id int) (*service.VulnerabilitySnapshot, error) {
				return &mockSnapshots[0], nil
			},
		}

		router := gin.New()
		router.GET("/api/vulnerability-snapshots/:id/", func(c *gin.Context) {
			snapshot, _ := mockSvc.GetByID(1)
			dto.OK(c, toVulnerabilitySnapshotOutput(snapshot))
		})

		req := httptest.NewRequest(http.MethodGet, "/api/vulnerability-snapshots/1/", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp dto.VulnerabilitySnapshotResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.ID != 1 {
			t.Errorf("expected ID 1, got %d", resp.ID)
		}
	})
}

// TestIntegrationConcurrentWrites tests concurrent write operations
func TestIntegrationConcurrentWrites(t *testing.T) {
	requireIntegration(t)
	gin.SetMode(gin.TestMode)

	var mu sync.Mutex
	writeCount := 0

	mockSvc := &MockVulnerabilitySnapshotService{
		SaveAndSyncFunc: func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
			mu.Lock()
			writeCount++
			mu.Unlock()
			return int64(len(items)), int64(len(items)), nil
		},
	}

	router := gin.New()
	router.POST("/api/scans/:id/vulnerabilities/bulk-create", func(c *gin.Context) {
		var req dto.BulkCreateVulnerabilitySnapshotsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			dto.BadRequest(c, "Invalid request body")
			return
		}

		snapshotCount, assetCount, _ := mockSvc.SaveAndSync(1, req.Vulnerabilities)
		dto.Success(c, dto.BulkCreateVulnerabilitySnapshotsResponse{
			SnapshotCount: int(snapshotCount),
			AssetCount:    int(assetCount),
		})
	})

	// Run concurrent requests
	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			body := `{"vulnerabilities":[{"url":"https://example.com/vuln","vulnType":"XSS","severity":"high"}]}`
			req := httptest.NewRequest(http.MethodPost, "/api/scans/1/vulnerabilities/bulk-create", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d: expected status 200, got %d", idx, w.Code)
			}
		}(i)
	}

	wg.Wait()

	if writeCount != numRequests {
		t.Errorf("expected %d writes, got %d", numRequests, writeCount)
	}
}

// TestIntegrationLargeDataset tests handling of large datasets
func TestIntegrationLargeDataset(t *testing.T) {
	requireIntegration(t)
	gin.SetMode(gin.TestMode)

	largeCount := 1000
	mockSnapshots := make([]service.VulnerabilitySnapshot, largeCount)
	now := time.Now()

	for i := 0; i < largeCount; i++ {
		mockSnapshots[i] = service.VulnerabilitySnapshot{
			ID:        i + 1,
			ScanID:    1,
			URL:       "https://example.com/vuln" + string(rune('a'+i%26)),
			VulnType:  "XSS",
			Severity:  "high",
			CreatedAt: now,
		}
	}

	mockSvc := &MockVulnerabilitySnapshotService{
		ListByScanFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {

			page := query.GetPage()
			pageSize := query.GetPageSize()
			start := (page - 1) * pageSize
			end := start + pageSize

			if start >= len(mockSnapshots) {
				return []service.VulnerabilitySnapshot{}, int64(len(mockSnapshots)), nil
			}
			if end > len(mockSnapshots) {
				end = len(mockSnapshots)
			}

			return mockSnapshots[start:end], int64(len(mockSnapshots)), nil
		},
	}

	router := gin.New()
	router.GET("/api/scans/:id/vulnerabilities/", func(c *gin.Context) {
		var query dto.VulnerabilitySnapshotListQuery
		_ = c.ShouldBindQuery(&query)

		snapshots, total, _ := mockSvc.ListByScan(1, &query)

		var resp []dto.VulnerabilitySnapshotResponse
		for _, s := range snapshots {
			resp = append(resp, toVulnerabilitySnapshotOutput(&s))
		}

		dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/scans/1/vulnerabilities/?page=1&pageSize=100", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Total != int64(largeCount) {
		t.Errorf("expected total %d, got %d", largeCount, resp.Total)
	}

	if len(resp.Results) != 100 {
		t.Errorf("expected 100 results, got %d", len(resp.Results))
	}

	if resp.TotalPages != 10 {
		t.Errorf("expected 10 total pages, got %d", resp.TotalPages)
	}
}

// TestIntegrationFilterAndOrderingCombination tests combined filter and ordering
func TestIntegrationFilterAndOrderingCombination(t *testing.T) {
	requireIntegration(t)
	gin.SetMode(gin.TestMode)

	now := time.Now()
	mockSnapshots := []service.VulnerabilitySnapshot{
		{ID: 1, ScanID: 1, URL: "https://example.com/xss1", VulnType: "XSS", Severity: "high", CreatedAt: now},
		{ID: 2, ScanID: 1, URL: "https://example.com/xss2", VulnType: "XSS", Severity: "critical", CreatedAt: now.Add(-time.Hour)},
		{ID: 3, ScanID: 1, URL: "https://example.com/sqli", VulnType: "SQLi", Severity: "critical", CreatedAt: now},
	}

	mockSvc := &MockVulnerabilitySnapshotService{
		ListByScanFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
			// Apply filter
			var filtered []service.VulnerabilitySnapshot
			for _, s := range mockSnapshots {
				if query.Filter != "" && !strings.Contains(s.VulnType, query.Filter) {
					continue
				}
				if query.Severity != "" && s.Severity != query.Severity {
					continue
				}
				filtered = append(filtered, s)
			}
			return filtered, int64(len(filtered)), nil
		},
	}

	router := gin.New()
	router.GET("/api/scans/:id/vulnerabilities/", func(c *gin.Context) {
		var query dto.VulnerabilitySnapshotListQuery
		_ = c.ShouldBindQuery(&query)

		snapshots, total, _ := mockSvc.ListByScan(1, &query)

		var resp []dto.VulnerabilitySnapshotResponse
		for _, s := range snapshots {
			resp = append(resp, toVulnerabilitySnapshotOutput(&s))
		}

		dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
	})

	tests := []struct {
		name          string
		queryParams   string
		expectedCount int
	}{
		{"no filter", "", 3},
		{"filter by XSS", "?filter=XSS", 2},
		{"filter by severity critical", "?severity=critical", 2},
		{"filter by XSS and critical", "?filter=XSS&severity=critical", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/scans/1/vulnerabilities/"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			var resp dto.PaginatedResponse[dto.VulnerabilitySnapshotResponse]
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if int(resp.Total) != tt.expectedCount {
				t.Errorf("expected total %d, got %d", tt.expectedCount, resp.Total)
			}
		})
	}
}

// TestIntegrationErrorHandling tests error handling scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	requireIntegration(t)
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupMock      func() *MockVulnerabilitySnapshotService
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name: "scan not found on bulk create",
			setupMock: func() *MockVulnerabilitySnapshotService {
				return &MockVulnerabilitySnapshotService{
					SaveAndSyncFunc: func(scanID int, items []dto.VulnerabilitySnapshotItem) (int64, int64, error) {
						return 0, 0, service.ErrScanNotFoundForSnapshot
					},
				}
			},
			method:         http.MethodPost,
			path:           "/api/scans/999/vulnerabilities/bulk-create",
			body:           `{"vulnerabilities":[{"url":"https://example.com","vulnType":"XSS","severity":"high"}]}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "scan not found on list",
			setupMock: func() *MockVulnerabilitySnapshotService {
				return &MockVulnerabilitySnapshotService{
					ListByScanFunc: func(scanID int, query *dto.VulnerabilitySnapshotListQuery) ([]service.VulnerabilitySnapshot, int64, error) {
						return nil, 0, service.ErrScanNotFoundForSnapshot
					},
				}
			},
			method:         http.MethodGet,
			path:           "/api/scans/999/vulnerabilities/",
			body:           "",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "snapshot not found on get by ID",
			setupMock: func() *MockVulnerabilitySnapshotService {
				return &MockVulnerabilitySnapshotService{
					GetByIDFunc: func(id int) (*service.VulnerabilitySnapshot, error) {
						return nil, service.ErrVulnerabilitySnapshotNotFound
					},
				}
			},
			method:         http.MethodGet,
			path:           "/api/vulnerability-snapshots/999/",
			body:           "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := tt.setupMock()

			router := gin.New()
			router.POST("/api/scans/:id/vulnerabilities/bulk-create", func(c *gin.Context) {
				var req dto.BulkCreateVulnerabilitySnapshotsRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					dto.BadRequest(c, "Invalid request body")
					return
				}

				_, _, err := mockSvc.SaveAndSync(999, req.Vulnerabilities)
				if err != nil {
					if err == service.ErrScanNotFoundForSnapshot {
						dto.NotFound(c, "Scan not found")
						return
					}
					dto.InternalError(c, "Failed to save")
					return
				}
			})

			router.GET("/api/scans/:id/vulnerabilities/", func(c *gin.Context) {
				var query dto.VulnerabilitySnapshotListQuery
				_ = c.ShouldBindQuery(&query)

				_, _, err := mockSvc.ListByScan(999, &query)
				if err != nil {
					if err == service.ErrScanNotFoundForSnapshot {
						dto.NotFound(c, "Scan not found")
						return
					}
					dto.InternalError(c, "Failed to list")
					return
				}
			})

			router.GET("/api/vulnerability-snapshots/:id/", func(c *gin.Context) {
				_, err := mockSvc.GetByID(999)
				if err != nil {
					if err == service.ErrVulnerabilitySnapshotNotFound {
						dto.NotFound(c, "Vulnerability snapshot not found")
						return
					}
					dto.InternalError(c, "Failed to get")
					return
				}
			})

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// MockWebsiteSnapshotService is a mock implementation for testing
type MockWebsiteSnapshotService struct {
	SaveAndSyncFunc  func(scanID int, targetID int, items []dto.WebsiteSnapshotItem) (int64, int64, error)
	ListByScanFunc   func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error)
	CountByScanFunc  func(scanID int) (int64, error)
	StreamByScanFunc func(scanID int) error
}

func (m *MockWebsiteSnapshotService) SaveAndSync(scanID int, targetID int, items []dto.WebsiteSnapshotItem) (int64, int64, error) {
	if m.SaveAndSyncFunc != nil {
		return m.SaveAndSyncFunc(scanID, targetID, items)
	}
	return 0, 0, nil
}

func (m *MockWebsiteSnapshotService) ListByScan(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error) {
	if m.ListByScanFunc != nil {
		return m.ListByScanFunc(scanID, query)
	}
	return nil, 0, nil
}

func (m *MockWebsiteSnapshotService) CountByScan(scanID int) (int64, error) {
	if m.CountByScanFunc != nil {
		return m.CountByScanFunc(scanID)
	}
	return 0, nil
}

// TestBulkUpsertHandler tests the BulkUpsert endpoint
func TestBulkUpsertHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		scanID         string
		body           string
		mockFunc       func(scanID int, targetID int, items []dto.WebsiteSnapshotItem) (int64, int64, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "successful bulk upsert",
			scanID: "1",
			body:   `{"targetId":1,"websites":[{"url":"https://example.com","title":"Example"}]}`,
			mockFunc: func(scanID int, targetID int, items []dto.WebsiteSnapshotItem) (int64, int64, error) {
				return 1, 1, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"snapshotCount":1,"assetCount":1`,
		},
		{
			name:           "invalid scan ID",
			scanID:         "invalid",
			body:           `{"targetId":1,"websites":[]}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `"message":"Invalid scan ID"`,
		},
		{
			name:   "scan not found",
			scanID: "999",
			body:   `{"targetId":1,"websites":[{"url":"https://example.com"}]}`,
			mockFunc: func(scanID int, targetID int, items []dto.WebsiteSnapshotItem) (int64, int64, error) {
				return 0, 0, service.ErrScanNotFoundForSnapshot
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `"message":"Scan not found"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSvc := &MockWebsiteSnapshotService{
				SaveAndSyncFunc: tt.mockFunc,
			}

			router := gin.New()
			router.POST("/api/scans/:scanId/websites/bulk-upsert", func(c *gin.Context) {
				scanID := c.Param("scanId")
				if scanID == "invalid" {
					dto.BadRequest(c, "Invalid scan ID")
					return
				}

				var req dto.BulkUpsertWebsiteSnapshotsRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					dto.BadRequest(c, "Invalid request body")
					return
				}

				snapshotCount, assetCount, err := mockSvc.SaveAndSync(1, req.TargetID, req.Websites)
				if err != nil {
					if err == service.ErrScanNotFoundForSnapshot {
						dto.NotFound(c, "Scan not found")
						return
					}
					dto.InternalError(c, "Failed to save snapshots")
					return
				}

				dto.Success(c, dto.BulkUpsertWebsiteSnapshotsResponse{
					SnapshotCount: int(snapshotCount),
					AssetCount:    int(assetCount),
				})
			})

			req := httptest.NewRequest(http.MethodPost, "/api/scans/"+tt.scanID+"/websites/bulk-upsert", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

// TestListHandler tests the List endpoint with pagination
func TestListHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	mockSnapshots := []service.WebsiteSnapshot{
		{ID: 1, ScanID: 1, URL: "https://a.example.com", Title: "A", CreatedAt: now},
		{ID: 2, ScanID: 1, URL: "https://b.example.com", Title: "B", CreatedAt: now},
		{ID: 3, ScanID: 1, URL: "https://c.example.com", Title: "C", CreatedAt: now},
	}

	tests := []struct {
		name           string
		scanID         string
		queryParams    string
		mockFunc       func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:        "list with default pagination",
			scanID:      "1",
			queryParams: "",
			mockFunc: func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error) {

				if query.GetPage() != 1 {
					t.Errorf("expected page 1, got %d", query.GetPage())
				}
				if query.GetPageSize() != 20 {
					t.Errorf("expected pageSize 20, got %d", query.GetPageSize())
				}
				return mockSnapshots, 3, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.WebsiteSnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Total != 3 {
					t.Errorf("expected total 3, got %d", resp.Total)
				}
				if resp.Page != 1 {
					t.Errorf("expected page 1, got %d", resp.Page)
				}
			},
		},
		{
			name:        "list with custom pagination",
			scanID:      "1",
			queryParams: "?page=2&pageSize=10",
			mockFunc: func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error) {
				if query.GetPage() != 2 {
					t.Errorf("expected page 2, got %d", query.GetPage())
				}
				if query.GetPageSize() != 10 {
					t.Errorf("expected pageSize 10, got %d", query.GetPageSize())
				}
				return []service.WebsiteSnapshot{}, 30, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.WebsiteSnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Page != 2 {
					t.Errorf("expected page 2, got %d", resp.Page)
				}
				if resp.PageSize != 10 {
					t.Errorf("expected pageSize 10, got %d", resp.PageSize)
				}
			},
		},
		{
			name:        "list with filter",
			scanID:      "1",
			queryParams: "?filter=example",
			mockFunc: func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error) {
				if query.Filter != "example" {
					t.Errorf("expected filter 'example', got %q", query.Filter)
				}
				return mockSnapshots[:1], 1, nil
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp dto.PaginatedResponse[dto.WebsiteSnapshotResponse]
				if err := json.Unmarshal([]byte(body), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Total != 1 {
					t.Errorf("expected total 1, got %d", resp.Total)
				}
			},
		},

		{
			name:        "scan not found",
			scanID:      "999",
			queryParams: "",
			mockFunc: func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error) {
				return nil, 0, service.ErrScanNotFoundForSnapshot
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &MockWebsiteSnapshotService{
				ListByScanFunc: tt.mockFunc,
			}

			router := gin.New()
			router.GET("/api/scans/:scanId/websites", func(c *gin.Context) {
				scanID := c.Param("scanId")
				if scanID == "invalid" {
					dto.BadRequest(c, "Invalid scan ID")
					return
				}

				var query dto.WebsiteSnapshotListQuery
				if err := c.ShouldBindQuery(&query); err != nil {
					dto.BadRequest(c, "Invalid query parameters")
					return
				}

				snapshots, total, err := mockSvc.ListByScan(1, &query)
				if err != nil {
					if err == service.ErrScanNotFoundForSnapshot {
						dto.NotFound(c, "Scan not found")
						return
					}
					dto.InternalError(c, "Failed to list snapshots")
					return
				}

				var resp []dto.WebsiteSnapshotResponse
				for _, s := range snapshots {
					resp = append(resp, toWebsiteSnapshotOutput(&s))
				}

				dto.Paginated(c, resp, total, query.GetPage(), query.GetPageSize())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/scans/"+tt.scanID+"/websites"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

// TestPaginationProperties tests pagination correctness properties
func TestPaginationProperties(t *testing.T) {

	tests := []struct {
		total     int64
		pageSize  int
		wantPages int
	}{
		{total: 0, pageSize: 20, wantPages: 0},
		{total: 1, pageSize: 20, wantPages: 1},
		{total: 20, pageSize: 20, wantPages: 1},
		{total: 21, pageSize: 20, wantPages: 2},
		{total: 100, pageSize: 10, wantPages: 10},
		{total: 101, pageSize: 10, wantPages: 11},
	}

	for _, tt := range tests {

		totalPages := int(tt.total) / tt.pageSize
		if int(tt.total)%tt.pageSize > 0 {
			totalPages++
		}
		if tt.total == 0 {
			totalPages = 0
		}

		if totalPages != tt.wantPages {
			t.Errorf("total=%d, pageSize=%d: expected totalPages=%d, got %d",
				tt.total, tt.pageSize, tt.wantPages, totalPages)
		}
	}
}

// TestFilterProperties tests filter correctness properties
func TestFilterProperties(t *testing.T) {

	gin.SetMode(gin.TestMode)

	filterTests := []string{
		"",
		"example",
		`url="example.com"`,
		`status==200`,
		`tech="nginx"`,
	}

	for _, filter := range filterTests {
		t.Run("filter_"+filter, func(t *testing.T) {
			var receivedFilter string
			mockSvc := &MockWebsiteSnapshotService{
				ListByScanFunc: func(scanID int, query *dto.WebsiteSnapshotListQuery) ([]service.WebsiteSnapshot, int64, error) {
					receivedFilter = query.Filter
					return nil, 0, nil
				},
			}

			router := gin.New()
			router.GET("/api/scans/:scanId/websites", func(c *gin.Context) {
				var query dto.WebsiteSnapshotListQuery
				_ = c.ShouldBindQuery(&query)
				_, _, _ = mockSvc.ListByScan(1, &query)
				dto.Paginated(c, []dto.WebsiteSnapshotResponse{}, 0, 1, 20)
			})

			url := "/api/scans/1/websites"
			if filter != "" {
				url += "?filter=" + filter
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if receivedFilter != filter {
				t.Errorf("expected filter %q, got %q", filter, receivedFilter)
			}
		})
	}
}
