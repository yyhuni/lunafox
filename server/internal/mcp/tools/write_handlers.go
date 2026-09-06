package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpErrors "github.com/yyhuni/lunafox/server/internal/mcp/errors"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

const (
	maxOrganizationNameLength        = 300
	maxOrganizationDescriptionLength = 1000
	maxTargetNameLength              = 300
	maxBatchTargetCount              = 5000
)

type organizationCreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type targetCreateInput struct {
	Targets      []targetCreateItem `json:"targets"`
	Organization string             `json:"organization,omitempty"`
}

type targetCreateItem struct {
	Name string `json:"name"`
}

func (registry *Registry) createOrganization(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input organizationCreateInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid create_organization arguments")
	}
	if strings.TrimSpace(input.Name) == "" || len(input.Name) > maxOrganizationNameLength || len(input.Description) > maxOrganizationDescriptionLength {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	if registry.deps.Organizations == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	output, err := registry.deps.Organizations.Create(ctx, OrganizationCreateInput{
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, output, "Created one organization.")
}

func (registry *Registry) createTarget(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var input targetCreateInput
	if err := decodeRequest(req, &input); err != nil {
		return nil, invalidParams("invalid create_target arguments")
	}
	if len(input.Targets) < 1 || len(input.Targets) > maxBatchTargetCount {
		return toolError(ctx, mcpErrors.ErrInvalidInput)
	}
	targets := make([]string, 0, len(input.Targets))
	for _, target := range input.Targets {
		if strings.TrimSpace(target.Name) == "" || len(target.Name) > maxTargetNameLength {
			return toolError(ctx, mcpErrors.ErrInvalidInput)
		}
		targets = append(targets, target.Name)
	}
	organizationID, err := parseOrganizationReference(input.Organization)
	if err != nil {
		return toolError(ctx, err)
	}
	if registry.deps.TargetCreator == nil {
		return toolError(ctx, mcpErrors.ErrInternal)
	}
	output, err := registry.deps.TargetCreator.Create(ctx, TargetBatchCreateInput{
		Names:          targets,
		OrganizationID: organizationID,
	})
	if err != nil {
		return toolError(ctx, err)
	}
	return successResult(ctx, output, fmt.Sprintf("Created %d targets; %d inputs were not recognized.", output.CreatedCount, output.FailedCount))
}

func parseOrganizationReference(value string) (*int, error) {
	if value == "" {
		return nil, nil
	}
	id, err := httpdto.ParseResourceNameID(value, "organizations")
	if err != nil || httpdto.OrganizationName(id) != value {
		return nil, fmt.Errorf("%w: organization must be a canonical organizations/{id} resource name", mcpErrors.ErrInvalidInput)
	}
	return &id, nil
}

func createOrganizationSchema() map[string]any {
	return objectSchema(map[string]any{
		"name": map[string]any{
			"type": "string", "minLength": 1, "maxLength": maxOrganizationNameLength,
		},
		"description": map[string]any{
			"type": "string", "maxLength": maxOrganizationDescriptionLength,
		},
	}, "name")
}

func createTargetSchema() map[string]any {
	return objectSchema(map[string]any{
		"targets": map[string]any{
			"type": "array", "minItems": 1, "maxItems": maxBatchTargetCount,
			"items": objectSchema(map[string]any{
				"name": map[string]any{"type": "string", "minLength": 1, "maxLength": maxTargetNameLength},
			}, "name"),
		},
		"organization": map[string]any{
			"type": "string", "pattern": "^organizations/[1-9][0-9]*$",
		},
	}, "targets")
}

func constrainedWriteAnnotations() *mcp.ToolAnnotations {
	destructive := false
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: &destructive,
		IdempotentHint:  false,
	}
}
