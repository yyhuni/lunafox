package handler

import (
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
	"time"

	service "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	"github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

// OrganizationHandler handles organization endpoints.
type OrganizationHandler struct {
	svc *service.OrganizationFacade
}

// NewOrganizationHandler creates a new organization handler.
func NewOrganizationHandler(svc *service.OrganizationFacade) *OrganizationHandler {
	return &OrganizationHandler{svc: svc}
}

func newOrganizationOutput(id int, name, description string, createdAt time.Time, targetCount int64) dto.OrganizationResponse {
	return dto.OrganizationResponse{
		ID:          id,
		Name:        name,
		Description: description,
		CreatedAt:   timeutil.ToUTC(createdAt),
		TargetCount: targetCount,
	}
}

func toTargetOutput(target service.OrganizationTargetRef) dto.TargetResponse {
	return dto.TargetResponse{
		ID:        target.ID,
		Name:      target.Name,
		Type:      target.Type,
		CreatedAt: timeutil.ToUTC(target.CreatedAt),
	}
}
