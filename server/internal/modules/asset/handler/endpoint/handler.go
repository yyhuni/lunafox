package endpoint

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

// EndpointHandler handles endpoint endpoints.
type EndpointHandler struct {
	svc *service.EndpointFacade
}

// NewEndpointHandler creates a new endpoint handler.
func NewEndpointHandler(svc *service.EndpointFacade) *EndpointHandler {
	return &EndpointHandler{svc: svc}
}
