package hostport

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

// HostPortHandler handles host-port endpoints.
type HostPortHandler struct {
	svc *service.HostPortFacade
}

// NewHostPortHandler creates a new host-port handler.
func NewHostPortHandler(svc *service.HostPortFacade) *HostPortHandler {
	return &HostPortHandler{svc: svc}
}
