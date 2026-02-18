package website

import service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"

// WebsiteHandler handles website endpoints.
type WebsiteHandler struct {
	svc *service.WebsiteFacade
}

// NewWebsiteHandler creates a new website handler.
func NewWebsiteHandler(svc *service.WebsiteFacade) *WebsiteHandler {
	return &WebsiteHandler{svc: svc}
}
