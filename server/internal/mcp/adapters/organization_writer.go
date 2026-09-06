package adapters

import (
	"context"

	"github.com/yyhuni/lunafox/server/internal/mcp/tools"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	identityapp "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identitydto "github.com/yyhuni/lunafox/server/internal/modules/identity/dto"
)

type organizationWriter struct {
	facade *identityapp.OrganizationFacade
}

// NewOrganizationWriter projects the shared identity command onto MCP without
// going through the REST transport.
func NewOrganizationWriter(facade *identityapp.OrganizationFacade) tools.OrganizationCreator {
	if facade == nil {
		return nil
	}
	return &organizationWriter{facade: facade}
}

func (writer *organizationWriter) Create(ctx context.Context, input tools.OrganizationCreateInput) (tools.OrganizationCreateOutput, error) {
	organization, err := writer.facade.CreateOrganizationContext(ctx, &identitydto.CreateOrganizationRequest{
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		return tools.OrganizationCreateOutput{}, mapCommandError(err)
	}
	return tools.OrganizationCreateOutput{
		ResourceName: httpdto.OrganizationName(organization.ID),
		DisplayName:  organization.Name,
	}, nil
}
