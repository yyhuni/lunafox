package screenshot

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

// ScreenshotHandler handles screenshot endpoints.
type ScreenshotHandler struct {
	svc *service.ScreenshotFacade
}

// NewScreenshotHandler creates a new screenshot handler.
func NewScreenshotHandler(svc *service.ScreenshotFacade) *ScreenshotHandler {
	return &ScreenshotHandler{svc: svc}
}
