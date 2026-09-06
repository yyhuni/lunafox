package scanwiring

import (
	"context"
	"errors"
	"fmt"

	"github.com/yyhuni/lunafox/contracts/ocidistribution"
	"github.com/yyhuni/lunafox/server/internal/installedengines"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	catalogapp "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
	scaninfra "github.com/yyhuni/lunafox/server/internal/modules/scan/infrastructure"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

func NewScanQueryStoreAdapter(repo *scanrepo.ScanRepository) scanapp.ScanQueryStore {
	return newScanQueryStoreAdapter(repo)
}

func NewScanCommandStoreAdapter(repo *scanrepo.ScanRepository, effectivePolicyResolver scanapp.EffectiveBlacklistPolicyResolver) scanapp.ScanApplicationCommandStore {
	if repo == nil || effectivePolicyResolver == nil {
		return nil
	}
	return newScanCommandStoreAdapter(repo, effectivePolicyResolver)
}

func NewScanDomainRepositoryAdapter(repo *scanrepo.ScanRepository) scandomain.ScanRepository {
	return newScanDomainRepositoryAdapter(repo)
}

func NewScanStopStoreAdapter(repo *scanrepo.ScanRepository) scanapp.ScanStopStore {
	return newScanStopStoreAdapter(repo)
}

func NewScanTargetLookupAdapter(repo *catalogrepo.TargetRepository, targetCommander *catalogapp.TargetCommandService, organizationRepo ...*identityrepo.OrganizationRepository) scanapp.ScanCreateTargetLookup {
	return newScanTargetLookupAdapter(repo, targetCommander, organizationRepo...)
}

func NewScanTaskStoreAdapter(repo scanrepo.ScanTaskRepository) scanapp.ScanTaskStore {
	return newScanTaskStoreAdapter(repo)
}

func NewScanTaskRuntimeScanStoreAdapter(repo *scanrepo.ScanRepository) scanapp.ScanTaskRuntimeScanStore {
	return newScanTaskRuntimeScanStoreAdapter(repo)
}

type scanCreateAgentLookupAdapter struct{ repo agentdomain.AgentRepository }

func NewScanCreateAgentLookupAdapter(repo agentdomain.AgentRepository) scanapp.ScanCreateAgentLookup {
	return scanCreateAgentLookupAdapter{repo: repo}
}

func (adapter scanCreateAgentLookupAdapter) AgentExists(ctx context.Context, id int) (bool, error) {
	if adapter.repo == nil {
		return false, nil
	}
	_, err := adapter.repo.GetByID(ctx, id)
	if dberrors.IsRecordNotFound(err) || errors.Is(err, agentdomain.ErrAgentNotFound) {
		return false, nil
	}
	return err == nil, err
}

func NewScanApplicationService(
	queryStore scanapp.ScanQueryStore,
	commandStore scanapp.ScanApplicationCommandStore,
	domainRepository scandomain.ScanRepository,
	stopStore scanapp.ScanStopStore,
	notifier scanapp.TaskCancelNotifier,
	targetLookup scanapp.ScanCreateTargetLookup,
	agentLookup scanapp.ScanCreateAgentLookup,
	workflowRepo *catalogrepo.ScanWorkflowRepository,
	engineRegistrationRepo *catalogrepo.EngineRepository,
	installedPackageQuery installedengines.Query,
	configResources scanapp.ConfigResourceResolver,
	cloudflareAcceleration ...bool,
) (*scanapp.ScanFacade, error) {
	if len(cloudflareAcceleration) > 1 {
		return nil, fmt.Errorf("at most one Cloudflare acceleration setting is supported")
	}
	useCloudflareAcceleration := len(cloudflareAcceleration) == 1 && cloudflareAcceleration[0]
	commandService := scanapp.NewScanCommandService(domainRepository, nil)
	queryService := scanapp.NewScanQueryService(queryStore)
	lifecycleService := scanapp.NewLifecycleService(commandStore, stopStore, notifier)

	var lookupFn scanapp.TargetLookupFunc
	var quickTargetEnsurer scanapp.QuickTargetEnsurerFunc
	var organizationTargets scanapp.OrganizationTargetLookupFunc
	if targetLookup != nil {
		lookupFn = targetLookup.GetTargetRefByID
		quickTargetEnsurer = targetLookup.EnsureQuickTargets
		organizationTargets = targetLookup.ListOrganizationTargetRefs
	}
	createService := scanapp.NewScanCreateService(commandStore, lookupFn, quickTargetEnsurer, newScanCreateWorkflowReader(workflowRepo), newScanCreateEnginePackageReader(installedPackageQuery), organizationTargets).
		WithEngineRegistrationReader(newScanCreateEngineRegistrationReader(engineRegistrationRepo)).
		WithAgentLookup(agentLookup)
	compiler, err := newPlanTaskCompiler(installedPackageQuery, configResources, useCloudflareAcceleration)
	if err != nil {
		return nil, err
	}
	if err := createService.ConfigurePlanTask(compiler, productionPlanTaskLimitsProvider()); err != nil {
		return nil, fmt.Errorf("configure scan-create PlanTask policy: %w", err)
	}

	return scanapp.NewScanFacade(queryStore, commandStore, commandService, queryService, lifecycleService, createService), nil
}

func productionPlanTaskLimitsProvider() scanapp.PlanTaskLimitsProvider {
	return scanapp.DefaultPlanTaskLimitsProvider()
}

func NewConfigResourceValidationComponents(
	installedPackageQuery installedengines.Query,
	wordlistCatalog scaninfra.PlanTaskWordlistCatalog,
) (scanapp.ConfigResourceResolver, *scanapp.WorkflowConfigResourceValidator, error) {
	if installedPackageQuery == nil {
		return nil, nil, fmt.Errorf("installed Engine Package v2 query is required")
	}
	resolver, err := scaninfra.NewConfigResourceResolver(wordlistCatalog)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize config resource resolver: %w", err)
	}
	validator, err := scanapp.NewWorkflowConfigResourceValidator(newScanCreateEnginePackageReader(installedPackageQuery), resolver)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize Workflow config resource validator: %w", err)
	}
	return resolver, validator, nil
}

func newPlanTaskCompiler(installedPackageQuery installedengines.Query, configResources scanapp.ConfigResourceResolver, cloudflareAcceleration ...bool) (*scanapp.PlanTaskCompiler, error) {
	if len(cloudflareAcceleration) > 1 {
		return nil, fmt.Errorf("at most one Cloudflare acceleration setting is supported")
	}
	useCloudflareAcceleration := len(cloudflareAcceleration) == 1 && cloudflareAcceleration[0]
	exactReader, err := scaninfra.NewPlanTaskExactPackageReader(installedPackageQuery)
	if err != nil {
		return nil, fmt.Errorf("initialize exact Engine Package reader: %w", err)
	}
	if configResources == nil {
		return nil, fmt.Errorf("config resource resolver is required")
	}
	compiler, err := scanapp.NewPlanTaskCompiler(exactReader, configResources)
	if err != nil {
		return nil, fmt.Errorf("initialize PlanTask compiler: %w", err)
	}
	if useCloudflareAcceleration {
		compiler.WithRuntimeImageReferencesMapper(func(refs []string) ([]string, error) {
			acceleration, err := ocidistribution.BuildCloudflareAcceleration(refs)
			if err != nil {
				return nil, err
			}
			return acceleration.DownloadReferenceStrings(), nil
		})
	}
	return compiler, nil
}

func NewScanTaskApplicationService(
	taskStore scanapp.ScanTaskStore,
	scanTaskRuntimeScanStore scanapp.ScanTaskRuntimeScanStore,
	subdomainSnapshotRepo *snapshotrepo.SubdomainSnapshotRepository,
	hostPortSnapshotRepo *snapshotrepo.HostPortSnapshotRepository,
) *scanapp.ScanTaskFacade {
	// The active claim path reads the saved Engine Container plan. It does not
	// resolve runtime manifests or materialize a legacy assignment envelope.
	taskBridgeService := scanapp.NewScanTaskBridgeService(taskStore, scanTaskRuntimeScanStore)
	if claimStore, ok := taskStore.(scanapp.EngineExecutionClaimStore); ok {
		taskBridgeService.WithEngineExecutionClaimStore(claimStore)
	}
	return scanapp.NewScanTaskFacade(taskBridgeService)
}
