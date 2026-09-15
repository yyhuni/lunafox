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
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

type handlerContractService struct {
	createdInput     application.CreateUpgradeOperationInput
	createdUserID    int
	createCalls      int
	createErr        error
	checkResult      application.CheckForUpdatesResult
	checkErr         error
	checkUserID      int
	checkCalls       int
	getOperation     *domain.Operation
	getErr           error
	getUserID        int
	getOperationID   string
	getCalls         int
	retryOperation   *domain.Operation
	retryErr         error
	retryUserID      int
	retryOperationID string
	retryConfirmed   bool
	retryCalls       int
}

func (service *handlerContractService) CheckForUpdates(_ context.Context, userID int) (application.CheckForUpdatesResult, error) {
	service.checkUserID = userID
	service.checkCalls++
	if service.checkErr != nil {
		return application.CheckForUpdatesResult{}, service.checkErr
	}
	if service.checkResult.CurrentVersion == "" {
		return application.CheckForUpdatesResult{CurrentVersion: "1.0.0", HasUpdate: false}, nil
	}
	return service.checkResult, nil
}
func (service *handlerContractService) CreateOperation(_ context.Context, userID int, input application.CreateUpgradeOperationInput) (*domain.Operation, bool, error) {
	service.createdInput = input
	service.createdUserID = userID
	service.createCalls++
	if service.createErr != nil {
		return nil, false, service.createErr
	}
	return &domain.Operation{OperationID: "11111111-1111-4111-8111-111111111111", RequestID: input.RequestID, OperatorID: 7, ManifestID: input.ManifestID, ManifestDigest: input.ManifestDigest, ReleaseVersion: "1.1.0", CompatibilityRange: "*", Status: domain.StatusQueued, MigrationStatus: domain.MigrationStatusNotStarted, MigrationType: "none", StageTimes: map[domain.Status]time.Time{}, ObservedDigests: map[string]string{}}, true, nil
}

func TestUpgradeHandlerReturnsStructuredMigrationDiagnostic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerContractService{createErr: domain.NewMigrationUnsupported("migration requires an approved phase-transition change")}
	engine := gin.New()
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: 1}))
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"requestId":"22222222-2222-4222-8222-222222222222","manifestId":"release-1.1.0","manifestDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","confirmed":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/upgradeOperations", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Error struct {
			Code    int `json:"code"`
			Details []struct {
				Reason   string            `json:"reason"`
				Code     string            `json:"code"`
				Metadata map[string]string `json:"metadata"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != http.StatusBadRequest || len(response.Error.Details) < 2 {
		t.Fatalf("structured error = %s", rec.Body.String())
	}
	if response.Error.Details[0].Reason != "MIGRATION_UNSUPPORTED" || response.Error.Details[0].Metadata["stage"] != "migration" || response.Error.Details[1].Code != string(domain.ErrorCodeMigrationUnsupported) {
		t.Fatalf("diagnostic detail = %#v", response.Error.Details)
	}
}

func TestUpgradeHandlerReturnsCompatibilityFailureAsPrecondition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerContractService{createErr: domain.NewReleaseCompatibilityUnsupported()}
	engine := gin.New()
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: 1}))
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/upgradeOperations", strings.NewReader(`{"requestId":"22222222-2222-4222-8222-222222222222","manifestId":"release-1.1.0","manifestDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","confirmed":true}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	var response struct {
		Error struct {
			Status  string `json:"status"`
			Details []struct {
				Reason string `json:"reason"`
				Code   string `json:"code"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusBadRequest || response.Error.Status != "FAILED_PRECONDITION" || len(response.Error.Details) < 2 {
		t.Fatalf("compatibility precondition response = %s", recorder.Body.String())
	}
	if response.Error.Details[0].Reason != "RELEASE_COMPATIBILITY_UNSUPPORTED" || response.Error.Details[1].Code != string(domain.ErrorCodeReleaseCompatibilityUnsupported) {
		t.Fatalf("compatibility diagnostic details = %#v", response.Error.Details)
	}
}
func (service *handlerContractService) GetOperation(_ context.Context, userID int, operationID string) (*domain.Operation, error) {
	service.getUserID = userID
	service.getOperationID = operationID
	service.getCalls++
	if service.getErr != nil {
		return nil, service.getErr
	}
	if service.getOperation != nil {
		return service.getOperation, nil
	}
	return nil, domain.ErrUpgradeNotFound
}
func (service *handlerContractService) RetryOperation(_ context.Context, userID int, operationID string, confirmed bool) (*domain.Operation, error) {
	service.retryUserID = userID
	service.retryOperationID = operationID
	service.retryConfirmed = confirmed
	service.retryCalls++
	if service.retryErr != nil {
		return nil, service.retryErr
	}
	if service.retryOperation != nil {
		return service.retryOperation, nil
	}
	return nil, domain.ErrUpgradeRetryNotAllowed
}

func handlerTestOperation() *domain.Operation {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	return &domain.Operation{
		OperationID:              "11111111-1111-4111-8111-111111111111",
		RequestID:                "22222222-2222-4222-8222-222222222222",
		OperatorID:               7,
		ManifestID:               "release-1.1.0",
		ManifestDigest:           "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ReleaseVersion:           "1.1.0",
		CompatibilityRange:       ">=1.0.0 <2.0.0",
		MaintenanceWindowMinutes: 15,
		Status:                   domain.StatusRestarting,
		MigrationStatus:          domain.MigrationStatusNotStarted,
		MigrationType:            "none",
		AgentSummary:             domain.AgentSummary{},
		ObservedDigests:          map[string]string{},
		StageTimes:               map[domain.Status]time.Time{domain.StatusRestarting: now},
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

func TestUpgradeHandlerRejectsClientImageReferencesBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerContractService{}
	engine := gin.New()
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: 1}))
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"requestId":"22222222-2222-4222-8222-222222222222","manifestId":"release-1.1.0","manifestDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","confirmed":true,"imageRef":"docker.io/example:latest","operatorId":999,"username":"attacker","role":"superuser"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/upgradeOperations", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || service.createCalls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", rec.Code, service.createCalls, rec.Body.String())
	}
	var response struct {
		Error struct {
			Details []struct {
				Reason string `json:"reason"`
				Code   string `json:"code"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Error.Details) < 2 || response.Error.Details[0].Reason != "RELEASE_MANIFEST_TARGET_INVALID" || response.Error.Details[1].Code != string(domain.ErrorCodeReleaseManifestTargetInvalid) {
		t.Fatalf("error details=%s, want %s", rec.Body.String(), domain.ErrorCodeReleaseManifestTargetInvalid)
	}
}

func TestUpgradeHandlerUsesAuthenticatedUserAndIgnoresBodyIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerContractService{}
	engine := gin.New()
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: 1}))
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"requestId":"22222222-2222-4222-8222-222222222222","manifestId":"release-1.1.0","manifestDigest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","confirmed":true,"operatorId":999,"username":"attacker","role":"superuser"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/upgradeOperations", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || service.createCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", rec.Code, service.createCalls, rec.Body.String())
	}
	if service.createdUserID != 7 {
		t.Fatalf("service received body identity instead of JWT user: %d", service.createdUserID)
	}
	if service.createdInput.ImageRef != "" || len(service.createdInput.ImageRefs) != 0 {
		t.Fatalf("forbidden image fields reached service: %#v", service.createdInput)
	}
}

func TestUpgradeHandlerRejectsUnauthenticatedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(&handlerContractService{}))
	req := httptest.NewRequest(http.MethodPost, "/v1/upgradeOperations", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
}

func TestUpgradeHandlerCheckForUpdatesUsesAuthenticatedIdentityAndStrictEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerContractService{checkResult: application.CheckForUpdatesResult{
		CurrentVersion: "1.0.0",
		HasUpdate:      true,
		Eligible:       true,
		Manifest: application.ManifestSummary{
			ManifestID:                "release-1.1.0",
			ManifestDigest:            "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ReleaseVersion:            "1.1.0",
			DeploymentMode:            "single-node-compose",
			CompatibilityRange:        ">=1.0.0 <2.0.0",
			MaintenanceWindowMinutes:  15,
			RequiresAdminConfirmation: true,
			MigrationType:             "none",
			MigrationPolicyVersion:    1,
			RuntimeImageDigests:       map[string]string{"server": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			EngineDigests:             []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
	}}
	engine := gin.New()
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: 1}))
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/system:checkForUpdates", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || service.checkCalls != 1 || service.checkUserID != 7 {
		t.Fatalf("check response status=%d calls=%d user=%d body=%s", rec.Code, service.checkCalls, service.checkUserID, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["candidate"] == nil || response["currentVersion"] != "1.0.0" || response["hasUpdate"] != true || response["eligible"] != true {
		t.Fatalf("unexpected check response: %s", rec.Body.String())
	}
	if _, snakeCase := response["current_version"]; snakeCase {
		t.Fatal("check response exposed snake_case field")
	}

	strictReq := httptest.NewRequest(http.MethodPost, "/v1/system:checkForUpdates", strings.NewReader(`{"imageRef":"docker.io/example:latest"}`))
	strictReq.Header.Set("Authorization", "Bearer "+token)
	strictReq.Header.Set("Content-Type", "application/json")
	strictRec := httptest.NewRecorder()
	engine.ServeHTTP(strictRec, strictReq)
	if strictRec.Code != http.StatusBadRequest || service.checkCalls != 1 {
		t.Fatalf("unknown check field status=%d calls=%d body=%s", strictRec.Code, service.checkCalls, strictRec.Body.String())
	}
}

func TestUpgradeHandlerGetAndRetryUseCanonicalOperationIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &handlerContractService{getOperation: handlerTestOperation(), retryOperation: handlerTestOperation()}
	engine := gin.New()
	manager := auth.NewJWTManager("test-secret-key-32-chars-long!!", time.Minute, time.Hour)
	engine.Use(middleware.AuthMiddleware(manager, handlerTokenVersionReader{version: 1}))
	RegisterTestUpgradeHandler(engine, NewUpgradeHandler(service))
	token, _, err := manager.GenerateAccessToken(7, "admin", 1)
	if err != nil {
		t.Fatal(err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/upgradeOperations/11111111-1111-4111-8111-111111111111", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	engine.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK || service.getCalls != 1 || service.getUserID != 7 || service.getOperationID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("get response status=%d calls=%d user=%d id=%q body=%s", getRec.Code, service.getCalls, service.getUserID, service.getOperationID, getRec.Body.String())
	}
	var getBody map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatal(err)
	}
	if getBody["name"] != "upgradeOperations/11111111-1111-4111-8111-111111111111" || getBody["operationId"] != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("get identity response=%s", getRec.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodGet, "/v1/upgradeOperations/not-a-uuid", nil)
	badReq.Header.Set("Authorization", "Bearer "+token)
	badRec := httptest.NewRecorder()
	engine.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest || service.getCalls != 1 {
		t.Fatalf("invalid operation id status=%d calls=%d body=%s", badRec.Code, service.getCalls, badRec.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/v1/upgradeOperations/11111111-1111-4111-8111-111111111111:retry", strings.NewReader(`{"confirmed":true}`))
	retryReq.Header.Set("Authorization", "Bearer "+token)
	retryReq.Header.Set("Content-Type", "application/json")
	retryRec := httptest.NewRecorder()
	engine.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK || service.retryCalls != 1 || service.retryUserID != 7 || !service.retryConfirmed || service.retryOperationID != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("retry response status=%d calls=%d user=%d id=%q confirmed=%v body=%s", retryRec.Code, service.retryCalls, service.retryUserID, service.retryOperationID, service.retryConfirmed, retryRec.Body.String())
	}
}

func RegisterTestUpgradeHandler(engine *gin.Engine, upgradeHandler *UpgradeHandler) {
	group := engine.Group("/v1")
	group.POST("/system:checkForUpdates", upgradeHandler.CheckForUpdates)
	group.POST("/upgradeOperations", upgradeHandler.CreateOperation)
	group.GET("/upgradeOperations/:upgradeOperation", upgradeHandler.GetOperation)
	group.POST("/upgradeOperations/*upgradeOperationAction", upgradeHandler.RetryAction)
}

type handlerTokenVersionReader struct{ version int }

func (reader handlerTokenVersionReader) GetTokenVersion(context.Context, int) (int, error) {
	return reader.version, nil
}
