package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	workflowconfig "github.com/yyhuni/lunafox/contracts/scanworkflow/configuration"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

type preparedScanCreatePlan struct {
	manifest         ScanCreateWorkflowManifest
	normalizedConfig dynamicConfiguration
}

func (service *ScanCreateService) CreateQuick(ctx context.Context, input *CreateQuickInput) (*CreateQuickResult, error) {
	if input == nil {
		return nil, ErrCreateInvalidConfig
	}
	if err := validateScanTriggerType(input.TriggerType); err != nil {
		return nil, err
	}
	if err := validateInputSource(input.InputSource); err != nil {
		return nil, err
	}
	if len(input.Targets) == 0 {
		return nil, ErrNoTargetsForScan
	}
	plan, err := service.prepareScanCreatePlan(ctx, input.ScanWorkflow, input.Configuration)
	if err != nil {
		return nil, err
	}
	if err := service.validateSelectedAgent(ctx, input.AgentID); err != nil {
		return nil, err
	}

	if service.quickTargetEnsurer == nil {
		return nil, ErrCreateTargetLookupNotReady
	}
	resolution, err := service.quickTargetEnsurer(ctx, input.Targets)
	if err != nil {
		return nil, err
	}
	if resolution == nil || len(resolution.Targets) == 0 {
		return nil, ErrNoTargetsForScan
	}

	result := &CreateQuickResult{
		Scans:       make([]CreateScan, 0, len(resolution.Targets)),
		TargetStats: resolution.TargetStats,
		Errors:      append([]QuickTargetError(nil), resolution.Errors...),
	}
	for index := range resolution.Targets {
		target := resolution.Targets[index]
		scan, err := service.createPreparedScan(ctx, &target, plan.manifest, plan.normalizedConfig, input.AgentID, input.InputSource, input.TriggerType)
		if err != nil {
			return nil, err
		}
		result.Scans = append(result.Scans, *scan)
	}
	return result, nil
}

func (service *ScanCreateService) CreateBatch(ctx context.Context, input *CreateBatchInput) (*CreateBatchResult, error) {
	if input == nil {
		return nil, ErrCreateInvalidConfig
	}
	if err := validateScanTriggerType(input.TriggerType); err != nil {
		return nil, err
	}
	if err := validateInputSource(input.InputSource); err != nil {
		return nil, err
	}
	if len(input.Requests) == 0 {
		return nil, ErrNoTargetsForScan
	}
	plan, err := service.prepareScanCreatePlan(ctx, input.ScanWorkflow, input.Configuration)
	if err != nil {
		return nil, err
	}
	if err := service.validateSelectedAgent(ctx, input.AgentID); err != nil {
		return nil, err
	}
	if service.targetLookup == nil {
		return nil, ErrCreateTargetLookupNotReady
	}
	if batchHasOrganizationScope(input.Requests) && service.organizationTargets == nil {
		return nil, ErrCreateTargetLookupNotReady
	}

	result := &CreateBatchResult{Scans: make([]CreateScan, 0, len(input.Requests))}
	seenTargets := map[int]struct{}{}
	for index, item := range input.Requests {
		targets, resolved := service.resolveBatchItemTargets(ctx, index, item, result)
		if !resolved {
			continue
		}
		for targetIndex := range targets {
			target := targets[targetIndex]
			if _, ok := seenTargets[target.ID]; ok {
				result.Skipped = append(result.Skipped, CreateBatchItemOutcome{Index: index, TargetID: target.ID, OrganizationID: item.OrganizationID, Reason: "DUPLICATE_TARGET", Message: "target already included in batch"})
				continue
			}
			seenTargets[target.ID] = struct{}{}
			scan, err := service.createPreparedScan(ctx, &target, plan.manifest, plan.normalizedConfig, input.AgentID, input.InputSource, input.TriggerType)
			if err != nil {
				if errors.Is(err, ErrCreateTargetNotFound) {
					result.Failed = append(result.Failed, CreateBatchItemOutcome{Index: index, TargetID: target.ID, OrganizationID: item.OrganizationID, Reason: "TARGET_NOT_FOUND", Message: err.Error()})
					continue
				}
				result.CreatedCount = len(result.Scans)
				return result, err
			}
			result.Scans = append(result.Scans, *scan)
		}
	}
	result.CreatedCount = len(result.Scans)
	return result, nil
}

func (service *ScanCreateService) resolveBatchItemTargets(ctx context.Context, index int, item CreateBatchItem, result *CreateBatchResult) ([]TargetRef, bool) {
	if item.TargetID > 0 {
		target, err := service.targetLookup(ctx, item.TargetID)
		if err != nil || target == nil {
			message := "target not found"
			if err != nil {
				message = err.Error()
			}
			result.Failed = append(result.Failed, CreateBatchItemOutcome{Index: index, TargetID: item.TargetID, Reason: "TARGET_NOT_FOUND", Message: message})
			return nil, false
		}
		return []TargetRef{*target}, true
	}
	if item.OrganizationID > 0 {
		targets, err := service.organizationTargets(ctx, item.OrganizationID)
		if err != nil {
			result.Failed = append(result.Failed, CreateBatchItemOutcome{Index: index, OrganizationID: item.OrganizationID, Reason: "ORGANIZATION_NOT_FOUND", Message: err.Error()})
			return nil, false
		}
		if len(targets) == 0 {
			result.Skipped = append(result.Skipped, CreateBatchItemOutcome{Index: index, OrganizationID: item.OrganizationID, Reason: "ORGANIZATION_EMPTY", Message: "organization has no targets"})
			return nil, false
		}
		return targets, true
	}
	result.Failed = append(result.Failed, CreateBatchItemOutcome{Index: index, Reason: "INVALID_SCOPE", Message: "target or organization is required"})
	return nil, false
}

func batchHasOrganizationScope(items []CreateBatchItem) bool {
	for _, item := range items {
		if item.OrganizationID > 0 {
			return true
		}
	}
	return false
}

func (service *ScanCreateService) prepareScanCreatePlan(ctx context.Context, scanWorkflow string, configuration map[string]any) (*preparedScanCreatePlan, error) {
	manifest, err := service.validateRequestedScanWorkflow(ctx, scanWorkflow)
	if err != nil {
		return nil, err
	}
	normalizedConfig, err := normalizePlanTaskConfiguration(configuration, manifest)
	if err != nil {
		return nil, err
	}
	if service.planTaskCompiler == nil {
		return nil, fmt.Errorf("scan-create PlanTask compiler is not configured")
	}
	if err := validatePlanTaskLimits(service.planTaskLimits); err != nil {
		return nil, fmt.Errorf("scan-create PlanTask limits policy: %w", err)
	}
	return &preparedScanCreatePlan{manifest: manifest, normalizedConfig: normalizedConfig}, nil
}

func (service *ScanCreateService) createPreparedScan(
	ctx context.Context,
	target *TargetRef,
	manifest ScanCreateWorkflowManifest,
	normalizedConfig dynamicConfiguration,
	agentID *int,
	inputSource InputSource,
	triggerType ScanTriggerType,
) (*CreateScan, error) {
	if target == nil {
		return nil, ErrCreateTargetNotFound
	}
	scan := &CreateScan{
		TargetID:       target.ID,
		ScanWorkflowID: manifest.ScanWorkflowID,
		Configuration:  encodeDynamicConfiguration(normalizedConfig),
		InputSource:    inputSource,
		TriggerType:    triggerType,
		AssignmentMode: assignmentModeForAgent(agentID),
		AgentID:        cloneIntPtr(agentID),
		Status:         CreateScanStatusPending,
	}

	scanTasks, err := buildPlanTaskScanTasks(manifest)
	if err != nil {
		return nil, err
	}
	scan.ScanTasks = scanTasks

	scan.Status = CreateScanStatusPending
	if service.scanStore == nil {
		return nil, fmt.Errorf("scan create store is not configured")
	}
	if err := service.scanStore.CreateWithScanTasksAndPlans(ctx, scan, service.planTaskFinalizer(ctx, manifest, normalizedConfig, target, len(scanTasks), scan)); err != nil {
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrCreateTargetNotFound
		}
		return nil, err
	}
	scan.Target = &TargetRef{
		ID:            target.ID,
		Name:          target.Name,
		Type:          target.Type,
		CreatedAt:     timeutil.ToUTC(target.CreatedAt),
		LastScannedAt: timeutil.ToUTCPtr(target.LastScannedAt),
		DeletedAt:     timeutil.ToUTCPtr(target.DeletedAt),
	}

	return scan, nil
}

func (service *ScanCreateService) planTaskFinalizer(
	ctx context.Context,
	manifest ScanCreateWorkflowManifest,
	normalizedConfig dynamicConfiguration,
	target *TargetRef,
	totalTasks int,
	scan *CreateScan,
) ScanCreateTaskFinalizer {
	firstExecutableStageOrder := 0
	allSkipped := true
	completedTasks := 0
	return func(scanID, taskID int, task *CreateScanTask) error {
		if task == nil {
			return fmt.Errorf("scan task is required")
		}
		stepConfig, ok := normalizedConfig.Steps[task.StepID]
		if !ok {
			return &PlanTaskError{Kind: PlanTaskInvalidRequest, Err: fmt.Errorf("workflow Step %q has no normalized configuration", task.StepID)}
		}
		// Every topology Step must resolve through the lightweight registration
		// boundary before the planner touches an exact package. This keeps a
		// disabled Step package-free while also rejecting broken references
		// atomically for enabled and disabled branches alike.
		if service.engineRegistrationReader != nil {
			exists, err := service.engineRegistrationReader.EngineRegistrationExists(ctx, task.EngineID)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("%w: step %q engine %q", ErrCreateScanWorkflowEngineUnavailable, task.StepID, task.EngineID)
			}
		} else if !stepConfig.Enabled {
			// Disabled branches have no package fallback; production wiring must
			// provide the lightweight registration reader for this gate.
			return fmt.Errorf("engine registration reader is not configured")
		}
		if !stepConfig.Enabled {
			task.Status = CreateTaskStatusSkipped
			task.SkipReason = string(scandomain.TaskSkipReasonUserDisabled)
			task.EngineConfig = nil
			task.TaskExecutionConfig = nil
			task.ResolvedExecutionPlan = nil
			completedTasks++
			if completedTasks == totalTasks && allSkipped {
				scan.Status = CreateScanStatusSucceeded
			}
			return nil
		}
		if service.enginePackageReader == nil {
			return fmt.Errorf("enabled Step package reader is not configured")
		}
		enginePackage, err := service.enginePackageReader.LoadEnginePackage(ctx, task.EngineID)
		if err != nil {
			return &PlanTaskError{Kind: PlanTaskPackageError, Err: err}
		}
		if enginePackage.Package.Identity.EngineID != task.EngineID || enginePackage.Package.Identity.PackageDigest == "" {
			return &PlanTaskError{Kind: PlanTaskPackageError, Err: fmt.Errorf("exact Engine Package v2 %q is not pinned", task.EngineID)}
		}
		rawConfig := rawTaskConfigForPlan(normalizedConfig, task.StepID)
		request := PlanTaskRequest{
			Execution:  fmt.Sprintf("executions/scan-%d-task-%d", scanID, taskID),
			Task:       resourcenames.Task(scanID, taskID),
			Scan:       fmt.Sprintf("scans/%d", scanID),
			Workflow:   resourcenames.ScanWorkflow(manifest.ScanWorkflowID),
			StageID:    task.StageID,
			StepID:     task.StepID,
			EngineID:   task.EngineID,
			Target:     PlanTaskTarget{Resource: resourcenames.Target(target.ID), Type: target.Type, Value: target.Name},
			TaskConfig: rawConfig,
			Package:    enginePackage.Package.Identity,
			Limits:     service.planTaskLimits,

			operationContext: ctx,
		}
		outcome, err := service.planTaskCompiler.PlanTask(request)
		if err != nil {
			return err
		}
		switch resolved := outcome.(type) {
		case SkippedPlanTask:
			task.Status = CreateTaskStatusSkipped
			task.SkipReason = resolved.Reason
			task.ResolvedExecutionPlan = nil
			task.EngineConfig = nil
			task.TaskExecutionConfig = nil
		case ExecutablePlanTask:
			encoded, err := EncodeResolvedExecutionPlan(resolved.Plan)
			if err != nil {
				return err
			}
			task.ResolvedExecutionPlan = encoded
			task.SkipReason = ""
			allSkipped = false
			if firstExecutableStageOrder != 0 && firstExecutableStageOrder != task.StageOrder {
				task.Status = CreateTaskStatusBlocked
			} else {
				task.Status = CreateTaskStatusPending
				firstExecutableStageOrder = task.StageOrder
			}
		default:
			return &PlanTaskError{Kind: PlanTaskProtocolError, Err: fmt.Errorf("PlanTask returned unknown outcome %T", outcome)}
		}
		completedTasks++
		if completedTasks == totalTasks && allSkipped {
			scan.Status = CreateScanStatusSucceeded
		}
		return nil
	}
}

func encodeDynamicConfiguration(configuration dynamicConfiguration) map[string]any {
	steps := make(map[string]workflowconfig.StepConfiguration, len(configuration.Steps))
	for stepID, stepConfig := range configuration.Steps {
		steps[stepID] = workflowconfig.StepConfiguration{
			Enabled:      stepConfig.Enabled,
			EngineConfig: cloneMap(stepConfig.EngineConfig),
		}
	}
	return map[string]any(workflowconfig.EncodeStepConfigurations(steps))
}
