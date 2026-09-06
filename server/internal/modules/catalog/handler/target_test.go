package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"gorm.io/gorm"
)

type targetHandlerStoreStub struct {
	activeByID        map[int]*catalogdomain.Target
	existsByName      map[string]bool
	updatedTarget     *catalogdomain.Target
	listItems         []catalogdomain.Target
	listTotal         int64
	lastPage          int
	lastPageSize      int
	lastFilter        string
	lastOrderBy       string
	listCalls         int
	deleteErr         error
	batchDeleteErr    error
	tombstoneIDs      []int
	batchDeleteIDs    [][]int
	batchDeletedCount int64
}

func (stub *targetHandlerStoreStub) GetActiveByID(id int) (*catalogdomain.Target, error) {
	target, ok := stub.activeByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTarget := *target
	return &copyTarget, nil
}

func (stub *targetHandlerStoreStub) ExistsByName(name string, excludeID ...int) (bool, error) {
	_ = excludeID
	return stub.existsByName[name], nil
}

func (stub *targetHandlerStoreStub) Create(target *catalogdomain.Target) error { return nil }

func (stub *targetHandlerStoreStub) Update(target *catalogdomain.Target) error {
	copyTarget := *target
	stub.updatedTarget = &copyTarget
	return nil
}

func (stub *targetHandlerStoreStub) SoftDelete(id int) error                  { return nil }
func (stub *targetHandlerStoreStub) BatchSoftDelete(ids []int) (int64, error) { return 0, nil }
func (stub *targetHandlerStoreStub) TombstoneAndEnsureCleanup(_ context.Context, id int) (bool, error) {
	stub.tombstoneIDs = append(stub.tombstoneIDs, id)
	if stub.deleteErr != nil {
		return false, stub.deleteErr
	}
	return true, nil
}
func (stub *targetHandlerStoreStub) BatchTombstoneAndEnsureCleanup(_ context.Context, ids []int) (int64, error) {
	stub.batchDeleteIDs = append(stub.batchDeleteIDs, append([]int(nil), ids...))
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	return stub.batchDeletedCount, nil
}
func (stub *targetHandlerStoreStub) BatchCreateIgnoreConflicts(targets []catalogdomain.Target) (int, error) {
	return 0, nil
}
func (stub *targetHandlerStoreStub) FindByNames(names []string) ([]catalogdomain.Target, error) {
	return nil, nil
}
func (stub *targetHandlerStoreStub) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Target, int64, error) {
	stub.listCalls++
	stub.lastPage = page
	stub.lastPageSize = pageSize
	stub.lastFilter = filter
	stub.lastOrderBy = orderBy
	return append([]catalogdomain.Target(nil), stub.listItems...), stub.listTotal, nil
}
func (stub *targetHandlerStoreStub) GetAssetCountsSummary(targetID int) (*catalogdomain.TargetAssetCounts, error) {
	return nil, nil
}
func (stub *targetHandlerStoreStub) GetVulnerabilityCountsSummary(targetID int) (*catalogdomain.VulnerabilityCounts, error) {
	return nil, nil
}
func (stub *targetHandlerStoreStub) ExistsByID(id int) (bool, error) { return false, nil }
func (stub *targetHandlerStoreStub) BatchAddTargets(organizationID int, targetIDs []int) error {
	return nil
}

func newTargetHandlerForTest(store *targetHandlerStoreStub) *TargetHandler {
	return NewTargetHandler(catalogapp.NewTargetFacade(catalogapp.NewTargetQueryService(store), catalogapp.NewTargetCommandService(store, store)))
}

func performTargetRequest(t *testing.T, handler gin.HandlerFunc, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, "/v1/targets/:target", handler)

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func performTargetBatchDeleteRequest(t *testing.T, handler *TargetHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/v1/targets:batchMethod", handler.BatchDelete)

	req := httptest.NewRequest(http.MethodPost, "/v1/targets:batchDelete", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func newTargetListTestRouter(store *targetHandlerStoreStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := newTargetHandlerForTest(store)
	router.GET("/v1/targets", handler.List)
	return router
}

func performTargetListRequest(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestTargetListReturnsCanonicalPaginatedResources(t *testing.T) {
	createdAt := time.Date(2026, 7, 4, 8, 0, 0, 0, time.UTC)
	lastScannedAt := createdAt.Add(time.Hour)
	store := &targetHandlerStoreStub{
		listItems: []catalogdomain.Target{{
			ID:            7,
			Name:          "example.com",
			Type:          "domain",
			CreatedAt:     createdAt,
			LastScannedAt: &lastScannedAt,
			Organizations: []catalogdomain.TargetOrganizationRef{{ID: 3, Name: "Acme"}},
		}},
		listTotal: 2,
	}
	router := newTargetListTestRouter(store)

	resp := performTargetListRequest(t, router, `/v1/targets?pageSize=1&filter=displayName%3D%22example%22%20%26%26%20type%3D%3D%22domain%22&orderBy=lastScannedAt%20desc`)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if store.lastPage != 1 || store.lastPageSize != 1 || store.lastFilter != `displayName="example" && type=="domain"` || store.lastOrderBy != "lastScannedAt desc" {
		t.Fatalf("unexpected list query: page=%d size=%d filter=%q orderBy=%q", store.lastPage, store.lastPageSize, store.lastFilter, store.lastOrderBy)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if body["totalSize"] != float64(2) || body["nextPageToken"] == "" {
		t.Fatalf("expected canonical totalSize and nextPageToken, got %s", resp.Body.String())
	}
	first := body["results"].([]any)[0].(map[string]any)
	if first["name"] != "targets/7" || first["displayName"] != "example.com" {
		t.Fatalf("expected canonical target name/displayName, got %+v", first)
	}
}

func TestTargetListReturnsCanonicalFieldsForEmptyResult(t *testing.T) {
	store := &targetHandlerStoreStub{listTotal: 0}
	resp := performTargetListRequest(t, newTargetListTestRouter(store), `/v1/targets?pageSize=10`)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal empty list response: %v", err)
	}
	if _, ok := body["results"]; !ok {
		t.Fatalf("empty target list response must expose results, got %s", resp.Body.String())
	}
	if totalSize, ok := body["totalSize"]; !ok || totalSize != float64(0) {
		t.Fatalf("empty target list response must expose totalSize=0, got %s", resp.Body.String())
	}
	if nextPageToken, ok := body["nextPageToken"]; !ok || nextPageToken != "" {
		t.Fatalf("empty target list response must expose nextPageToken as empty string, got %s", resp.Body.String())
	}
}

func TestTargetListRejectsLegacyQueryAliases(t *testing.T) {
	for _, path := range []string{
		"/v1/targets?page=2",
		"/v1/targets?type=domain",
		"/v1/targets?sort=createdAt",
		"/v1/targets?sortBy=createdAt",
		"/v1/targets?sortOrder=desc",
		"/v1/targets?keyword=example",
	} {
		store := &targetHandlerStoreStub{}
		resp := performTargetListRequest(t, newTargetListTestRouter(store), path)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d body=%s", path, resp.Code, resp.Body.String())
		}
		if store.listCalls != 0 {
			t.Fatalf("legacy query %s must be rejected before store access", path)
		}
	}
}

func TestTargetListRejectsUnsupportedFilterAndOrderBy(t *testing.T) {
	for _, path := range []string{
		`/v1/targets?filter=type%3D%22domain%22`,
		`/v1/targets?filter=owner%3D%3D%22acme%22`,
		`/v1/targets?filter=type%3D%3D%22hostname%22`,
		`/v1/targets?orderBy=actions%20asc`,
	} {
		store := &targetHandlerStoreStub{}
		resp := performTargetListRequest(t, newTargetListTestRouter(store), path)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d body=%s", path, resp.Code, resp.Body.String())
		}
		if store.listCalls != 0 {
			t.Fatalf("unsupported query %s must be rejected before store access", path)
		}
	}
}

func TestTargetUpdateRequiresNameAndUpdateMask(t *testing.T) {
	createdAt := time.Date(2026, 3, 19, 8, 0, 0, 0, time.UTC)
	handler := newTargetHandlerForTest(&targetHandlerStoreStub{
		activeByID: map[int]*catalogdomain.Target{
			5: {ID: 5, Name: "old.example", Type: "domain", CreatedAt: createdAt},
		},
	})

	recorder := performTargetRequest(
		t,
		handler.Update,
		http.MethodPatch,
		"/v1/targets/5",
		`{"name":"targets/5","displayName":"  Beta.Example  ","updateMask":"displayName"}`,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var body dto.TargetResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal target response: %v", err)
	}
	if body.Name != "targets/5" || body.DisplayName != "beta.example" {
		t.Fatalf("unexpected target response: %+v", body)
	}
}

func TestTargetUpdateRejectsInvalidUpdateMask(t *testing.T) {
	handler := newTargetHandlerForTest(&targetHandlerStoreStub{
		activeByID: map[int]*catalogdomain.Target{
			5: {ID: 5, Name: "old.example", Type: "domain"},
		},
	})

	recorder := performTargetRequest(
		t,
		handler.Update,
		http.MethodPatch,
		"/v1/targets/5",
		`{"name":"targets/5","displayName":"Beta","updateMask":"name"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = performTargetRequest(
		t,
		handler.Update,
		http.MethodPatch,
		"/v1/targets/5",
		`{"name":"targets/9","displayName":"Beta","updateMask":"displayName"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for name mismatch, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTargetUpdatePropagatesNotFound(t *testing.T) {
	handler := newTargetHandlerForTest(&targetHandlerStoreStub{})

	recorder := performTargetRequest(
		t,
		handler.Update,
		http.MethodPatch,
		"/v1/targets/5",
		`{"name":"targets/5","displayName":"Beta","updateMask":"displayName"}`,
	)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTargetUpdateMapsStoreErrors(t *testing.T) {
	store := &targetHandlerStoreStub{
		activeByID: map[int]*catalogdomain.Target{
			5: {ID: 5, Name: "old.example", Type: "domain"},
		},
		existsByName: map[string]bool{"taken.example": true},
	}
	handler := newTargetHandlerForTest(store)

	recorder := performTargetRequest(
		t,
		handler.Update,
		http.MethodPatch,
		"/v1/targets/5",
		`{"name":"targets/5","displayName":"Taken.Example","updateMask":"displayName"}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"message":"Target name already exists"`) {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestTargetDeleteReturnsNoContentForActiveAndRepeatedTombstones(t *testing.T) {
	store := &targetHandlerStoreStub{}
	handler := newTargetHandlerForTest(store)

	for attempt := 0; attempt < 2; attempt++ {
		recorder := performTargetRequest(t, handler.Delete, http.MethodDelete, "/v1/targets/9", "")
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("delete attempt %d: expected 204, got %d body=%s", attempt+1, recorder.Code, recorder.Body.String())
		}
	}
	if len(store.tombstoneIDs) != 2 || store.tombstoneIDs[0] != 9 || store.tombstoneIDs[1] != 9 {
		t.Fatalf("delete calls = %v, want two calls for target 9", store.tombstoneIDs)
	}
}

func TestTargetDeleteReturnsNotFoundOnlyForNeverExistingTarget(t *testing.T) {
	store := &targetHandlerStoreStub{deleteErr: gorm.ErrRecordNotFound}
	handler := newTargetHandlerForTest(store)

	recorder := performTargetRequest(t, handler.Delete, http.MethodDelete, "/v1/targets/404", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTargetBatchDeleteMapsMissingTargetAndPreservesResponseShape(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := &targetHandlerStoreStub{batchDeletedCount: 2}
		handler := newTargetHandlerForTest(store)

		recorder := performTargetBatchDeleteRequest(t, handler, `{"names":["targets/9","targets/3","targets/9"]}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		var response dto.BatchDeleteResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("unmarshal batch delete response: %v", err)
		}
		if response.DeletedCount != 2 {
			t.Fatalf("deletedCount = %d, want 2", response.DeletedCount)
		}
		if len(store.batchDeleteIDs) != 1 || len(store.batchDeleteIDs[0]) != 3 {
			t.Fatalf("batch delete IDs = %v, want parsed request names", store.batchDeleteIDs)
		}
	})

	t.Run("never existing target", func(t *testing.T) {
		store := &targetHandlerStoreStub{batchDeleteErr: gorm.ErrRecordNotFound}
		handler := newTargetHandlerForTest(store)

		recorder := performTargetBatchDeleteRequest(t, handler, `{"names":["targets/404"]}`)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func TestTargetBatchDeleteRejectsMoreThanFiveThousandNamesBeforeStoreAccess(t *testing.T) {
	store := &targetHandlerStoreStub{}
	handler := newTargetHandlerForTest(store)
	names := make([]string, 5001)
	for index := range names {
		names[index] = "targets/1"
	}
	body, err := json.Marshal(dto.BatchDeleteRequest{Names: names})
	if err != nil {
		t.Fatalf("marshal batch delete request: %v", err)
	}

	recorder := performTargetBatchDeleteRequest(t, handler, string(body))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(store.batchDeleteIDs) != 0 {
		t.Fatalf("batch delete store was called for oversized request: %v", store.batchDeleteIDs)
	}
}
