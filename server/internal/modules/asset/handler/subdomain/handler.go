package subdomain

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

// SubdomainHandler handles subdomain endpoints.
type SubdomainHandler struct {
	svc *service.SubdomainFacade
}

// NewSubdomainHandler creates a new subdomain handler.
func NewSubdomainHandler(svc *service.SubdomainFacade) *SubdomainHandler {
	return &SubdomainHandler{svc: svc}
}
