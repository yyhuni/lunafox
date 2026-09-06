package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/contracts/enginecontract/engineexecution"
	enginecontract "github.com/yyhuni/lunafox/contracts/enginemanifest"
)

// ConfigResourceValidationCause is the closed failure classification exposed
// to HTTP adapters. The wrapped cause remains private diagnostic context.
type ConfigResourceValidationCause string

const (
	ConfigResourceUnavailable           ConfigResourceValidationCause = "unavailable"
	ConfigResourceValidationUnavailable ConfigResourceValidationCause = "validationUnavailable"
	ConfigResourceInternal              ConfigResourceValidationCause = "internal"
)

// ConfigResourceValidationError preserves one safe resource location while
// retaining the private lower-level cause for Server diagnostics.
type ConfigResourceValidationError struct {
	cause        ConfigResourceValidationCause
	field        string
	resourceKind string
	resourceName string
	err          error
}

func (e *ConfigResourceValidationError) Error() string {
	if e == nil {
		return ""
	}
	return "Engine configuration resource validation failed"
}

func (e *ConfigResourceValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// ConfigResourceValidationCause returns the stable boundary classification.
func (e *ConfigResourceValidationError) ConfigResourceValidationCause() string {
	if e == nil {
		return ""
	}
	return string(e.cause)
}

// ConfigResourceValidationField returns the canonical configuration path.
func (e *ConfigResourceValidationError) ConfigResourceValidationField() string {
	if e == nil {
		return ""
	}
	return e.field
}

// ConfigResourceValidationKind returns the manifest-declared resource kind.
func (e *ConfigResourceValidationError) ConfigResourceValidationKind() string {
	if e == nil {
		return ""
	}
	return e.resourceKind
}

// ConfigResourceValidationName returns the selected logical resource name.
func (e *ConfigResourceValidationError) ConfigResourceValidationName() string {
	if e == nil {
		return ""
	}
	return e.resourceName
}

func newConfigResourceValidationError(
	cause ConfigResourceValidationCause,
	field string,
	resourceKind string,
	resourceName string,
	err error,
) *ConfigResourceValidationError {
	if err == nil {
		err = errors.New("config resource validation failed without a diagnostic cause")
	}
	return &ConfigResourceValidationError{
		cause:        cause,
		field:        field,
		resourceKind: resourceKind,
		resourceName: resourceName,
		err:          err,
	}
}

// NewConfigResourceUnavailableError reports a confirmed deployment-state
// failure such as a missing Catalog row or mismatched file bytes.
func NewConfigResourceUnavailableError(field, resourceKind, resourceName string, err error) *ConfigResourceValidationError {
	return newConfigResourceValidationError(ConfigResourceUnavailable, field, resourceKind, resourceName, err)
}

// NewConfigResourceValidationUnavailableError reports a transient dependency
// failure that prevents the Server from deciding current availability.
func NewConfigResourceValidationUnavailableError(field, resourceKind, resourceName string, err error) *ConfigResourceValidationError {
	return newConfigResourceValidationError(ConfigResourceValidationUnavailable, field, resourceKind, resourceName, err)
}

// NewConfigResourceInternalError reports incomplete or inconsistent Server
// assembly without classifying the selected value as unavailable.
func NewConfigResourceInternalError(field, resourceKind, resourceName string, err error) *ConfigResourceValidationError {
	return newConfigResourceValidationError(ConfigResourceInternal, field, resourceKind, resourceName, err)
}

// ConfigResourceInputError marks a selected value that is malformed before an
// availability lookup. It remains an INVALID_ARGUMENT-style config failure.
type ConfigResourceInputError struct {
	Field        string
	ResourceKind string
	ResourceName string
	Err          error
}

func (e *ConfigResourceInputError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return "Engine configuration resource selection is invalid"
	}
	return fmt.Sprintf("%s contains an invalid Engine configuration resource selection", e.Field)
}

func (e *ConfigResourceInputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsConfigResourceInputError reports whether err contains malformed resource
// input rather than a deployment availability failure.
func IsConfigResourceInputError(err error) bool {
	var inputErr *ConfigResourceInputError
	return errors.As(err, &inputErr)
}

// ConfigResourceResolveRequest carries one canonical field and selected
// logical name into the deployment-owned resource resolver.
type ConfigResourceResolveRequest struct {
	Field        string
	ResourceKind string
	ResourceName string
}

// ConfigResourceResolver is the shared deployment-state boundary used by
// ordinary Scan planning and Scheduled Scan persistence validation.
type ConfigResourceResolver interface {
	ResolveConfigResource(context.Context, ConfigResourceResolveRequest) (PlanTaskWordlist, error)
}

// WorkflowConfigResourceValidationRequest is a detached current Workflow plus
// its complete canonical configuration.
type WorkflowConfigResourceValidationRequest struct {
	Workflow      ScanCreateWorkflowManifest
	Configuration map[string]any
}

// WorkflowConfigResourceValidator validates current resource selections
// without compiling or persisting execution plans.
type WorkflowConfigResourceValidator struct {
	packages  ScanCreateEnginePackageReader
	resources ConfigResourceResolver
}

func NewWorkflowConfigResourceValidator(
	packages ScanCreateEnginePackageReader,
	resources ConfigResourceResolver,
) (*WorkflowConfigResourceValidator, error) {
	if packages == nil {
		return nil, fmt.Errorf("current Engine Package reader is required")
	}
	if resources == nil {
		return nil, fmt.Errorf("config resource resolver is required")
	}
	return &WorkflowConfigResourceValidator{packages: packages, resources: resources}, nil
}

// Validate checks only enabled Workflow Steps and enabled configSections. A
// Scheduled Scan discards the returned observations; ordinary Scan planning
// resolves them again inside its existing persistence transaction.
func (validator *WorkflowConfigResourceValidator) Validate(ctx context.Context, request WorkflowConfigResourceValidationRequest) error {
	if validator == nil || validator.packages == nil || validator.resources == nil {
		return NewConfigResourceInternalError("", "", "", errors.New("config resource validator is not configured"))
	}
	normalized, err := normalizePlanTaskConfiguration(request.Configuration, request.Workflow)
	if err != nil {
		return &ConfigResourceInputError{Err: err}
	}
	for _, stage := range request.Workflow.Stages {
		for _, step := range stage.Steps {
			stepConfig, ok := normalized.Steps[step.StepID]
			if !ok {
				return &ConfigResourceInputError{Field: configStepFieldPath(step.StepID), Err: fmt.Errorf("Workflow Step configuration is missing")}
			}
			if !stepConfig.Enabled {
				continue
			}
			loaded, err := validator.packages.LoadEnginePackage(ctx, step.EngineID)
			if err != nil {
				return &PlanTaskError{Kind: PlanTaskPackageError, Err: err}
			}
			definition, err := enginecontract.NormalizeEngineDefinition(loaded.Package.Definition)
			if err != nil {
				return &PlanTaskError{Kind: PlanTaskPackageError, Err: fmt.Errorf("normalize current Engine Definition: %w", err)}
			}
			finalConfig, err := engineexecution.ValidateCompleteConfig(stepConfig.EngineConfig, definition.Execution)
			if err != nil {
				return &ConfigResourceInputError{Field: configStepFieldPath(step.StepID), Err: err}
			}
			if err := validateEnabledConfigResources(ctx, step.StepID, finalConfig, definition.Execution, validator.resources); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateEnabledConfigResources(
	ctx context.Context,
	stepID string,
	finalConfig map[string]any,
	definition engineexecution.ExecutionDefinition,
	resolver ConfigResourceResolver,
) error {
	for _, section := range definition.ConfigSections {
		sectionValue, ok := finalConfig[section.ID].(map[string]any)
		if !ok {
			return &ConfigResourceInputError{Field: configStepFieldPath(stepID), Err: fmt.Errorf("config section %q is missing", section.ID)}
		}
		enabled, ok := sectionValue["enabled"].(bool)
		if !ok {
			return &ConfigResourceInputError{Field: configStepFieldPath(stepID), Err: fmt.Errorf("config section %q enabled is invalid", section.ID)}
		}
		if !enabled {
			continue
		}
		for _, param := range section.Params {
			if param.Resource == nil {
				continue
			}
			field := configResourceFieldPath(stepID, section.ID, param.Key)
			name, ok := sectionValue[param.Key].(string)
			if !ok || strings.TrimSpace(name) == "" {
				return &ConfigResourceInputError{Field: field, ResourceKind: param.Resource.Kind, Err: fmt.Errorf("resource name must be a non-empty string")}
			}
			if _, err := resolver.ResolveConfigResource(ctx, ConfigResourceResolveRequest{
				Field: field, ResourceKind: param.Resource.Kind, ResourceName: name,
			}); err != nil {
				if IsConfigResourceInputError(err) {
					return err
				}
				var validationErr *ConfigResourceValidationError
				if !errors.As(err, &validationErr) {
					return NewConfigResourceInternalError(field, param.Resource.Kind, name, err)
				}
				return err
			}
		}
	}
	return nil
}

func configStepFieldPath(stepID string) string {
	return fmt.Sprintf("configuration.steps[%q].engineConfig", stepID)
}

func configResourceFieldPath(stepID, sectionID, paramKey string) string {
	return fmt.Sprintf("configuration.steps[%q].engineConfig.%s.%s", stepID, sectionID, paramKey)
}
