package application

import "context"

type ScanCreateWorkflowReader interface {
	GetScanWorkflowManifest(ctx context.Context, scanWorkflowID string) (ScanCreateWorkflowManifest, error)
}

type ScanCreateEnginePackageReader interface {
	LoadEnginePackage(ctx context.Context, engineID string) (ScanCreateEnginePackage, error)
}

// ScanCreateEngineRegistrationReader is the planning-time existence boundary
// for disabled Steps. It must return registration facts without resolving a
// Definition, package archive, expanded cache entry, or runtime image.
type ScanCreateEngineRegistrationReader interface {
	EngineRegistrationExists(ctx context.Context, engineID string) (bool, error)
}

type ScanCreateEnginePackage struct {
	Package PlanTaskPackage
}

type ScanCreateWorkflowManifest struct {
	ScanWorkflowID string
	Stages         []ScanCreateWorkflowStage
}

type ScanCreateWorkflowStage struct {
	StageID string
	Steps   []ScanCreateWorkflowStep
}

type ScanCreateWorkflowStep struct {
	StepID   string
	EngineID string
}
