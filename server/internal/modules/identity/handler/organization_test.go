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
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
	"gorm.io/gorm"
)

func performOrganizationRequest(t *testing.T, method, route, target string, handler gin.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, route, handler)

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

type organizationHandlerStoreStub struct {
	activeByID          map[int]*identitydomain.Organization
	existsByName        map[string]bool
	existsByNameErr     error
	updateErr           error
	softDeleteErr       error
	batchDeleteErr      error
	batchAddErr         error
	unlinkErr           error
	listErr             error
	findTargetsErr      error
	list                []identitydomain.OrganizationWithTargetCount
	total               int64
	targets             []identitydomain.OrganizationTargetRef
	targetTotal         int64
	batchDeleteCount    int64
	unlinkCount         int64
	lastBatchDeleteIDs  []int
	lastDeletedID       int
	lastListPage        int
	lastListPageSize    int
	lastListFilter      string
	lastListOrderBy     string
	lastTargetOrgID     int
	lastTargetPage      int
	lastTargetPageSize  int
	lastTargetType      string
	lastTargetFilter    string
	lastLinkOrgID       int
	lastLinkTargetIDs   []int
	lastUnlinkOrgID     int
	lastUnlinkTargetIDs []int
	updatedOrg          *identitydomain.Organization
}

func (stub *organizationHandlerStoreStub) GetActiveByID(id int) (*identitydomain.Organization, error) {
	org, ok := stub.activeByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyOrg := *org
	return &copyOrg, nil
}

func (stub *organizationHandlerStoreStub) GetActiveByIDContext(_ context.Context, id int) (*identitydomain.Organization, error) {
	return stub.GetActiveByID(id)
}

func (stub *organizationHandlerStoreStub) ExistsByName(name string, excludeID ...int) (bool, error) {
	_ = excludeID
	if stub.existsByNameErr != nil {
		return false, stub.existsByNameErr
	}
	return stub.existsByName[name], nil
}

func (stub *organizationHandlerStoreStub) ExistsByNameContext(_ context.Context, name string, excludeID ...int) (bool, error) {
	return stub.ExistsByName(name, excludeID...)
}

func (stub *organizationHandlerStoreStub) Create(org *identitydomain.Organization) error {
	_ = org
	return nil
}

func (stub *organizationHandlerStoreStub) CreateContext(_ context.Context, org *identitydomain.Organization) error {
	return stub.Create(org)
}

func (stub *organizationHandlerStoreStub) Update(org *identitydomain.Organization) error {
	if stub.updateErr != nil {
		return stub.updateErr
	}
	copyOrg := *org
	stub.updatedOrg = &copyOrg
	return nil
}

func (stub *organizationHandlerStoreStub) UpdateContext(_ context.Context, org *identitydomain.Organization) error {
	return stub.Update(org)
}

func (stub *organizationHandlerStoreStub) SoftDelete(id int) error {
	if stub.softDeleteErr != nil {
		return stub.softDeleteErr
	}
	stub.lastDeletedID = id
	return nil
}

func (stub *organizationHandlerStoreStub) SoftDeleteContext(_ context.Context, id int) error {
	return stub.SoftDelete(id)
}

func (stub *organizationHandlerStoreStub) BatchSoftDelete(ids []int) (int64, error) {
	stub.lastBatchDeleteIDs = append([]int(nil), ids...)
	if stub.batchDeleteErr != nil {
		return 0, stub.batchDeleteErr
	}
	return stub.batchDeleteCount, nil
}

func (stub *organizationHandlerStoreStub) BatchSoftDeleteContext(_ context.Context, ids []int) (int64, error) {
	return stub.BatchSoftDelete(ids)
}

func (stub *organizationHandlerStoreStub) BatchAddTargets(organizationID int, targetIDs []int) error {
	stub.lastLinkOrgID = organizationID
	stub.lastLinkTargetIDs = append([]int(nil), targetIDs...)
	return stub.batchAddErr
}

func (stub *organizationHandlerStoreStub) BatchAddTargetsContext(_ context.Context, organizationID int, targetIDs []int) error {
	return stub.BatchAddTargets(organizationID, targetIDs)
}

func (stub *organizationHandlerStoreStub) UnlinkTargets(organizationID int, targetIDs []int) (int64, error) {
	stub.lastUnlinkOrgID = organizationID
	stub.lastUnlinkTargetIDs = append([]int(nil), targetIDs...)
	if stub.unlinkErr != nil {
		return 0, stub.unlinkErr
	}
	return stub.unlinkCount, nil
}

func (stub *organizationHandlerStoreStub) UnlinkTargetsContext(_ context.Context, organizationID int, targetIDs []int) (int64, error) {
	return stub.UnlinkTargets(organizationID, targetIDs)
}

func (stub *organizationHandlerStoreStub) FindByIDWithCount(id int) (*identitydomain.OrganizationWithTargetCount, error) {
	_ = id
	return nil, gorm.ErrRecordNotFound
}

func (stub *organizationHandlerStoreStub) List(page, pageSize int, filter, orderBy string) ([]identitydomain.OrganizationWithTargetCount, int64, error) {
	stub.lastListPage = page
	stub.lastListPageSize = pageSize
	stub.lastListFilter = filter
	stub.lastListOrderBy = orderBy
	if stub.listErr != nil {
		return nil, 0, stub.listErr
	}
	result := make([]identitydomain.OrganizationWithTargetCount, len(stub.list))
	copy(result, stub.list)
	return result, stub.total, nil
}

func (stub *organizationHandlerStoreStub) ListTargetsByOrganizationID(organizationID int, page, pageSize int, targetType, filter string) ([]identitydomain.OrganizationTargetRef, int64, error) {
	stub.lastTargetOrgID = organizationID
	stub.lastTargetPage = page
	stub.lastTargetPageSize = pageSize
	stub.lastTargetType = targetType
	stub.lastTargetFilter = filter
	if stub.findTargetsErr != nil {
		return nil, 0, stub.findTargetsErr
	}
	result := make([]identitydomain.OrganizationTargetRef, len(stub.targets))
	copy(result, stub.targets)
	return result, stub.targetTotal, nil
}

func newOrganizationHandlerForTest(store *organizationHandlerStoreStub) *OrganizationHandler {
	return NewOrganizationHandler(service.NewOrganizationFacade(service.NewOrganizationQueryService(store), service.NewOrganizationCommandService(store)))
}

func TestNewOrganizationHandler(t *testing.T) {
	svc := &service.OrganizationFacade{}

	handler := NewOrganizationHandler(svc)

	if handler == nil {
		t.Fatal("expected handler instance")
	}
	if handler.svc != svc {
		t.Fatal("expected service to be stored on handler")
	}
}

func TestOrganizationOutputHelpers(t *testing.T) {
	t.Run("newOrganizationOutput converts timestamps to UTC and preserves fields", func(t *testing.T) {
		createdAt := time.Date(2026, 3, 19, 10, 30, 45, 0, time.FixedZone("UTC+8", 8*60*60))

		response := newOrganizationOutput(7, "Acme", "Platform team", createdAt, 5)

		if response.ID != 7 || response.Name != "organizations/7" || response.DisplayName != "Acme" || response.Description != "Platform team" || response.TargetCount != 5 {
			t.Fatalf("unexpected organization response: %+v", response)
		}
		if response.CreatedAt.Location() != time.UTC {
			t.Fatalf("expected UTC location, got %s", response.CreatedAt.Location())
		}
		if !response.CreatedAt.Equal(createdAt.UTC()) {
			t.Fatalf("expected UTC timestamp %s, got %s", createdAt.UTC(), response.CreatedAt)
		}
	})

	t.Run("toTargetOutput converts timestamps to UTC and preserves target fields", func(t *testing.T) {
		createdAt := time.Date(2026, 3, 19, 9, 15, 0, 0, time.FixedZone("UTC-5", -5*60*60))

		response := toTargetOutput(service.OrganizationTargetRef{
			ID:        9,
			Name:      "api.example.com",
			Type:      "subdomain",
			CreatedAt: createdAt,
		})

		if response.ID != 9 || response.Name != "targets/9" || response.DisplayName != "api.example.com" || response.Type != "subdomain" {
			t.Fatalf("unexpected target response: %+v", response)
		}
		if response.CreatedAt.Location() != time.UTC {
			t.Fatalf("expected UTC location, got %s", response.CreatedAt.Location())
		}
		if !response.CreatedAt.Equal(createdAt.UTC()) {
			t.Fatalf("expected UTC timestamp %s, got %s", createdAt.UTC(), response.CreatedAt)
		}
	})
}

func TestOrganizationHandlerValidationFastFail(t *testing.T) {
	handler := NewOrganizationHandler(nil)

	t.Run("invalid organization id maps to bad request before service access", func(t *testing.T) {
		recorder := performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations/:organization",
			"/v1/organizations/not-a-number",
			handler.GetOrganizationByID,
			"",
		)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid organization ID"`) {
			t.Fatalf("unexpected bad-request response: %s", recorder.Body.String())
		}
	})

	t.Run("missing required create field maps to validation error before service access", func(t *testing.T) {
		recorder := performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations",
			"/v1/organizations",
			handler.CreateOrganization,
			`{"description":"team"}`,
		)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"Name"`) {
			t.Fatalf("unexpected validation response: %s", recorder.Body.String())
		}
	})
}

func TestOrganizationHandlerCommandEntrypoints(t *testing.T) {
	createdAt := time.Date(2026, 3, 19, 8, 0, 0, 0, time.UTC)

	t.Run("update organization success and public error mappings remain stable", func(t *testing.T) {
		store := &organizationHandlerStoreStub{
			activeByID: map[int]*identitydomain.Organization{
				2: {ID: 2, Name: "Acme", Description: "team", CreatedAt: createdAt},
				3: {ID: 3, Name: "Ops", Description: "ops", CreatedAt: createdAt.Add(time.Hour)},
			},
			existsByName: map[string]bool{"Taken": true},
		}
		handler := newOrganizationHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/2",
			handler.UpdateOrganization,
			`{"name":"organizations/2","displayName":"  Beta Platform  ","description":"  refreshed  ","updateMask":"displayName,description"}`,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body dto.OrganizationResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal update organization response: %v", err)
		}
		if body.ID != 2 || body.Name != "organizations/2" || body.DisplayName != "Beta Platform" || body.Description != "refreshed" {
			t.Fatalf("unexpected update response: %+v", body)
		}
		if store.updatedOrg == nil || store.updatedOrg.Name != "Beta Platform" || store.updatedOrg.Description != "refreshed" {
			t.Fatalf("expected normalized organization persisted, got %+v", store.updatedOrg)
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/2",
			handler.UpdateOrganization,
			`{"name":"organizations/2","displayName":"Beta Platform","description":"refreshed"}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing updateMask, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/2",
			handler.UpdateOrganization,
			`{"name":"organizations/99","displayName":"Beta Platform","description":"refreshed","updateMask":"displayName,description"}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for mismatched resource name, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/3",
			handler.UpdateOrganization,
			`{"name":"organizations/3","displayName":"Taken","description":"ops","updateMask":"displayName,description"}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Organization name already exists"`) {
			t.Fatalf("unexpected duplicate-name response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/99",
			handler.UpdateOrganization,
			`{"name":"organizations/99","displayName":"Ghost","description":"missing","updateMask":"displayName,description"}`,
		)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"Organization not found"`) {
			t.Fatalf("unexpected not-found response: %s", recorder.Body.String())
		}

		store.updateErr = errors.New("write unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/2",
			handler.UpdateOrganization,
			`{"name":"organizations/2","displayName":"Acme","description":"team","updateMask":"displayName,description"}`,
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to update organization"`) {
			t.Fatalf("unexpected internal-error response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPatch,
			"/v1/organizations/:organization",
			"/v1/organizations/not-a-number",
			handler.UpdateOrganization,
			`{"name":"organizations/not-a-number","displayName":"Ghost","description":"missing","updateMask":"displayName,description"}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid organization ID"`) {
			t.Fatalf("unexpected invalid-id response: %s", recorder.Body.String())
		}
	})

	t.Run("delete organization success and public error mappings remain stable", func(t *testing.T) {
		store := &organizationHandlerStoreStub{
			activeByID: map[int]*identitydomain.Organization{
				2: {ID: 2, Name: "Acme", Description: "team", CreatedAt: createdAt},
			},
		}
		handler := newOrganizationHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodDelete,
			"/v1/organizations/:organization",
			"/v1/organizations/2",
			handler.DeleteOrganization,
			"",
		)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if recorder.Body.Len() != 0 {
			t.Fatalf("expected empty response body, got %q", recorder.Body.String())
		}
		if store.lastDeletedID != 2 {
			t.Fatalf("expected delete id 2, got %d", store.lastDeletedID)
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodDelete,
			"/v1/organizations/:organization",
			"/v1/organizations/99",
			handler.DeleteOrganization,
			"",
		)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"Organization not found"`) {
			t.Fatalf("unexpected delete not-found response: %s", recorder.Body.String())
		}

		store.softDeleteErr = errors.New("delete unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodDelete,
			"/v1/organizations/:organization",
			"/v1/organizations/2",
			handler.DeleteOrganization,
			"",
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to delete organization"`) {
			t.Fatalf("unexpected delete internal-error response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodDelete,
			"/v1/organizations/:organization",
			"/v1/organizations/not-a-number",
			handler.DeleteOrganization,
			"",
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid organization ID"`) {
			t.Fatalf("unexpected delete invalid-id response: %s", recorder.Body.String())
		}
	})

	t.Run("batch delete organizations success validation and generic error paths remain stable", func(t *testing.T) {
		store := &organizationHandlerStoreStub{batchDeleteCount: 2}
		handler := newOrganizationHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations:batchDelete",
			"/v1/organizations:batchDelete",
			handler.BatchDeleteOrganizations,
			`{"names":["organizations/2","organizations/3"]}`,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body dto.BatchDeleteResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal batch delete response: %v", err)
		}
		if body.DeletedCount != 2 {
			t.Fatalf("expected deleted count 2, got %d", body.DeletedCount)
		}
		if len(store.lastBatchDeleteIDs) != 2 || store.lastBatchDeleteIDs[0] != 2 || store.lastBatchDeleteIDs[1] != 3 {
			t.Fatalf("unexpected batch delete ids: %v", store.lastBatchDeleteIDs)
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations:batchDelete",
			"/v1/organizations:batchDelete",
			handler.BatchDeleteOrganizations,
			`{"ids":[]}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid input data"`) {
			t.Fatalf("unexpected batch-delete validation response: %s", recorder.Body.String())
		}

		store.batchDeleteErr = errors.New("batch delete unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations:batchDelete",
			"/v1/organizations:batchDelete",
			handler.BatchDeleteOrganizations,
			`{"names":["organizations/2"]}`,
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to delete organizations"`) {
			t.Fatalf("unexpected batch-delete internal-error response: %s", recorder.Body.String())
		}
	})
}

func TestOrganizationHandlerQueryEntrypoints(t *testing.T) {
	createdAt := time.Date(2026, 3, 19, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	t.Run("list organizations success error mapping and query validation remain stable", func(t *testing.T) {
		store := &organizationHandlerStoreStub{
			list: []identitydomain.OrganizationWithTargetCount{
				{
					Organization: identitydomain.Organization{
						ID:          7,
						Name:        "Acme",
						Description: "platform team",
						CreatedAt:   createdAt,
					},
					TargetCount: 3,
				},
			},
			total: 1,
		}
		handler := newOrganizationHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations",
			"/v1/organizations?pageSize=1&filter=displayName%3D%22ac%22&orderBy=displayName",
			handler.List,
			"",
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var body httpdto.PaginatedResponse[dto.OrganizationResponse]
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal list organizations response: %v", err)
		}
		if body.TotalSize != 1 || body.NextPageToken != "" || len(body.Results) != 1 {
			t.Fatalf("unexpected list response metadata: %+v", body)
		}
		if body.Results[0].ID != 7 || body.Results[0].Name != "organizations/7" || body.Results[0].DisplayName != "Acme" || body.Results[0].TargetCount != 3 {
			t.Fatalf("unexpected organization result: %+v", body.Results[0])
		}
		if store.lastListPage != 1 || store.lastListPageSize != 1 || store.lastListFilter != `displayName="ac"` || store.lastListOrderBy != "displayName asc" {
			t.Fatalf("unexpected list args: page=%d pageSize=%d filter=%q orderBy=%q", store.lastListPage, store.lastListPageSize, store.lastListFilter, store.lastListOrderBy)
		}

		for _, path := range []string{
			"/v1/organizations?page=2",
			"/v1/organizations?sort=name",
			"/v1/organizations?sortBy=name",
			"/v1/organizations?sortOrder=asc",
			"/v1/organizations?keyword=acme",
		} {
			recorder = performOrganizationRequest(
				t,
				http.MethodGet,
				"/v1/organizations",
				path,
				handler.List,
				"",
			)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 for legacy query alias %s, got %d body=%s", path, recorder.Code, recorder.Body.String())
			}
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations",
			"/v1/organizations?orderBy=targetCount%20desc",
			handler.List,
			"",
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for unsupported orderBy, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations",
			"/v1/organizations?filter=description%3D%22ops%22",
			handler.List,
			"",
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for unsupported filter, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		store.listErr = errors.New("list unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations",
			"/v1/organizations",
			handler.List,
			"",
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to list organizations"`) {
			t.Fatalf("unexpected list internal-error response: %s", recorder.Body.String())
		}
		store.listErr = nil

		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations",
			"/v1/organizations?pageSize=-1",
			handler.List,
			"",
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"PageSize"`) {
			t.Fatalf("unexpected query validation response: %s", recorder.Body.String())
		}
	})
}

func TestOrganizationHandlerTargetEntrypoints(t *testing.T) {
	createdAt := time.Date(2026, 3, 19, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	t.Run("list link and unlink targets success error mapping and validation remain stable", func(t *testing.T) {
		store := &organizationHandlerStoreStub{
			activeByID: map[int]*identitydomain.Organization{
				5: {ID: 5, Name: "Ops", CreatedAt: createdAt},
			},
			targets: []identitydomain.OrganizationTargetRef{
				{ID: 11, Name: "example.com", Type: "domain", CreatedAt: createdAt},
			},
			targetTotal: 1,
			unlinkCount: 2,
		}
		handler := newOrganizationHandlerForTest(store)

		recorder := performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations/:organization/targets",
			"/v1/organizations/5/targets?pageToken=cGFnZToy&pageSize=1&type=domain&filter=example",
			handler.ListOrganizationTargets,
			"",
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var listBody httpdto.PaginatedResponse[dto.TargetResponse]
		if err := json.Unmarshal(recorder.Body.Bytes(), &listBody); err != nil {
			t.Fatalf("unmarshal list targets response: %v", err)
		}
		if listBody.TotalSize != 1 || listBody.NextPageToken != "" || len(listBody.Results) != 1 {
			t.Fatalf("unexpected target list response metadata: %+v", listBody)
		}
		if listBody.Results[0].ID != 11 || listBody.Results[0].Name != "targets/11" || listBody.Results[0].DisplayName != "example.com" || listBody.Results[0].Type != "domain" {
			t.Fatalf("unexpected target result: %+v", listBody.Results[0])
		}
		if store.lastTargetOrgID != 5 || store.lastTargetPage != 2 || store.lastTargetPageSize != 1 || store.lastTargetType != "domain" || store.lastTargetFilter != "example" {
			t.Fatalf(
				"unexpected target list args: org=%d page=%d pageSize=%d type=%q filter=%q",
				store.lastTargetOrgID,
				store.lastTargetPage,
				store.lastTargetPageSize,
				store.lastTargetType,
				store.lastTargetFilter,
			)
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchLink",
			"/v1/organizations/5/targets:batchLink",
			handler.BatchLinkOrganizationTargets,
			`{"targets":["targets/11","targets/12"]}`,
		)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if store.lastLinkOrgID != 5 || len(store.lastLinkTargetIDs) != 2 || store.lastLinkTargetIDs[0] != 11 || store.lastLinkTargetIDs[1] != 12 {
			t.Fatalf("unexpected link args: org=%d targetIDs=%v", store.lastLinkOrgID, store.lastLinkTargetIDs)
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchUnlink",
			"/v1/organizations/5/targets:batchUnlink",
			handler.BatchUnlinkOrganizationTargets,
			`{"targets":["targets/11","targets/12"]}`,
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
		}

		var unlinkBody struct {
			UnlinkedCount int64 `json:"unlinkedCount"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &unlinkBody); err != nil {
			t.Fatalf("unmarshal unlink targets response: %v", err)
		}
		if unlinkBody.UnlinkedCount != 2 {
			t.Fatalf("expected unlinked count 2, got %d", unlinkBody.UnlinkedCount)
		}
		if store.lastUnlinkOrgID != 5 || len(store.lastUnlinkTargetIDs) != 2 || store.lastUnlinkTargetIDs[0] != 11 || store.lastUnlinkTargetIDs[1] != 12 {
			t.Fatalf("unexpected unlink args: org=%d targetIDs=%v", store.lastUnlinkOrgID, store.lastUnlinkTargetIDs)
		}

		store.findTargetsErr = errors.New("target list unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations/:organization/targets",
			"/v1/organizations/5/targets",
			handler.ListOrganizationTargets,
			"",
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to list targets"`) {
			t.Fatalf("unexpected list targets internal-error response: %s", recorder.Body.String())
		}
		store.findTargetsErr = nil

		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations/:organization/targets",
			"/v1/organizations/99/targets",
			handler.ListOrganizationTargets,
			"",
		)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"Organization not found"`) {
			t.Fatalf("unexpected list targets not-found response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations/:organization/targets",
			"/v1/organizations/not-a-number/targets",
			handler.ListOrganizationTargets,
			"",
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid organization ID"`) {
			t.Fatalf("unexpected list targets invalid-id response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodGet,
			"/v1/organizations/:organization/targets",
			"/v1/organizations/5/targets?type=hostname",
			handler.ListOrganizationTargets,
			"",
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"Type"`) {
			t.Fatalf("unexpected list targets validation response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchLink",
			"/v1/organizations/not-a-number/targets:batchLink",
			handler.BatchLinkOrganizationTargets,
			`{"targets":["targets/11"]}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid organization ID"`) {
			t.Fatalf("unexpected link invalid-id response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchLink",
			"/v1/organizations/5/targets:batchLink",
			handler.BatchLinkOrganizationTargets,
			`{}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"Targets"`) {
			t.Fatalf("unexpected link validation response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchLink",
			"/v1/organizations/99/targets:batchLink",
			handler.BatchLinkOrganizationTargets,
			`{"targets":["targets/11"]}`,
		)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"Organization not found"`) {
			t.Fatalf("unexpected link not-found response: %s", recorder.Body.String())
		}

		store.batchAddErr = service.ErrTargetNotFound
		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchLink",
			"/v1/organizations/5/targets:batchLink",
			handler.BatchLinkOrganizationTargets,
			`{"targets":["targets/99"]}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"One or more target IDs do not exist"`) {
			t.Fatalf("unexpected link target-not-found response: %s", recorder.Body.String())
		}

		store.batchAddErr = errors.New("link unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchLink",
			"/v1/organizations/5/targets:batchLink",
			handler.BatchLinkOrganizationTargets,
			`{"targets":["targets/11"]}`,
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to link targets"`) {
			t.Fatalf("unexpected link internal-error response: %s", recorder.Body.String())
		}
		store.batchAddErr = nil

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchUnlink",
			"/v1/organizations/not-a-number/targets:batchUnlink",
			handler.BatchUnlinkOrganizationTargets,
			`{"targets":["targets/11"]}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"BAD_REQUEST"`) || !strings.Contains(recorder.Body.String(), `"message":"Invalid organization ID"`) {
			t.Fatalf("unexpected unlink invalid-id response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchUnlink",
			"/v1/organizations/5/targets:batchUnlink",
			handler.BatchUnlinkOrganizationTargets,
			`{}`,
		)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"VALIDATION_ERROR"`) || !strings.Contains(recorder.Body.String(), `"field":"Targets"`) {
			t.Fatalf("unexpected unlink validation response: %s", recorder.Body.String())
		}

		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchUnlink",
			"/v1/organizations/99/targets:batchUnlink",
			handler.BatchUnlinkOrganizationTargets,
			`{"targets":["targets/11"]}`,
		)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"NOT_FOUND"`) || !strings.Contains(recorder.Body.String(), `"message":"Organization not found"`) {
			t.Fatalf("unexpected unlink not-found response: %s", recorder.Body.String())
		}

		store.unlinkErr = errors.New("unlink unavailable")
		recorder = performOrganizationRequest(
			t,
			http.MethodPost,
			"/v1/organizations/:organization/targets:batchUnlink",
			"/v1/organizations/5/targets:batchUnlink",
			handler.BatchUnlinkOrganizationTargets,
			`{"targets":["targets/11"]}`,
		)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
		}
		if !strings.Contains(recorder.Body.String(), `"reason":"INTERNAL_ERROR"`) || !strings.Contains(recorder.Body.String(), `"message":"Failed to unlink targets"`) {
			t.Fatalf("unexpected unlink internal-error response: %s", recorder.Body.String())
		}
	})
}
