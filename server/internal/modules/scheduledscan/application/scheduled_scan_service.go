package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

var (
	ErrScheduledScanInvalidArgument = errors.New("scheduled scan invalid argument")
	ErrScheduledScanNotFound        = errors.New("scheduled scan not found")
	ErrScheduledScanAgentNotFound   = errors.New("selected agent not found")
)

// MaxScheduledScanBatchStatusUpdates bounds one atomic status transition.
const MaxScheduledScanBatchStatusUpdates = 100

type ScheduledScan struct {
	ID                     int
	Name                   string
	ScanWorkflowID         string
	Configuration          map[string]any
	InputSource            scandomain.InputSource
	OrganizationID         *int
	OrganizationName       *string
	TargetID               *int
	TargetName             *string
	AgentID                *int
	CronExpression         string
	IsEnabled              bool
	NextRunTime            *time.Time
	LastRunTime            *time.Time
	RunCount               int
	SuccessfulHandoffCount int
	FailedHandoffCount     int
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ScheduledScanCreate struct {
	Name           string
	ScanWorkflowID string
	Configuration  map[string]any
	InputSource    scandomain.InputSource
	OrganizationID *int
	TargetID       *int
	AgentID        *int
	CronExpression string
	IsEnabled      bool
}

type ScheduledScanUpdate struct {
	Name           *string
	ScanWorkflowID *string
	Configuration  map[string]any
	InputSource    *scandomain.InputSource
	OrganizationID *int
	TargetID       *int
	AgentID        *int
	AgentSet       bool
	CronExpression *string
	IsEnabled      *bool
}

// ScheduledScanStatusUpdate assigns one Scheduled Scan to an explicit target state.
type ScheduledScanStatusUpdate struct {
	ID        int
	IsEnabled bool
}

type ScheduledScanListQuery struct {
	Page           int
	PageSize       int
	Search         string
	TargetID       int
	OrganizationID int
}

type CreateScheduledScanInput struct {
	Name           string
	ScanWorkflow   string
	Configuration  map[string]any
	InputSource    scandomain.InputSource
	Organization   string
	Target         string
	OrganizationID *int
	TargetID       *int
	Agent          string
	CronExpression string
	IsEnabled      *bool
}

type UpdateScheduledScanInput struct {
	Name           *string
	ScanWorkflow   *string
	Configuration  map[string]any
	InputSource    *scandomain.InputSource
	Organization   *string
	Target         *string
	OrganizationID *int
	TargetID       *int
	Agent          *string
	CronExpression *string
	IsEnabled      *bool
}

type ScheduledScanStore interface {
	Create(context.Context, *ScheduledScanCreate) (*ScheduledScan, error)
	Update(context.Context, int, *ScheduledScanUpdate) (*ScheduledScan, error)
	BatchUpdateStatus(context.Context, []ScheduledScanStatusUpdate) (int, error)
	List(context.Context, ScheduledScanListQuery) ([]ScheduledScan, int64, error)
	GetOverviewSummary(context.Context, ScheduledScanOverviewQuery) (*ScheduledScanOverviewProjection, error)
	GetByID(context.Context, int) (*ScheduledScan, error)
	Delete(context.Context, int) error
}

// ScheduledScanWorkflowStore deliberately reads only the current workflow
// aggregate. A scheduled scan persists its resource ID, never a revision or
// topology snapshot.
type ScheduledScanWorkflowStore interface {
	GetScanWorkflowByID(scanWorkflowID string) (*catalogdomain.ManagedScanWorkflow, error)
}

type ScheduledScanConfigResourceValidator interface {
	Validate(context.Context, scanapp.WorkflowConfigResourceValidationRequest) error
}

type ScheduledScanService struct {
	store      ScheduledScanStore
	workflows  ScheduledScanWorkflowStore
	agents     scanapp.ScanCreateAgentLookup
	resources  ScheduledScanConfigResourceValidator
	calculator ScheduleCalculator
	now        func() time.Time
}

func NewScheduledScanService(store ScheduledScanStore, workflows ScheduledScanWorkflowStore) *ScheduledScanService {
	service := &ScheduledScanService{store: store, workflows: workflows, calculator: NewCronScheduleCalculator(), now: time.Now}
	return service
}

func (service *ScheduledScanService) WithClock(now func() time.Time) *ScheduledScanService {
	if service != nil && now != nil {
		service.now = now
	}
	return service
}

func (service *ScheduledScanService) WithScheduleCalculator(calculator ScheduleCalculator) *ScheduledScanService {
	if service != nil && calculator != nil {
		service.calculator = calculator
	}
	return service
}

func (service *ScheduledScanService) WithAgentLookup(lookup scanapp.ScanCreateAgentLookup) *ScheduledScanService {
	if service != nil {
		service.agents = lookup
	}
	return service
}

func (service *ScheduledScanService) WithConfigResourceValidator(validator ScheduledScanConfigResourceValidator) *ScheduledScanService {
	if service != nil {
		service.resources = validator
	}
	return service
}

func (service *ScheduledScanService) Create(ctx context.Context, input *CreateScheduledScanInput) (*ScheduledScan, error) {
	if service == nil || service.store == nil {
		return nil, ErrScheduledScanInvalidArgument
	}
	if input == nil {
		return nil, fmt.Errorf("%w: request is required", ErrScheduledScanInvalidArgument)
	}
	if !input.InputSource.Valid() {
		return nil, fmt.Errorf("%w: %w", ErrScheduledScanInvalidArgument, scandomain.ErrInvalidInputSource)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrScheduledScanInvalidArgument)
	}
	scanWorkflowID, workflow, err := service.resolveCurrentScanWorkflow(input.ScanWorkflow)
	if err != nil {
		return nil, err
	}
	config, err := canonicalScheduledConfiguration(input.Configuration, workflow)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid configuration: %w", ErrScheduledScanInvalidArgument, err)
	}
	if err := service.validateConfigResources(ctx, workflow, config); err != nil {
		return nil, err
	}
	targetID, organizationID, err := resolveTargetOrganization(input.Target, input.Organization, input.TargetID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	agentID, err := service.resolveSelectedAgent(ctx, input.Agent)
	if err != nil {
		return nil, err
	}
	cron := strings.TrimSpace(input.CronExpression)
	if service.calculator == nil {
		return nil, fmt.Errorf("%w: schedule calculator is not configured", ErrScheduledScanInvalidArgument)
	}
	if err := service.calculator.Validate(cron); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScheduledScanInvalidArgument, err)
	}
	enabled := true
	if input.IsEnabled != nil {
		enabled = *input.IsEnabled
	}
	created, err := service.store.Create(ctx, &ScheduledScanCreate{
		Name:           name,
		ScanWorkflowID: scanWorkflowID,
		Configuration:  config,
		InputSource:    input.InputSource,
		TargetID:       targetID,
		AgentID:        agentID,
		OrganizationID: organizationID,
		CronExpression: cron,
		IsEnabled:      enabled,
	})
	if errors.Is(err, ErrInvalidScheduleRule) {
		return nil, fmt.Errorf("%w: %v", ErrScheduledScanInvalidArgument, err)
	}
	return created, err
}

func (service *ScheduledScanService) Update(ctx context.Context, id int, input *UpdateScheduledScanInput) (*ScheduledScan, error) {
	if service == nil || service.store == nil {
		return nil, ErrScheduledScanInvalidArgument
	}
	if id <= 0 || input == nil {
		return nil, fmt.Errorf("%w: invalid scheduled scan", ErrScheduledScanInvalidArgument)
	}
	if input.InputSource != nil && !input.InputSource.Valid() {
		return nil, fmt.Errorf("%w: %w", ErrScheduledScanInvalidArgument, scandomain.ErrInvalidInputSource)
	}
	var scanWorkflowID *string
	selectedScanWorkflow := ""
	var canonicalConfiguration map[string]any
	if input.ScanWorkflow != nil {
		parsed, workflow, err := service.resolveCurrentScanWorkflow(*input.ScanWorkflow)
		if err != nil {
			return nil, err
		}
		scanWorkflowID = &parsed
		selectedScanWorkflow = parsed
		if input.Configuration != nil {
			canonicalConfiguration, err = canonicalScheduledConfiguration(input.Configuration, workflow)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid configuration: %w", ErrScheduledScanInvalidArgument, err)
			}
			if err := service.validateConfigResources(ctx, workflow, canonicalConfiguration); err != nil {
				return nil, err
			}
		}
	}
	if input.Configuration != nil {
		if selectedScanWorkflow == "" {
			return nil, fmt.Errorf("%w: scanWorkflow is required when configuration is updated", ErrScheduledScanInvalidArgument)
		}
	}
	var name *string
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: name is required", ErrScheduledScanInvalidArgument)
		}
		name = &trimmed
	}
	var cron *string
	if input.CronExpression != nil {
		trimmed := strings.TrimSpace(*input.CronExpression)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: cronExpression is required", ErrScheduledScanInvalidArgument)
		}
		cron = &trimmed
	}
	if cron != nil {
		validationCron := "0 0 * * *"
		if cron != nil {
			validationCron = *cron
		}
		if service.calculator == nil {
			return nil, fmt.Errorf("%w: schedule calculator is not configured", ErrScheduledScanInvalidArgument)
		}
		if err := service.calculator.Validate(validationCron); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrScheduledScanInvalidArgument, err)
		}
	}
	targetID, organizationID, err := resolveTargetOrganizationPtr(input.Target, input.Organization, input.TargetID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	var agentID *int
	agentSet := input.Agent != nil
	if agentSet {
		agentID, err = service.resolveSelectedAgent(ctx, *input.Agent)
		if err != nil {
			return nil, err
		}
	}
	updated, err := service.store.Update(ctx, id, &ScheduledScanUpdate{
		Name:           name,
		ScanWorkflowID: scanWorkflowID,
		Configuration:  canonicalConfiguration,
		InputSource:    input.InputSource,
		TargetID:       targetID,
		AgentID:        agentID,
		AgentSet:       agentSet,
		OrganizationID: organizationID,
		CronExpression: cron,
		IsEnabled:      input.IsEnabled,
	})
	if errors.Is(err, ErrInvalidScheduleRule) {
		return nil, fmt.Errorf("%w: %v", ErrScheduledScanInvalidArgument, err)
	}
	return updated, err
}

// BatchUpdateStatus validates a bounded, unique status batch before delegating
// the atomic lifecycle transition to persistence.
func (service *ScheduledScanService) BatchUpdateStatus(ctx context.Context, updates []ScheduledScanStatusUpdate) (int, error) {
	if service == nil || service.store == nil {
		return 0, ErrScheduledScanInvalidArgument
	}
	if len(updates) == 0 || len(updates) > MaxScheduledScanBatchStatusUpdates {
		return 0, fmt.Errorf("%w: batch must contain between 1 and %d scheduled scans", ErrScheduledScanInvalidArgument, MaxScheduledScanBatchStatusUpdates)
	}

	normalized := append([]ScheduledScanStatusUpdate(nil), updates...)
	seen := make(map[int]struct{}, len(normalized))
	for _, update := range normalized {
		if update.ID <= 0 {
			return 0, fmt.Errorf("%w: scheduled scan ID must be positive", ErrScheduledScanInvalidArgument)
		}
		if _, exists := seen[update.ID]; exists {
			return 0, fmt.Errorf("%w: duplicate scheduled scan ID", ErrScheduledScanInvalidArgument)
		}
		seen[update.ID] = struct{}{}
	}
	sort.Slice(normalized, func(left, right int) bool {
		return normalized[left].ID < normalized[right].ID
	})

	updatedCount, err := service.store.BatchUpdateStatus(ctx, normalized)
	if errors.Is(err, ErrInvalidScheduleRule) {
		return 0, fmt.Errorf("%w: %v", ErrScheduledScanInvalidArgument, err)
	}
	return updatedCount, err
}

func (service *ScheduledScanService) List(ctx context.Context, query ScheduledScanListQuery) ([]ScheduledScan, int64, error) {
	return service.store.List(ctx, query)
}

func (service *ScheduledScanService) GetByID(ctx context.Context, id int) (*ScheduledScan, error) {
	if id <= 0 {
		return nil, ErrScheduledScanNotFound
	}
	return service.store.GetByID(ctx, id)
}

func (service *ScheduledScanService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrScheduledScanNotFound
	}
	return service.store.Delete(ctx, id)
}

func (service *ScheduledScanService) resolveCurrentScanWorkflow(workflow string) (string, *catalogdomain.ManagedScanWorkflow, error) {
	scanWorkflowID, err := resourcenames.ParseScanWorkflow(workflow)
	if err != nil {
		return "", nil, fmt.Errorf("%w: scanWorkflow must use scanWorkflows/{workflow}", ErrScheduledScanInvalidArgument)
	}
	if service.workflows == nil {
		return "", nil, fmt.Errorf("%w: scan workflow repository is not configured", ErrScheduledScanInvalidArgument)
	}
	resolved, err := service.workflows.GetScanWorkflowByID(scanWorkflowID)
	if err != nil {
		return "", nil, fmt.Errorf("%w: scanWorkflow %q not found: %v", ErrScheduledScanInvalidArgument, scanWorkflowID, err)
	}
	return scanWorkflowID, resolved, nil
}

func canonicalScheduledConfiguration(configuration map[string]any, workflow *catalogdomain.ManagedScanWorkflow) (map[string]any, error) {
	if workflow == nil {
		return nil, errors.New("scan workflow is required")
	}
	knownSteps := make(map[string]struct{})
	for _, stage := range workflow.Stages {
		for _, step := range stage.Steps {
			knownSteps[step.StepID] = struct{}{}
		}
	}
	configs, err := workflowconfig.Decode(configuration, knownSteps)
	if err != nil {
		return nil, err
	}
	return map[string]any(workflowconfig.EncodeStepConfigurations(configs)), nil
}

func (service *ScheduledScanService) validateConfigResources(ctx context.Context, workflow *catalogdomain.ManagedScanWorkflow, configuration map[string]any) error {
	if service == nil || service.resources == nil {
		return scanapp.NewConfigResourceInternalError("", "", "", errors.New("Scheduled Scan config resource validator is not configured"))
	}
	err := service.resources.Validate(ctx, scanapp.WorkflowConfigResourceValidationRequest{
		Workflow:      scheduledWorkflowValidationProjection(workflow),
		Configuration: cloneMap(configuration),
	})
	if scanapp.IsConfigResourceInputError(err) {
		return fmt.Errorf("%w: invalid configuration: %w", ErrScheduledScanInvalidArgument, err)
	}
	return err
}

func scheduledWorkflowValidationProjection(workflow *catalogdomain.ManagedScanWorkflow) scanapp.ScanCreateWorkflowManifest {
	if workflow == nil {
		return scanapp.ScanCreateWorkflowManifest{}
	}
	manifest := scanapp.ScanCreateWorkflowManifest{
		ScanWorkflowID: workflow.ScanWorkflowID,
		Stages:         make([]scanapp.ScanCreateWorkflowStage, 0, len(workflow.Stages)),
	}
	for _, stage := range workflow.Stages {
		projectedStage := scanapp.ScanCreateWorkflowStage{
			StageID: stage.StageID,
			Steps:   make([]scanapp.ScanCreateWorkflowStep, 0, len(stage.Steps)),
		}
		for _, step := range stage.Steps {
			projectedStage.Steps = append(projectedStage.Steps, scanapp.ScanCreateWorkflowStep{
				StepID: step.StepID, EngineID: step.EngineID,
			})
		}
		manifest.Stages = append(manifest.Stages, projectedStage)
	}
	return manifest
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func resolveTargetOrganization(target, organization string, targetID, organizationID *int) (*int, *int, error) {
	if strings.TrimSpace(target) != "" {
		parsed, err := httpdto.ParseResourceNameID(target, "targets")
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid target", ErrScheduledScanInvalidArgument)
		}
		targetID = &parsed
	}
	if strings.TrimSpace(organization) != "" {
		parsed, err := httpdto.ParseResourceNameID(organization, "organizations")
		if err != nil {
			return nil, nil, fmt.Errorf("%w: invalid organization", ErrScheduledScanInvalidArgument)
		}
		organizationID = &parsed
	}
	if targetID != nil && organizationID != nil {
		return nil, nil, fmt.Errorf("%w: target and organization are mutually exclusive", ErrScheduledScanInvalidArgument)
	}
	return targetID, organizationID, nil
}

func resolveTargetOrganizationPtr(target, organization *string, targetID, organizationID *int) (*int, *int, error) {
	targetValue := ""
	if target != nil {
		targetValue = *target
	}
	organizationValue := ""
	if organization != nil {
		organizationValue = *organization
	}
	return resolveTargetOrganization(targetValue, organizationValue, targetID, organizationID)
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (service *ScheduledScanService) resolveSelectedAgent(ctx context.Context, resource string) (*int, error) {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return nil, nil
	}
	id, err := httpdto.ParseResourceNameID(resource, "agents")
	if err != nil || id <= 0 || service.agents == nil {
		return nil, ErrScheduledScanAgentNotFound
	}
	exists, err := service.agents.AgentExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrScheduledScanAgentNotFound
	}
	return &id, nil
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
