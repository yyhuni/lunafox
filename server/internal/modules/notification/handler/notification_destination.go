package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	notificationapp "github.com/yyhuni/lunafox/server/internal/modules/notification/application"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/dto"
)

type destinationSettingsService interface {
	Get(context.Context, int, domain.Provider) (domain.Destination, error)
	List(context.Context, int) ([]domain.Destination, error)
	Update(context.Context, int, domain.Destination) (domain.Destination, error)
}

type destinationTestDeliveryService interface {
	Deliver(context.Context, int, domain.Provider, string) (notificationapp.TestDeliveryResult, error)
}

// NotificationDestinationHandler exposes the constrained installation settings
// surface. It deliberately omits delivery history and manual redrive APIs.
type NotificationDestinationHandler struct {
	service      destinationSettingsService
	testDelivery destinationTestDeliveryService
}

// NewNotificationDestinationHandler creates the authorized settings boundary.
func NewNotificationDestinationHandler(service destinationSettingsService) *NotificationDestinationHandler {
	if service == nil {
		panic("notification destination handler service is required")
	}
	return &NotificationDestinationHandler{service: service}
}

// SetTestDeliveryService wires the optional custom method after the settings
// boundary is assembled. It is explicit to keep ordinary settings tests narrow.
func (handler *NotificationDestinationHandler) SetTestDeliveryService(service destinationTestDeliveryService) {
	if service == nil {
		panic("notification test delivery handler service is required")
	}
	handler.testDelivery = service
}

// List handles GET /v1/settings/notificationDestinations.
func (handler *NotificationDestinationHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	destinations, err := handler.service.List(c.Request.Context(), userID)
	if err != nil {
		handler.writeDestinationError(c, err, "list notification destinations")
		return
	}
	results := make([]dto.DestinationResponse, 0, len(destinations))
	for _, destination := range destinations {
		results = append(results, toDestinationOutput(destination))
	}
	httpdto.Success(c, dto.DestinationListResponse{Results: results, SupportedKinds: notificationSupportedKinds()})
}

// Get handles GET /v1/settings/notificationDestinations/:destination.
func (handler *NotificationDestinationHandler) Get(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	provider, err := parseDestinationProvider(c.Param("destination"))
	if err != nil {
		httpdto.BadRequest(c, "Unsupported notification destination")
		return
	}
	destination, err := handler.service.Get(c.Request.Context(), userID, provider)
	if err != nil {
		handler.writeDestinationError(c, err, "get notification destination")
		return
	}
	httpdto.Success(c, toDestinationOutput(destination))
}

// Update handles PATCH /v1/settings/notificationDestinations/:destination.
func (handler *NotificationDestinationHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	provider, err := parseDestinationProvider(c.Param("destination"))
	if err != nil {
		httpdto.BadRequest(c, "Unsupported notification destination")
		return
	}
	var request dto.DestinationUpdateRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	if request.Enabled == nil {
		httpdto.BadRequest(c, "enabled is required")
		return
	}
	subscriptions := make([]domain.Kind, 0, len(request.Subscriptions))
	for _, rawKind := range request.Subscriptions {
		subscriptions = append(subscriptions, domain.Kind(strings.TrimSpace(rawKind)))
	}
	destination := domain.Destination{
		Provider:      provider,
		Credential:    destinationCredentialForProvider(provider, request.Credential),
		Enabled:       *request.Enabled,
		Subscriptions: subscriptions,
	}
	if err := domain.ValidateDestination(destination); err != nil {
		httpdto.BadRequest(c, err.Error())
		return
	}
	updated, err := handler.service.Update(c.Request.Context(), userID, destination)
	if err != nil {
		handler.writeDestinationError(c, err, "update notification destination")
		return
	}
	httpdto.Success(c, toDestinationOutput(updated))
}

// TestDelivery handles POST /v1/settings/notificationDestinations/:destination:testDelivery.
func (handler *NotificationDestinationHandler) TestDelivery(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	provider, err := parseDestinationTestProvider(c.Param("destinationTestMethod"))
	if err != nil {
		httpdto.BadRequest(c, "Unsupported notification destination")
		return
	}
	if handler.testDelivery == nil {
		httpdto.NotFound(c, "Notification destination command not found")
		return
	}
	var request dto.DestinationTestDeliveryRequest
	if !httpdto.BindJSON(c, &request) {
		return
	}
	result, err := handler.testDelivery.Deliver(c.Request.Context(), userID, provider, destinationCredentialForProvider(provider, request.Credential))
	if err != nil {
		handler.writeDestinationError(c, err, "test notification destination")
		return
	}
	httpdto.Success(c, dto.DestinationTestDeliveryResponse{Result: string(result)})
}

func destinationCredentialForProvider(provider domain.Provider, credential string) string {
	if provider == domain.ProviderFeishu {
		return credential
	}
	return strings.TrimSpace(credential)
}

func (handler *NotificationDestinationHandler) writeDestinationError(c *gin.Context, err error, operation string) {
	if errors.Is(err, notificationapp.ErrNotificationPermissionDenied) {
		httpdto.Forbidden(c, "Permission denied")
		return
	}
	// Destination validation is run before the service call, so remaining errors
	// are persistence/authorization failures and must not echo credential data.
	httpdto.InternalError(c, "Failed to "+operation)
}

func parseDestinationProvider(raw string) (domain.Provider, error) {
	provider := domain.Provider(strings.TrimSpace(raw))
	if err := domain.ValidateProvider(provider); err != nil {
		return "", fmt.Errorf("unsupported notification provider")
	}
	return provider, nil
}

func parseDestinationTestProvider(raw string) (domain.Provider, error) {
	provider, method, found := strings.Cut(strings.TrimSpace(raw), ":")
	if !found || method != "testDelivery" {
		return "", fmt.Errorf("unsupported notification destination command")
	}
	return parseDestinationProvider(provider)
}

func toDestinationOutput(destination domain.Destination) dto.DestinationResponse {
	subscriptions := make([]string, 0, len(destination.Subscriptions))
	for _, kind := range destination.Subscriptions {
		subscriptions = append(subscriptions, string(kind))
	}
	return dto.DestinationResponse{
		Provider:              string(destination.Provider),
		Credential:            destination.Credential,
		Enabled:               destination.Enabled,
		Subscriptions:         subscriptions,
		RequiresWebhookUpdate: domain.RequiresWebhookUpdate(destination),
	}
}

func notificationSupportedKinds() []string {
	kinds := domain.ExternallyDeliverableKinds()
	result := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		result = append(result, string(kind))
	}
	return result
}
