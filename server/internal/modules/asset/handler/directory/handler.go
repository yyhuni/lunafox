package directory

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

// DirectoryHandler handles directory endpoints.
type DirectoryHandler struct {
	svc *service.DirectoryFacade
}

// NewDirectoryHandler creates a new directory handler.
func NewDirectoryHandler(svc *service.DirectoryFacade) *DirectoryHandler {
	return &DirectoryHandler{svc: svc}
}
