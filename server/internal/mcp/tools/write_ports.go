package tools

import "context"

// OrganizationCreateInput is the bounded MCP organization command input.
type OrganizationCreateInput struct {
	Name        string
	Description string
}

// OrganizationCreateOutput is the safe chainable organization projection.
type OrganizationCreateOutput struct {
	ResourceName string `json:"name"`
	DisplayName  string `json:"displayName"`
}

// OrganizationCreator is the narrow mutation port required by the registry.
type OrganizationCreator interface {
	Create(context.Context, OrganizationCreateInput) (OrganizationCreateOutput, error)
}

// TargetBatchCreateInput is the shared canonical batch command input.
type TargetBatchCreateInput struct {
	Names          []string
	OrganizationID *int
}

// TargetBatchCreateOutput is a bounded aggregate; successful targets are not
// expanded into one result item per input.
type TargetBatchCreateOutput struct {
	CreatedCount         int            `json:"createdCount"`
	FailedCount          int            `json:"failedCount"`
	FailedTargets        []FailedTarget `json:"failedTargets"`
	Organization         string         `json:"organization,omitempty"`
	AssociationCompleted bool           `json:"associationCompleted"`
}

type FailedTarget struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// TargetBatchCreator is the narrow mutation port required by create_target.
type TargetBatchCreator interface {
	Create(context.Context, TargetBatchCreateInput) (TargetBatchCreateOutput, error)
}
