package application

import "fmt"

type ScanCreateService struct {
	scanStore                ScanCreateCommandStore
	targetLookup             TargetLookupFunc
	quickTargetEnsurer       QuickTargetEnsurerFunc
	organizationTargets      OrganizationTargetLookupFunc
	workflowReader           ScanCreateWorkflowReader
	enginePackageReader      ScanCreateEnginePackageReader
	engineRegistrationReader ScanCreateEngineRegistrationReader
	planTaskCompiler         *PlanTaskCompiler
	planTaskLimits           PlanTaskLimits
	agentLookup              ScanCreateAgentLookup
}

// ConfigurePlanTask installs the scan-create compiler with one immutable
// Server-owned limits snapshot. The snapshot cannot be changed by a scan
// request or re-resolved after its plans have been persisted.
func (service *ScanCreateService) ConfigurePlanTask(compiler *PlanTaskCompiler, limitsProvider PlanTaskLimitsProvider) error {
	if service == nil {
		return fmt.Errorf("scan create service is required")
	}
	if service.planTaskCompiler != nil {
		return fmt.Errorf("PlanTask policy is already configured")
	}
	if compiler == nil {
		return fmt.Errorf("PlanTask compiler is required")
	}
	if limitsProvider == nil {
		return fmt.Errorf("PlanTask limits provider is required")
	}
	limits := limitsProvider.Limits()
	if err := validatePlanTaskLimits(limits); err != nil {
		return fmt.Errorf("PlanTask limits provider: %w", err)
	}
	service.planTaskCompiler = compiler
	service.planTaskLimits = limits
	return nil
}

// WithEngineRegistrationReader adds the lightweight catalog boundary used by
// disabled Steps. Keeping it separate from the package reader prevents a
// disabled Step from accidentally entering exact package loading.
func (service *ScanCreateService) WithEngineRegistrationReader(reader ScanCreateEngineRegistrationReader) *ScanCreateService {
	if service != nil {
		service.engineRegistrationReader = reader
	}
	return service
}

func NewScanCreateService(
	scanStore ScanCreateCommandStore,
	targetLookup TargetLookupFunc,
	quickTargetEnsurer QuickTargetEnsurerFunc,
	workflowReader ScanCreateWorkflowReader,
	enginePackageReader ScanCreateEnginePackageReader,
	organizationTargets ...OrganizationTargetLookupFunc,
) *ScanCreateService {
	var organizationTargetLookup OrganizationTargetLookupFunc
	if len(organizationTargets) > 0 {
		organizationTargetLookup = organizationTargets[0]
	}
	return &ScanCreateService{
		scanStore:           scanStore,
		targetLookup:        targetLookup,
		quickTargetEnsurer:  quickTargetEnsurer,
		organizationTargets: organizationTargetLookup,
		workflowReader:      workflowReader,
		enginePackageReader: enginePackageReader,
	}
}
