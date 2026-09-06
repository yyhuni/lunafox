package application

import "context"

type TargetLookupFunc func(ctx context.Context, id int) (*TargetRef, error)

type QuickTargetEnsurerFunc func(ctx context.Context, names []string) (*QuickTargetResolution, error)

type OrganizationTargetLookupFunc func(ctx context.Context, organizationID int) ([]TargetRef, error)

type ScanCreateTargetLookup interface {
	GetTargetRefByID(ctx context.Context, id int) (*TargetRef, error)
	EnsureQuickTargets(ctx context.Context, names []string) (*QuickTargetResolution, error)
	ListOrganizationTargetRefs(ctx context.Context, organizationID int) ([]TargetRef, error)
}
