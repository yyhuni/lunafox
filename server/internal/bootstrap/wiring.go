package bootstrap

import (
	"fmt"

	assetwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/asset"
	catalogwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/catalog"
	identitywiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/identity"
	scanwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/scan"
	scanlogwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/scanlog"
	securitywiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/security"
	snapshotwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/snapshot"
	workerwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/worker"
	"github.com/yyhuni/lunafox/server/internal/config"
	runtimesvc "github.com/yyhuni/lunafox/server/internal/grpc/runtime/service"
	agentservice "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	agenthandler "github.com/yyhuni/lunafox/server/internal/modules/agent/handler"
	agentinfra "github.com/yyhuni/lunafox/server/internal/modules/agent/infrastructure"
	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
	assetservice "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assethandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler"
	directoryhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/directory"
	endpointhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/endpoint"
	hostporthandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/host_port"
	screenshothandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/screenshot"
	subdomainhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/subdomain"
	websitehandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/website"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	catalogservice "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	cataloghandler "github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	identityservice "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identityhandler "github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	scanhandler "github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	securityservice "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	securityhandler "github.com/yyhuni/lunafox/server/internal/modules/security/handler"
	securityrepo "github.com/yyhuni/lunafox/server/internal/modules/security/repository"
	snapshotservice "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
	"github.com/yyhuni/lunafox/server/internal/preset"
)

type deps struct {
	healthHandler        *assethandler.HealthHandler
	authHandler          *identityhandler.AuthHandler
	userHandler          *identityhandler.UserHandler
	orgHandler           *identityhandler.OrganizationHandler
	targetHandler        *cataloghandler.TargetHandler
	engineHandler        *cataloghandler.EngineHandler
	wordlistHandler      *cataloghandler.WordlistHandler
	websiteHandler       *websitehandler.WebsiteHandler
	subdomainHandler     *subdomainhandler.SubdomainHandler
	endpointHandler      *endpointhandler.EndpointHandler
	directoryHandler     *directoryhandler.DirectoryHandler
	hostPortHandler      *hostporthandler.HostPortHandler
	screenshotHandler    *screenshothandler.ScreenshotHandler
	vulnerabilityHandler *securityhandler.VulnerabilityHandler
	scanHandler          *scanhandler.ScanHandler
	scanLogHandler       *scanhandler.ScanLogHandler

	agentHandler    *agenthandler.AgentHandler
	agentLogHandler *agenthandler.AgentLogHandler

	websiteSnapshotHandler       *snapshothandler.WebsiteSnapshotHandler
	subdomainSnapshotHandler     *snapshothandler.SubdomainSnapshotHandler
	endpointSnapshotHandler      *snapshothandler.EndpointSnapshotHandler
	directorySnapshotHandler     *snapshothandler.DirectorySnapshotHandler
	hostPortSnapshotHandler      *snapshothandler.HostPortSnapshotHandler
	screenshotSnapshotHandler    *snapshothandler.ScreenshotSnapshotHandler
	vulnerabilitySnapshotHandler *snapshothandler.VulnerabilitySnapshotHandler
	presetHandler                *cataloghandler.PresetHandler

	agentRepo    agentdomain.AgentRepository
	scanTaskRepo scanrepo.ScanTaskRepository

	agentRuntimeService *agentservice.AgentRuntimeService
	agentTaskService    *agentservice.AgentTaskService

	workerProviderConfigService *catalogservice.WorkerProviderConfigService
	wordlistService             *catalogservice.WordlistFacade
	subdomainSnapshotService    *snapshotservice.SubdomainSnapshotFacade
	websiteSnapshotService      *snapshotservice.WebsiteSnapshotFacade
	endpointSnapshotService     *snapshotservice.EndpointSnapshotFacade
	hostPortSnapshotService     *snapshotservice.HostPortSnapshotFacade
	runtimeStreamRegistry       *runtimesvc.AgentStreamRegistry
}

type repositoryBundle struct {
	userRepo                      *identityrepo.UserRepository
	orgRepo                       *identityrepo.OrganizationRepository
	targetRepo                    *catalogrepo.TargetRepository
	engineRepo                    *catalogrepo.EngineRepository
	wordlistRepo                  *catalogrepo.WordlistRepository
	websiteRepo                   *assetrepo.WebsiteRepository
	subdomainRepo                 *assetrepo.SubdomainRepository
	endpointRepo                  *assetrepo.EndpointRepository
	directoryRepo                 *assetrepo.DirectoryRepository
	hostPortRepo                  *assetrepo.HostPortRepository
	screenshotRepo                *assetrepo.ScreenshotRepository
	vulnerabilityRepo             securityrepo.VulnerabilityRepository
	scanRepo                      *scanrepo.ScanRepository
	scanLogRepo                   *scanrepo.ScanLogRepository
	subfinderProviderSettingsRepo *catalogrepo.SubfinderProviderSettingsRepository
	websiteSnapshotRepo           *snapshotrepo.WebsiteSnapshotRepository
	subdomainSnapshotRepo         *snapshotrepo.SubdomainSnapshotRepository
	endpointSnapshotRepo          *snapshotrepo.EndpointSnapshotRepository
	directorySnapshotRepo         *snapshotrepo.DirectorySnapshotRepository
	hostPortSnapshotRepo          *snapshotrepo.HostPortSnapshotRepository
	screenshotSnapshotRepo        *snapshotrepo.ScreenshotSnapshotRepository
	vulnerabilitySnapshotRepo     *snapshotrepo.VulnerabilitySnapshotRepository
	agentRepo                     agentdomain.AgentRepository
	registrationTokenRepo         agentdomain.RegistrationTokenRepository
	scanTaskRepo                  scanrepo.ScanTaskRepository
}

type identityModuleHandlers struct {
	authHandler *identityhandler.AuthHandler
	userHandler *identityhandler.UserHandler
	orgHandler  *identityhandler.OrganizationHandler
}

type catalogModuleHandlers struct {
	targetHandler   *cataloghandler.TargetHandler
	engineHandler   *cataloghandler.EngineHandler
	wordlistHandler *cataloghandler.WordlistHandler

	wordlistService *catalogservice.WordlistFacade
}

type assetModuleWiring struct {
	websiteHandler    *websitehandler.WebsiteHandler
	subdomainHandler  *subdomainhandler.SubdomainHandler
	endpointHandler   *endpointhandler.EndpointHandler
	directoryHandler  *directoryhandler.DirectoryHandler
	hostPortHandler   *hostporthandler.HostPortHandler
	screenshotHandler *screenshothandler.ScreenshotHandler

	websiteSvc    *assetservice.WebsiteFacade
	subdomainSvc  *assetservice.SubdomainFacade
	endpointSvc   *assetservice.EndpointFacade
	directorySvc  *assetservice.DirectoryFacade
	hostPortSvc   *assetservice.HostPortFacade
	screenshotSvc *assetservice.ScreenshotFacade
}

type securityModuleWiring struct {
	vulnerabilityHandler *securityhandler.VulnerabilityHandler
	vulnerabilitySvc     *securityservice.VulnerabilityFacade
}

type scanModuleWiring struct {
	scanHandler     *scanhandler.ScanHandler
	scanLogHandler  *scanhandler.ScanLogHandler
	scanTaskRuntime agentservice.ScanTaskRuntimePort
}

type workerModuleHandlers struct {
	workerProviderConfigService *catalogservice.WorkerProviderConfigService
}

type agentModuleHandlers struct {
	agentHandler    *agenthandler.AgentHandler
	agentLogHandler *agenthandler.AgentLogHandler

	agentRuntimeService *agentservice.AgentRuntimeService
	agentTaskService    *agentservice.AgentTaskService
}

type snapshotModuleHandlers struct {
	websiteSnapshotHandler       *snapshothandler.WebsiteSnapshotHandler
	subdomainSnapshotHandler     *snapshothandler.SubdomainSnapshotHandler
	endpointSnapshotHandler      *snapshothandler.EndpointSnapshotHandler
	directorySnapshotHandler     *snapshothandler.DirectorySnapshotHandler
	hostPortSnapshotHandler      *snapshothandler.HostPortSnapshotHandler
	screenshotSnapshotHandler    *snapshothandler.ScreenshotSnapshotHandler
	vulnerabilitySnapshotHandler *snapshothandler.VulnerabilitySnapshotHandler
	subdomainSnapshotService     *snapshotservice.SubdomainSnapshotFacade
	websiteSnapshotService       *snapshotservice.WebsiteSnapshotFacade
	endpointSnapshotService      *snapshotservice.EndpointSnapshotFacade
	hostPortSnapshotService      *snapshotservice.HostPortSnapshotFacade
}

func buildDependencies(infra *infra, cfg *config.Config) *deps {
	repos := newRepositoryBundle(infra)
	streamRegistry := runtimesvc.NewAgentStreamRegistry()
	grpcPublisher := runtimesvc.NewAgentRuntimeEventPublisher(streamRegistry)

	identity := wireIdentityModule(repos, infra)
	catalog := wireCatalogModule(repos, cfg)
	asset := wireAssetModule(repos)
	security := wireSecurityModule(repos)
	scan := wireScanModule(repos, infra, grpcPublisher)
	worker := wireWorkerModule(repos)
	agent := wireAgentModule(repos, infra, cfg, scan.scanTaskRuntime, grpcPublisher)
	snapshot := wireSnapshotModule(repos, asset, security)
	presetHandler := cataloghandler.NewPresetHandler(preset.NewService(infra.presetLoader))

	return &deps{
		healthHandler:        assethandler.NewHealthHandler(infra.db, infra.redisClient),
		authHandler:          identity.authHandler,
		userHandler:          identity.userHandler,
		orgHandler:           identity.orgHandler,
		targetHandler:        catalog.targetHandler,
		engineHandler:        catalog.engineHandler,
		wordlistHandler:      catalog.wordlistHandler,
		websiteHandler:       asset.websiteHandler,
		subdomainHandler:     asset.subdomainHandler,
		endpointHandler:      asset.endpointHandler,
		directoryHandler:     asset.directoryHandler,
		hostPortHandler:      asset.hostPortHandler,
		screenshotHandler:    asset.screenshotHandler,
		vulnerabilityHandler: security.vulnerabilityHandler,
		scanHandler:          scan.scanHandler,
		scanLogHandler:       scan.scanLogHandler,

		agentHandler:    agent.agentHandler,
		agentLogHandler: agent.agentLogHandler,

		websiteSnapshotHandler:       snapshot.websiteSnapshotHandler,
		subdomainSnapshotHandler:     snapshot.subdomainSnapshotHandler,
		endpointSnapshotHandler:      snapshot.endpointSnapshotHandler,
		directorySnapshotHandler:     snapshot.directorySnapshotHandler,
		hostPortSnapshotHandler:      snapshot.hostPortSnapshotHandler,
		screenshotSnapshotHandler:    snapshot.screenshotSnapshotHandler,
		vulnerabilitySnapshotHandler: snapshot.vulnerabilitySnapshotHandler,
		presetHandler:                presetHandler,

		agentRepo:    repos.agentRepo,
		scanTaskRepo: repos.scanTaskRepo,

		agentRuntimeService:         agent.agentRuntimeService,
		agentTaskService:            agent.agentTaskService,
		workerProviderConfigService: worker.workerProviderConfigService,
		wordlistService:             catalog.wordlistService,
		subdomainSnapshotService:    snapshot.subdomainSnapshotService,
		websiteSnapshotService:      snapshot.websiteSnapshotService,
		endpointSnapshotService:     snapshot.endpointSnapshotService,
		hostPortSnapshotService:     snapshot.hostPortSnapshotService,
		runtimeStreamRegistry:       streamRegistry,
	}
}

func newRepositoryBundle(infra *infra) *repositoryBundle {
	db := infra.db
	return &repositoryBundle{
		userRepo:                      identityrepo.NewUserRepository(db),
		orgRepo:                       identityrepo.NewOrganizationRepository(db),
		targetRepo:                    catalogrepo.NewTargetRepository(db),
		engineRepo:                    catalogrepo.NewEngineRepository(db),
		wordlistRepo:                  catalogrepo.NewWordlistRepository(db),
		websiteRepo:                   assetrepo.NewWebsiteRepository(db),
		subdomainRepo:                 assetrepo.NewSubdomainRepository(db),
		endpointRepo:                  assetrepo.NewEndpointRepository(db),
		directoryRepo:                 assetrepo.NewDirectoryRepository(db),
		hostPortRepo:                  assetrepo.NewHostPortRepository(db),
		screenshotRepo:                assetrepo.NewScreenshotRepository(db),
		vulnerabilityRepo:             securityrepo.NewVulnerabilityRepository(db),
		scanRepo:                      scanrepo.NewScanRepository(db),
		scanLogRepo:                   scanrepo.NewScanLogRepository(db),
		subfinderProviderSettingsRepo: catalogrepo.NewSubfinderProviderSettingsRepository(db),
		websiteSnapshotRepo:           snapshotrepo.NewWebsiteSnapshotRepository(db),
		subdomainSnapshotRepo:         snapshotrepo.NewSubdomainSnapshotRepository(db),
		endpointSnapshotRepo:          snapshotrepo.NewEndpointSnapshotRepository(db),
		directorySnapshotRepo:         snapshotrepo.NewDirectorySnapshotRepository(db),
		hostPortSnapshotRepo:          snapshotrepo.NewHostPortSnapshotRepository(db),
		screenshotSnapshotRepo:        snapshotrepo.NewScreenshotSnapshotRepository(db),
		vulnerabilitySnapshotRepo:     snapshotrepo.NewVulnerabilitySnapshotRepository(db),
		agentRepo:                     agentrepo.NewAgentRepository(db),
		registrationTokenRepo:         agentrepo.NewRegistrationTokenRepository(db),
		scanTaskRepo:                  scanrepo.NewScanTaskRepository(db),
	}
}

func wireIdentityModule(repos *repositoryBundle, infra *infra) identityModuleHandlers {
	identityUserQueryStore := identitywiring.NewIdentityUserQueryStoreAdapter(repos.userRepo)
	identityUserCommandStore := identitywiring.NewIdentityUserCommandStoreAdapter(repos.userRepo)
	identityOrgQueryStore := identitywiring.NewIdentityOrganizationQueryStoreAdapter(repos.orgRepo)
	identityOrgCommandStore := identitywiring.NewIdentityOrganizationCommandStoreAdapter(repos.orgRepo)
	identityAuthUserStore := identitywiring.NewIdentityAuthUserStoreAdapter(repos.userRepo)

	userSvc := identityservice.NewUserFacade(identityUserQueryStore, identityUserCommandStore)
	orgSvc := identityservice.NewOrganizationFacade(identityOrgQueryStore, identityOrgCommandStore)
	authSvc := identityservice.NewAuthFacade(identityAuthUserStore, infra.jwtManager)

	return identityModuleHandlers{
		authHandler: identityhandler.NewAuthHandler(authSvc),
		userHandler: identityhandler.NewUserHandler(userSvc),
		orgHandler:  identityhandler.NewOrganizationHandler(orgSvc),
	}
}

func wireCatalogModule(repos *repositoryBundle, cfg *config.Config) catalogModuleHandlers {
	catalogTargetQueryStore := catalogwiring.NewCatalogTargetQueryStoreAdapter(repos.targetRepo)
	catalogTargetCommandStore := catalogwiring.NewCatalogTargetCommandStoreAdapter(repos.targetRepo)
	catalogEngineQueryStore := catalogwiring.NewCatalogEngineQueryStoreAdapter(repos.engineRepo)
	catalogEngineCommandStore := catalogwiring.NewCatalogEngineCommandStoreAdapter(repos.engineRepo)
	catalogWordlistQueryStore := catalogwiring.NewCatalogWordlistQueryStoreAdapter(repos.wordlistRepo)
	catalogWordlistCommandStore := catalogwiring.NewCatalogWordlistCommandStoreAdapter(repos.wordlistRepo)
	catalogOrganizationTargetBindingStore := catalogwiring.NewCatalogOrganizationTargetBindingStoreAdapter(repos.orgRepo)

	targetSvc := catalogservice.NewTargetFacade(catalogTargetQueryStore, catalogTargetCommandStore, catalogOrganizationTargetBindingStore)
	engineSvc := catalogservice.NewEngineFacade(catalogEngineQueryStore, catalogEngineCommandStore)
	wordlistSvc := catalogservice.NewWordlistFacade(catalogWordlistQueryStore, catalogWordlistCommandStore, cfg.Storage.WordlistsBasePath)

	return catalogModuleHandlers{
		targetHandler:   cataloghandler.NewTargetHandler(targetSvc),
		engineHandler:   cataloghandler.NewEngineHandler(engineSvc),
		wordlistHandler: cataloghandler.NewWordlistHandler(wordlistSvc),
		wordlistService: wordlistSvc,
	}
}

func wireAssetModule(repos *repositoryBundle) assetModuleWiring {
	assetTargetLookup := assetwiring.NewAssetTargetLookupAdapter(repos.targetRepo)
	assetWebsiteStore := assetwiring.NewAssetWebsiteStoreAdapter(repos.websiteRepo)
	assetSubdomainStore := assetwiring.NewAssetSubdomainStoreAdapter(repos.subdomainRepo)
	assetEndpointStore := assetwiring.NewAssetEndpointStoreAdapter(repos.endpointRepo)
	assetDirectoryStore := assetwiring.NewAssetDirectoryStoreAdapter(repos.directoryRepo)
	assetHostPortStore := assetwiring.NewAssetHostPortStoreAdapter(repos.hostPortRepo)
	assetScreenshotStore := assetwiring.NewAssetScreenshotStoreAdapter(repos.screenshotRepo)

	websiteSvc := assetservice.NewWebsiteFacade(assetWebsiteStore, assetTargetLookup)
	subdomainSvc := assetservice.NewSubdomainFacade(assetSubdomainStore, assetTargetLookup)
	endpointSvc := assetservice.NewEndpointFacade(assetEndpointStore, assetTargetLookup)
	directorySvc := assetservice.NewDirectoryFacade(assetDirectoryStore, assetTargetLookup)
	hostPortSvc := assetservice.NewHostPortFacade(assetHostPortStore, assetTargetLookup)
	screenshotSvc := assetservice.NewScreenshotFacade(assetScreenshotStore, assetTargetLookup)

	return assetModuleWiring{
		websiteHandler:    websitehandler.NewWebsiteHandler(websiteSvc),
		subdomainHandler:  subdomainhandler.NewSubdomainHandler(subdomainSvc),
		endpointHandler:   endpointhandler.NewEndpointHandler(endpointSvc),
		directoryHandler:  directoryhandler.NewDirectoryHandler(directorySvc),
		hostPortHandler:   hostporthandler.NewHostPortHandler(hostPortSvc),
		screenshotHandler: screenshothandler.NewScreenshotHandler(screenshotSvc),
		websiteSvc:        websiteSvc,
		subdomainSvc:      subdomainSvc,
		endpointSvc:       endpointSvc,
		directorySvc:      directorySvc,
		hostPortSvc:       hostPortSvc,
		screenshotSvc:     screenshotSvc,
	}
}

func wireSecurityModule(repos *repositoryBundle) securityModuleWiring {
	securityVulnerabilityStore := securitywiring.NewSecurityVulnerabilityStoreAdapter(repos.vulnerabilityRepo)
	securityTargetLookup := securitywiring.NewSecurityTargetLookupAdapter(repos.targetRepo)
	vulnerabilitySvc := securityservice.NewVulnerabilityFacade(securityVulnerabilityStore, securityTargetLookup)

	return securityModuleWiring{
		vulnerabilityHandler: securityhandler.NewVulnerabilityHandler(vulnerabilitySvc),
		vulnerabilitySvc:     vulnerabilitySvc,
	}
}

func wireScanModule(repos *repositoryBundle, infra *infra, notifier agentservice.AgentMessagePublisher) scanModuleWiring {
	scanQueryStore := scanwiring.NewScanQueryStoreAdapter(repos.scanRepo)
	scanCommandStore := scanwiring.NewScanCommandStoreAdapter(repos.scanRepo)
	scanDomainRepository := scanwiring.NewScanDomainRepositoryAdapter(repos.scanRepo)
	scanTaskStore := scanwiring.NewScanTaskStoreAdapter(repos.scanTaskRepo)
	scanTaskRuntimeStore := scanwiring.NewScanTaskRuntimeStoreAdapter(repos.scanRepo)
	scanTaskRuntimeAgentStore := scanwiring.NewScanTaskRuntimeAgentStoreAdapter(repos.agentRepo)
	scanLogQueryStore := scanlogwiring.NewScanLogQueryStoreAdapter(repos.scanLogRepo)
	scanLogCommandStore := scanlogwiring.NewScanLogCommandStoreAdapter(repos.scanLogRepo)
	scanTaskCanceller := scanwiring.NewScanTaskCancellerAdapter(repos.scanTaskRepo)
	scanTargetLookup := scanwiring.NewScanTargetLookupAdapter(repos.targetRepo)
	scanLogLookup := scanlogwiring.NewScanLogScanLookupAdapter(repos.scanRepo)

	scanSvc := scanwiring.NewScanApplicationService(
		scanQueryStore,
		scanCommandStore,
		scanDomainRepository,
		scanTaskCanceller,
		notifier,
		scanTargetLookup,
	)
	scanTaskSvc := scanwiring.NewScanTaskApplicationService(scanTaskStore, scanTaskRuntimeStore, scanTaskRuntimeAgentStore)
	scanLogSvc := scanlogwiring.NewScanLogApplicationService(scanLogQueryStore, scanLogCommandStore, scanLogLookup)

	return scanModuleWiring{
		scanHandler:     scanhandler.NewScanHandler(scanSvc),
		scanLogHandler:  scanhandler.NewScanLogHandler(scanLogSvc),
		scanTaskRuntime: scanTaskSvc,
	}
}

func wireWorkerModule(repos *repositoryBundle) workerModuleHandlers {
	workerScanGuard := workerwiring.NewWorkerProviderConfigScanGuardAdapter(repos.scanRepo)
	workerSettingsStore := workerwiring.NewWorkerProviderConfigSettingsStoreAdapter(repos.subfinderProviderSettingsRepo)
	workerSvc := workerwiring.NewWorkerProviderConfigApplicationService(workerScanGuard, workerSettingsStore)
	return workerModuleHandlers{
		workerProviderConfigService: workerSvc,
	}
}

func wireAgentModule(
	repos *repositoryBundle,
	infra *infra,
	cfg *config.Config,
	scanTaskRuntime agentservice.ScanTaskRuntimePort,
	messageBus agentservice.AgentMessagePublisher,
) agentModuleHandlers {
	agentClock := agentinfra.NewSystemClock()
	agentTokenGenerator := agentinfra.NewCryptoTokenGenerator()
	agentSvc := agentservice.NewAgentFacade(repos.agentRepo, repos.registrationTokenRepo, agentClock, agentTokenGenerator)
	agentRuntimeSvc := agentservice.NewAgentRuntimeService(
		repos.agentRepo,
		infra.heartbeatCache,
		messageBus,
		agentClock,
		infra.serverVersion,
		infra.agentImageRef,
	)
	agentTaskSvc := agentservice.NewAgentTaskService(scanTaskRuntime)
	lokiLogQuerySvc := agentservice.NewLokiLogQueryService(infra.lokiClient, cfg.JWT.Secret)

	return agentModuleHandlers{
		agentHandler: agenthandler.NewAgentHandler(
			agentSvc,
			agentRuntimeSvc,
			infra.serverVersion,
			cfg.PublicURL,
			fmt.Sprintf("http://server:%d", cfg.Server.GRPCPort),
			infra.agentImageRef,
			infra.workerImageRef,
			infra.sharedDataVolumeBind,
			infra.heartbeatCache,
		),
		agentLogHandler: agenthandler.NewAgentLogHandler(repos.agentRepo, lokiLogQuerySvc),

		agentRuntimeService: agentRuntimeSvc,
		agentTaskService:    agentTaskSvc,
	}
}

func wireSnapshotModule(
	repos *repositoryBundle,
	asset assetModuleWiring,
	security securityModuleWiring,
) snapshotModuleHandlers {
	snapshotScanLookup := snapshotwiring.NewSnapshotScanRefLookupAdapter(repos.scanRepo)

	websiteSnapshotQueryStore := snapshotwiring.NewSnapshotWebsiteQueryStoreAdapter(repos.websiteSnapshotRepo)
	subdomainSnapshotQueryStore := snapshotwiring.NewSnapshotSubdomainQueryStoreAdapter(repos.subdomainSnapshotRepo)
	endpointSnapshotQueryStore := snapshotwiring.NewSnapshotEndpointQueryStoreAdapter(repos.endpointSnapshotRepo)
	directorySnapshotQueryStore := snapshotwiring.NewSnapshotDirectoryQueryStoreAdapter(repos.directorySnapshotRepo)
	hostPortSnapshotQueryStore := snapshotwiring.NewSnapshotHostPortQueryStoreAdapter(repos.hostPortSnapshotRepo)
	screenshotSnapshotQueryStore := snapshotwiring.NewSnapshotScreenshotQueryStoreAdapter(repos.screenshotSnapshotRepo)
	vulnerabilitySnapshotQueryStore := snapshotwiring.NewSnapshotVulnerabilityQueryStoreAdapter(repos.vulnerabilitySnapshotRepo)

	websiteSnapshotCommandStore := snapshotwiring.NewSnapshotWebsiteCommandStoreAdapter(repos.websiteSnapshotRepo)
	subdomainSnapshotCommandStore := snapshotwiring.NewSnapshotSubdomainCommandStoreAdapter(repos.subdomainSnapshotRepo)
	endpointSnapshotCommandStore := snapshotwiring.NewSnapshotEndpointCommandStoreAdapter(repos.endpointSnapshotRepo)
	directorySnapshotCommandStore := snapshotwiring.NewSnapshotDirectoryCommandStoreAdapter(repos.directorySnapshotRepo)
	hostPortSnapshotCommandStore := snapshotwiring.NewSnapshotHostPortCommandStoreAdapter(repos.hostPortSnapshotRepo)
	screenshotSnapshotCommandStore := snapshotwiring.NewSnapshotScreenshotCommandStoreAdapter(repos.screenshotSnapshotRepo)
	vulnerabilitySnapshotCommandStore := snapshotwiring.NewSnapshotVulnerabilityCommandStoreAdapter(repos.vulnerabilitySnapshotRepo)

	websiteAssetSync := snapshotwiring.NewSnapshotWebsiteAssetSyncAdapter(asset.websiteSvc)
	subdomainAssetSync := snapshotwiring.NewSnapshotSubdomainAssetSyncAdapter(asset.subdomainSvc)
	endpointAssetSync := snapshotwiring.NewSnapshotEndpointAssetSyncAdapter(asset.endpointSvc)
	directoryAssetSync := snapshotwiring.NewSnapshotDirectoryAssetSyncAdapter(asset.directorySvc)
	hostPortAssetSync := snapshotwiring.NewSnapshotHostPortAssetSyncAdapter(asset.hostPortSvc)
	screenshotAssetSync := snapshotwiring.NewSnapshotScreenshotAssetSyncAdapter(asset.screenshotSvc)
	vulnerabilityAssetSync := snapshotwiring.NewSnapshotVulnerabilityAssetSyncAdapter(security.vulnerabilitySvc)
	vulnerabilityRawOutputCodec := snapshotwiring.NewSnapshotVulnerabilityRawOutputCodec()

	websiteSnapshotSvc := snapshotwiring.NewSnapshotWebsiteApplicationService(websiteSnapshotQueryStore, websiteSnapshotCommandStore, snapshotScanLookup, websiteAssetSync)
	subdomainSnapshotSvc := snapshotwiring.NewSnapshotSubdomainApplicationService(subdomainSnapshotQueryStore, subdomainSnapshotCommandStore, snapshotScanLookup, subdomainAssetSync)
	endpointSnapshotSvc := snapshotwiring.NewSnapshotEndpointApplicationService(endpointSnapshotQueryStore, endpointSnapshotCommandStore, snapshotScanLookup, endpointAssetSync)
	directorySnapshotSvc := snapshotwiring.NewSnapshotDirectoryApplicationService(directorySnapshotQueryStore, directorySnapshotCommandStore, snapshotScanLookup, directoryAssetSync)
	hostPortSnapshotSvc := snapshotwiring.NewSnapshotHostPortApplicationService(hostPortSnapshotQueryStore, hostPortSnapshotCommandStore, snapshotScanLookup, hostPortAssetSync)
	screenshotSnapshotSvc := snapshotwiring.NewSnapshotScreenshotApplicationService(screenshotSnapshotQueryStore, screenshotSnapshotCommandStore, snapshotScanLookup, screenshotAssetSync)
	vulnerabilitySnapshotSvc := snapshotwiring.NewSnapshotVulnerabilityApplicationService(vulnerabilitySnapshotQueryStore, vulnerabilitySnapshotCommandStore, snapshotScanLookup, vulnerabilityAssetSync, vulnerabilityRawOutputCodec)

	return snapshotModuleHandlers{
		websiteSnapshotHandler:       snapshothandler.NewWebsiteSnapshotHandler(websiteSnapshotSvc),
		subdomainSnapshotHandler:     snapshothandler.NewSubdomainSnapshotHandler(subdomainSnapshotSvc),
		endpointSnapshotHandler:      snapshothandler.NewEndpointSnapshotHandler(endpointSnapshotSvc),
		directorySnapshotHandler:     snapshothandler.NewDirectorySnapshotHandler(directorySnapshotSvc),
		hostPortSnapshotHandler:      snapshothandler.NewHostPortSnapshotHandler(hostPortSnapshotSvc),
		screenshotSnapshotHandler:    snapshothandler.NewScreenshotSnapshotHandler(screenshotSnapshotSvc),
		vulnerabilitySnapshotHandler: snapshothandler.NewVulnerabilitySnapshotHandler(vulnerabilitySnapshotSvc),
		subdomainSnapshotService:     subdomainSnapshotSvc,
		websiteSnapshotService:       websiteSnapshotSvc,
		endpointSnapshotService:      endpointSnapshotSvc,
		hostPortSnapshotService:      hostPortSnapshotSvc,
	}
}
