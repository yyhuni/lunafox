package router

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/auth"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/handler"
)

func TestRegisterNotificationRoutesDispatchesCanonicalCustomMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inbox := &routeInboxServiceStub{item: routeInboxItem()}
	destinations := &routeDestinationServiceStub{}
	engine, token := notificationRouteEngine(t, inbox, destinations)

	request := func(method, target string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		httpRequest := httptest.NewRequest(method, target, nil)
		httpRequest.Header.Set("Authorization", "Bearer "+token)
		engine.ServeHTTP(recorder, httpRequest)
		return recorder
	}

	list := request(http.MethodGet, "/v1/users/current/notifications?pageSize=17&pageToken=next")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", list.Code, list.Body.String())
	}
	if inbox.listUserID != 7 || inbox.listInput != (notificationapp.InboxListInput{PageSize: 17, PageToken: "next"}) {
		t.Fatalf("list call = user=%d input=%#v", inbox.listUserID, inbox.listInput)
	}

	markRead := request(http.MethodPost, "/v1/users/current/notifications/42:markRead")
	if markRead.Code != http.StatusOK {
		t.Fatalf("markRead status = %d, body=%s", markRead.Code, markRead.Body.String())
	}
	if inbox.markReadUserID != 7 || inbox.markReadFactID != 42 {
		t.Fatalf("markRead call = user=%d fact=%d", inbox.markReadUserID, inbox.markReadFactID)
	}

	markAll := request(http.MethodPost, "/v1/users/current/notifications:markAllRead")
	if markAll.Code != http.StatusNoContent {
		t.Fatalf("markAllRead status = %d, body=%s", markAll.Code, markAll.Body.String())
	}
	if inbox.markAllUserID != 7 {
		t.Fatalf("markAllRead user = %d", inbox.markAllUserID)
	}

	unread := request(http.MethodGet, "/v1/users/current/notifications:unreadCount")
	if unread.Code != http.StatusOK {
		t.Fatalf("unreadCount status = %d, body=%s", unread.Code, unread.Body.String())
	}
}

func TestRegisterNotificationRoutesRejectsLegacyPageAndUnauthorizedDestinationRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inbox := &routeInboxServiceStub{item: routeInboxItem()}
	destinations := &routeDestinationServiceStub{err: notificationapp.ErrNotificationPermissionDenied}
	engine, token := notificationRouteEngine(t, inbox, destinations)

	legacyRecorder := httptest.NewRecorder()
	legacyRequest := httptest.NewRequest(http.MethodGet, "/v1/users/current/notifications?page=1&pageSize=10", nil)
	legacyRequest.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(legacyRecorder, legacyRequest)
	if legacyRecorder.Code != http.StatusBadRequest {
		t.Fatalf("legacy page status = %d, body=%s", legacyRecorder.Code, legacyRecorder.Body.String())
	}
	if inbox.listCalls != 0 {
		t.Fatal("legacy query reached inbox service")
	}

	destinationRecorder := httptest.NewRecorder()
	destinationRequest := httptest.NewRequest(http.MethodGet, "/v1/settings/notificationDestinations/discord", nil)
	destinationRequest.Header.Set("Authorization", "Bearer "+token)
	engine.ServeHTTP(destinationRecorder, destinationRequest)
	if destinationRecorder.Code != http.StatusForbidden {
		t.Fatalf("destination authorization status = %d, body=%s", destinationRecorder.Code, destinationRecorder.Body.String())
	}
	if strings.Contains(destinationRecorder.Body.String(), "credential") {
		t.Fatalf("permission error exposes a credential shape: %s", destinationRecorder.Body.String())
	}
}

func TestRegisterNotificationRoutesUpdatesCanonicalNotificationLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inbox := &routeInboxServiceStub{item: routeInboxItem()}
	engine, token := notificationRouteEngine(t, inbox, &routeDestinationServiceStub{})

	request := httptest.NewRequest(
		http.MethodPatch,
		"/v1/users/current/notificationLocale",
		strings.NewReader(`{"locale":" zh "}`),
	)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"locale":"zh"`) {
		t.Fatalf("locale update response = status %d body=%s", recorder.Code, recorder.Body.String())
	}
	if inbox.updateLocaleCalls != 1 || inbox.updateLocaleUserID != 7 || inbox.updateLocale != domain.LocaleChinese {
		t.Fatalf("locale update call = calls=%d user=%d locale=%q", inbox.updateLocaleCalls, inbox.updateLocaleUserID, inbox.updateLocale)
	}

	inbox.updateLocaleErr = errors.New("unsupported locale")
	invalidRequest := httptest.NewRequest(
		http.MethodPatch,
		"/v1/users/current/notificationLocale",
		strings.NewReader(`{"locale":"fr"}`),
	)
	invalidRequest.Header.Set("Authorization", "Bearer "+token)
	invalidRequest.Header.Set("Content-Type", "application/json")
	invalidRecorder := httptest.NewRecorder()
	engine.ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid locale status = %d body=%s", invalidRecorder.Code, invalidRecorder.Body.String())
	}

	legacyRequest := httptest.NewRequest(
		http.MethodPost,
		"/v1/users/current/notificationLocale:bootstrap",
		strings.NewReader(`{"browserLocale":"en-US"}`),
	)
	legacyRequest.Header.Set("Authorization", "Bearer "+token)
	legacyRequest.Header.Set("Content-Type", "application/json")
	legacyRecorder := httptest.NewRecorder()
	engine.ServeHTTP(legacyRecorder, legacyRequest)
	if legacyRecorder.Code != http.StatusNotFound {
		t.Fatalf("legacy bootstrap status = %d body=%s", legacyRecorder.Code, legacyRecorder.Body.String())
	}
}

func TestRegisterNotificationRoutesDispatchesCredentialSafeTestDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inbox := &routeInboxServiceStub{item: routeInboxItem()}
	destinations := &routeDestinationServiceStub{}
	testDelivery := &routeTestDeliveryServiceStub{result: notificationapp.TestDeliveryProviderRejected}
	destinationHandler := handler.NewNotificationDestinationHandler(destinations)
	destinationHandler.SetTestDeliveryService(testDelivery)
	engine := gin.New()
	protected := engine.Group("/v1")
	manager := auth.NewJWTManager("test-notification-route-secret-32!", time.Minute, time.Hour)
	protected.Use(middleware.AuthMiddleware(manager, routeTokenVersionReader{}))
	RegisterNotificationRoutes(protected, handler.NewNotificationInboxHandler(inbox), destinationHandler, handler.NewNotificationSSEHandler(&routeRefreshBroker{}))
	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatal(err)
	}

	credential := "https://discord.com/api/webhooks/test-id/test-token"
	body, err := json.Marshal(map[string]string{"credential": credential})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/settings/notificationDestinations/discord:testDelivery", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"result":"provider_rejected"`) {
		t.Fatalf("test delivery response = status %d body %s", recorder.Code, recorder.Body.String())
	}
	if testDelivery.calls != 1 || testDelivery.provider != domain.ProviderDiscord || testDelivery.credential != credential {
		t.Fatalf("test delivery call = %#v", testDelivery)
	}
	if strings.Contains(recorder.Body.String(), credential) {
		t.Fatalf("test delivery response leaked credential: %s", recorder.Body.String())
	}
}

func TestRegisterNotificationRoutesPreservesFeishuCredentialWhitespaceForStrictValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inbox := &routeInboxServiceStub{item: routeInboxItem()}
	destinations := &routeDestinationServiceStub{}
	testDelivery := &routeTestDeliveryServiceStub{result: notificationapp.TestDeliveryProviderRejected}
	destinationHandler := handler.NewNotificationDestinationHandler(destinations)
	destinationHandler.SetTestDeliveryService(testDelivery)
	engine := gin.New()
	protected := engine.Group("/v1")
	manager := auth.NewJWTManager("test-notification-route-secret-32!", time.Minute, time.Hour)
	protected.Use(middleware.AuthMiddleware(manager, routeTokenVersionReader{}))
	RegisterNotificationRoutes(protected, handler.NewNotificationInboxHandler(inbox), destinationHandler, handler.NewNotificationSSEHandler(&routeRefreshBroker{}))
	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatal(err)
	}

	credential := " https://open.feishu.cn/open-apis/bot/v2/hook/test-token "
	request := httptest.NewRequest(http.MethodPost, "/v1/settings/notificationDestinations/feishu:testDelivery", strings.NewReader(`{"credential":" `+`https://open.feishu.cn/open-apis/bot/v2/hook/test-token `+`"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Feishu test delivery status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if testDelivery.credential != credential {
		t.Fatalf("Feishu credential = %q, want original whitespace preserved", testDelivery.credential)
	}
}

func TestRegisterNotificationRoutesRedactsDeniedTestDelivery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("test-notification-route-secret-32!", time.Minute, time.Hour)
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(middleware.AuthMiddleware(manager, routeTokenVersionReader{}))
	destinations := &routeDestinationServiceStub{}
	testDelivery := &routeTestDeliveryServiceStub{err: notificationapp.ErrNotificationPermissionDenied}
	destinationHandler := handler.NewNotificationDestinationHandler(destinations)
	destinationHandler.SetTestDeliveryService(testDelivery)
	RegisterNotificationRoutes(protected, handler.NewNotificationInboxHandler(&routeInboxServiceStub{item: routeInboxItem()}), destinationHandler, handler.NewNotificationSSEHandler(&routeRefreshBroker{}))

	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/settings/notificationDestinations/discord:testDelivery", strings.NewReader(`{"credential":"https://discord.com/api/webhooks/test-id/test-token"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden || strings.Contains(recorder.Body.String(), "test-token") {
		t.Fatalf("unauthorized test response = status %d body %s", recorder.Code, recorder.Body.String())
	}
}

func TestRegisterNotificationRoutesReturnsAndClearsServerDerivedWebhookRemediationState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	legacyCredential := "https://discord.example.test/api/webhooks/legacy/token"
	destinations := &routeDestinationServiceStub{
		listDestinations: []domain.Destination{{Provider: domain.ProviderDiscord, Credential: legacyCredential}},
		updatedDestination: domain.Destination{
			Provider:      domain.ProviderDiscord,
			Credential:    "https://discord.com/api/webhooks/updated-id/updated-token",
			Enabled:       true,
			Subscriptions: []domain.Kind{domain.KindScanFailed},
		},
	}
	engine, token := notificationRouteEngine(t, &routeInboxServiceStub{item: routeInboxItem()}, destinations)

	listRequest := httptest.NewRequest(http.MethodGet, "/v1/settings/notificationDestinations", nil)
	listRequest.Header.Set("Authorization", "Bearer "+token)
	listRecorder := httptest.NewRecorder()
	engine.ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK || !strings.Contains(listRecorder.Body.String(), `"requiresWebhookUpdate":true`) {
		t.Fatalf("legacy remediation response = status %d body %s", listRecorder.Code, listRecorder.Body.String())
	}

	updateBody := strings.NewReader(`{"credential":"https://discord.com/api/webhooks/updated-id/updated-token","enabled":true,"subscriptions":["scan-failed"]}`)
	updateRequest := httptest.NewRequest(http.MethodPatch, "/v1/settings/notificationDestinations/discord", updateBody)
	updateRequest.Header.Set("Authorization", "Bearer "+token)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	engine.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusOK || !strings.Contains(updateRecorder.Body.String(), `"requiresWebhookUpdate":false`) {
		t.Fatalf("compliant remediation response = status %d body %s", updateRecorder.Code, updateRecorder.Body.String())
	}
}

func TestRegisterNotificationRoutesLeavesRemovedPreferenceResourceUnregistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	inbox := &routeInboxServiceStub{item: routeInboxItem()}
	engine, token := notificationRouteEngine(t, inbox, &routeDestinationServiceStub{})
	request := func(method, target string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		httpRequest := httptest.NewRequest(method, target, nil)
		httpRequest.Header.Set("Authorization", "Bearer "+token)
		engine.ServeHTTP(recorder, httpRequest)
		return recorder
	}

	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		response := request(method, "/v1/users/current/notificationPreferences")
		if response.Code != http.StatusNotFound {
			t.Fatalf("removed preference %s status = %d, body=%s", method, response.Code, response.Body.String())
		}
	}
}

func TestRegisterNotificationRoutesStreamsRefreshHintUntilRequestCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("test-notification-stream-secret-32!", time.Minute, time.Hour)
	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	broker := &controllableRefreshBroker{updates: make(chan struct{}), subscribed: make(chan struct{})}
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(middleware.AuthMiddleware(manager, routeTokenVersionReader{}))
	RegisterNotificationRoutes(
		protected,
		handler.NewNotificationInboxHandler(&routeInboxServiceStub{item: routeInboxItem()}),
		handler.NewNotificationDestinationHandler(&routeDestinationServiceStub{}),
		handler.NewNotificationSSEHandler(broker),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/current/notifications:stream", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer "+token)
	done := make(chan struct{})
	go func() {
		engine.ServeHTTP(recorder, request)
		close(done)
	}()

	select {
	case <-broker.subscribed:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not subscribe")
	}
	broker.updates <- struct{}{}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not stop after request cancellation")
	}
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("SSE response = status %d content-type %q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
	if !strings.Contains(recorder.Body.String(), "event: refresh\ndata: {}\n\n") {
		t.Fatalf("SSE response lacks refresh hint: %q", recorder.Body.String())
	}
}

func notificationRouteEngine(t *testing.T, inbox *routeInboxServiceStub, destinations *routeDestinationServiceStub) (*gin.Engine, string) {
	t.Helper()
	manager := auth.NewJWTManager("test-notification-route-secret-32!", time.Minute, time.Hour)
	token, _, err := manager.GenerateAccessToken(7, "operator", 1)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	engine := gin.New()
	protected := engine.Group("/v1")
	protected.Use(middleware.AuthMiddleware(manager, routeTokenVersionReader{}))
	RegisterNotificationRoutes(
		protected,
		handler.NewNotificationInboxHandler(inbox),
		handler.NewNotificationDestinationHandler(destinations),
		handler.NewNotificationSSEHandler(&routeRefreshBroker{}),
	)
	return engine, token
}

type routeTokenVersionReader struct{}

func (routeTokenVersionReader) GetTokenVersion(context.Context, int) (int, error) { return 1, nil }

type routeInboxServiceStub struct {
	item               domain.InboxItem
	listCalls          int
	listUserID         int
	listInput          notificationapp.InboxListInput
	markReadUserID     int
	markReadFactID     int64
	markAllUserID      int
	updateLocaleCalls  int
	updateLocaleUserID int
	updateLocale       domain.Locale
	updateLocaleErr    error
}

func (stub *routeInboxServiceStub) List(_ context.Context, userID int, input notificationapp.InboxListInput) (notificationapp.InboxPage, error) {
	stub.listCalls++
	stub.listUserID = userID
	stub.listInput = input
	return notificationapp.InboxPage{Results: []domain.InboxItem{stub.item}, TotalSize: 1}, nil
}

func (stub *routeInboxServiceStub) Get(context.Context, int, int64) (domain.InboxItem, error) {
	return stub.item, nil
}

func (*routeInboxServiceStub) UnreadCount(context.Context, int) (int64, error) { return 1, nil }

func (stub *routeInboxServiceStub) MarkRead(_ context.Context, userID int, factID int64) (domain.InboxItem, error) {
	stub.markReadUserID = userID
	stub.markReadFactID = factID
	return stub.item, nil
}

func (stub *routeInboxServiceStub) MarkAllRead(_ context.Context, userID int) error {
	stub.markAllUserID = userID
	return nil
}

func (*routeInboxServiceStub) Locale(context.Context, int) (domain.Locale, error) {
	return domain.LocaleEnglish, nil
}

func (stub *routeInboxServiceStub) UpdateLocale(_ context.Context, userID int, locale domain.Locale) error {
	stub.updateLocaleCalls++
	stub.updateLocaleUserID = userID
	stub.updateLocale = locale
	return stub.updateLocaleErr
}

type routeDestinationServiceStub struct {
	err                error
	getDestination     domain.Destination
	listDestinations   []domain.Destination
	updatedDestination domain.Destination
}

func (stub *routeDestinationServiceStub) Get(context.Context, int, domain.Provider) (domain.Destination, error) {
	return stub.getDestination, stub.err
}

func (stub *routeDestinationServiceStub) List(context.Context, int) ([]domain.Destination, error) {
	return stub.listDestinations, stub.err
}

func (stub *routeDestinationServiceStub) Update(_ context.Context, _ int, destination domain.Destination) (domain.Destination, error) {
	if stub.updatedDestination.Provider != "" {
		return stub.updatedDestination, stub.err
	}
	return destination, stub.err
}

type routeTestDeliveryServiceStub struct {
	calls      int
	provider   domain.Provider
	credential string
	result     notificationapp.TestDeliveryResult
	err        error
}

func (stub *routeTestDeliveryServiceStub) Deliver(_ context.Context, _ int, provider domain.Provider, credential string) (notificationapp.TestDeliveryResult, error) {
	stub.calls++
	stub.provider = provider
	stub.credential = credential
	return stub.result, stub.err
}

type routeRefreshBroker struct{}

func (*routeRefreshBroker) Subscribe(int) (<-chan struct{}, func()) {
	updates := make(chan struct{})
	return updates, func() {}
}

type controllableRefreshBroker struct {
	updates    chan struct{}
	subscribed chan struct{}
}

func (broker *controllableRefreshBroker) Subscribe(int) (<-chan struct{}, func()) {
	close(broker.subscribed)
	return broker.updates, func() {}
}

func routeInboxItem() domain.InboxItem {
	now := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	return domain.InboxItem{
		FactID:     42,
		Name:       "users/7/notifications/42",
		Kind:       domain.KindScanSucceeded,
		Category:   domain.CategoryScan,
		Priority:   domain.PriorityNormal,
		Subject:    "scans/8",
		Title:      "Scan completed",
		Message:    "Completed",
		OccurredAt: now,
		CreatedAt:  now,
	}
}
