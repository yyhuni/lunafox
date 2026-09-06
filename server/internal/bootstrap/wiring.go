package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/yyhuni/lunafox/contracts/sharedstorage"
	"github.com/yyhuni/lunafox/server/internal/auth"
	agentwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/agent"
	assetwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/asset"
	blacklistwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/blacklist"
	catalogwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/catalog"
	identitywiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/identity"
	loginvisualwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/loginvisual"
	nucleipocwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/nucleipoc"
	// Nuclei POC source sync is intentionally wired as an isolated catalog
	// module; it does not enter Agent, Worker, or Scan execution paths.
	notificationwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/notification"
	resultingestwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/resultingest"
	scanwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/scan"
	securitywiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/security"
	snapshotwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/snapshot"
	targetcleanupwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/targetcleanup"
	taskprogresslogwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/taskprogresslog"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/geolocation"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	"github.com/yyhuni/lunafox/server/internal/job"
	mcpadapters "github.com/yyhuni/lunafox/server/internal/mcp/adapters"
	mcptools "github.com/yyhuni/lunafox/server/internal/mcp/tools"
	mcptransport "github.com/yyhuni/lunafox/server/internal/mcp/transport"
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
	searchhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/search"
	subdomainhandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/subdomain"
	websitehandler "github.com/yyhuni/lunafox/server/internal/modules/asset/handler/website"
	assetrepo "github.com/yyhuni/lunafox/server/internal/modules/asset/repository"
	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	blacklisthandler "github.com/yyhuni/lunafox/server/internal/modules/blacklist/handler"
	blacklistrepo "github.com/yyhuni/lunafox/server/internal/modules/blacklist/repository"
	catalogservice "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	cataloghandler "github.com/yyhuni/lunafox/server/internal/modules/catalog/handler"
	catalogrepo "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository"
	fingerprintapp "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/application"
	fingerprinthandler "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/handler"
	fingerprintrepo "github.com/yyhuni/lunafox/server/internal/modules/fingerprint/repository"
	identityservice "github.com/yyhuni/lunafox/server/internal/modules/identity/application"
	identityhandler "github.com/yyhuni/lunafox/server/internal/modules/identity/handler"
	identityinfra "github.com/yyhuni/lunafox/server/internal/modules/identity/infrastructure"
	identityrepo "github.com/yyhuni/lunafox/server/internal/modules/identity/repository"
	loginvisualhandler "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/handler"
	notificationdomain "github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
	notificationhandler "github.com/yyhuni/lunafox/server/internal/modules/notification/handler"
	notificationrepo "github.com/yyhuni/lunafox/server/internal/modules/notification/repository"
	nucleipocapp "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/application"
	nucleipocHandler "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/handler"
	nucleipocRepo "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scanhandler "github.com/yyhuni/lunafox/server/internal/modules/scan/handler"
	scaninfra "github.com/yyhuni/lunafox/server/internal/modules/scan/infrastructure"
	scanrepo "github.com/yyhuni/lunafox/server/internal/modules/scan/repository"
	scheduledscanapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	scheduledscanhandler "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/handler"
	scheduledscanrepo "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/repository"
	securityservice "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	securityhandler "github.com/yyhuni/lunafox/server/internal/modules/security/handler"
	securityrepo "github.com/yyhuni/lunafox/server/internal/modules/security/repository"
	snapshotservice "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
	snapshothandler "github.com/yyhuni/lunafox/server/internal/modules/snapshot/handler"
	snapshotrepo "github.com/yyhuni/lunafox/server/internal/modules/snapshot/repository"
	systemservice "github.com/yyhuni/lunafox/server/internal/modules/system/application"
	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
	systemhandler "github.com/yyhuni/lunafox/server/internal/modules/system/handler"
	systeminfra "github.com/yyhuni/lunafox/server/internal/modules/system/infrastructure"
	systemrepo "github.com/yyhuni/lunafox/server/internal/modules/system/repository"
	targetcleanupapp "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/application"
	targetcleanupinfra "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/infrastructure"
	targetcleanuprepo "github.com/yyhuni/lunafox/server/internal/modules/targetcleanup/repository"
	"gorm.io/gorm"
)

type deps struct {
	tokenVersionReader             auth.TokenVersionReader
	mcpKeyService                  *identityservice.MCPKeyLifecycleService
	mcpHandler                     *mcptransport.Handler
	healthHandler                  *assethandler.HealthHandler
	authHandler                    *identityhandler.AuthHandler
	userHandler                    *identityhandler.UserHandler
	orgHandler                     *identityhandler.OrganizationHandler
	targetHandler                  *cataloghandler.TargetHandler
	engineCatalogHandler           *cataloghandler.EngineCatalogHandler
	scanWorkflowCatalogHandler     *cataloghandler.ScanWorkflowManagementHandler
	wordlistHandler                *cataloghandler.WordlistHandler
	subfinderAPIKeySettingsHandler *cataloghandler.SubfinderAPIKeySettingsHandler
	blacklistPolicyHandler         *blacklisthandler.BlacklistPolicyHandler
	websiteHandler                 *websitehandler.WebsiteHandler
	subdomainHandler               *subdomainhandler.SubdomainHandler
	endpointHandler                *endpointhandler.EndpointHandler
	directoryHandler               *directoryhandler.DirectoryHandler
	hostPortHandler                *hostporthandler.HostPortHandler
	screenshotHandler              *screenshothandler.ScreenshotHandler
	globalAssetSearchHandler       *searchhandler.GlobalAssetSearchHandler
	assetStatisticsHandler         *assethandler.AssetStatisticsHandler
	vulnerabilityHandler           *securityhandler.VulnerabilityHandler
	scanHandler                    *scanhandler.ScanHandler
	scheduledScanHandler           *scheduledscanhandler.ScheduledScanHandler
	taskProgressLogHandler         *scanhandler.TaskProgressLogHandler
	fingerprintHandler             *fingerprinthandler.FingerprintHandler
	fingerprintArtifacts           *fingerprintapp.FingerprintArtifactService
	nucleiPocRepo                  *nucleipocRepo.NucleiPOCRepository

	agentHandler                   *agenthandler.AgentHandler
	agentLogHandler                *agenthandler.AgentLogHandler
	agentClusterSummaryHandler     *agenthandler.AgentClusterSummaryHandler
	agentLocationMapHandler        *agenthandler.AgentLocationMapHandler
	serverLogHandler               *systemhandler.ServerLogHandler
	runtimeMetricsHandler          *systemhandler.RuntimeMetricsHandler
	runtimeMetricsJob              runtimeMetricsJob
	scanHistoryRetentionJob        runtimeMetricsJob
	registrationTokenCleanupJob    runtimeMetricsJob
	scheduledScanController        managedBackgroundJob
	occurrenceRetentionJob         managedBackgroundJob
	targetCleanupRunner            managedBackgroundJob
	notificationOutboxWorker       managedBackgroundJob
	notificationDeliveryWorker     managedBackgroundJob
	notificationRetentionJob       managedBackgroundJob
	notificationInboxHandler       *notificationhandler.NotificationInboxHandler
	notificationDestinationHandler *notificationhandler.NotificationDestinationHandler
	notificationSSEHandler         *notificationhandler.NotificationSSEHandler
	loginVisualHandler             *loginvisualhandler.LoginVisualHandler
	nucleiPocHandler               *nucleipocHandler.NucleiPOCHandler
	nucleiPocRunner                *nucleipocapp.SyncRunner
	nucleiPocRetentionJob          *nucleipocapp.RetentionJob

	websiteSnapshotHandler       *snapshothandler.WebsiteSnapshotHandler
	subdomainSnapshotHandler     *snapshothandler.SubdomainSnapshotHandler
	endpointSnapshotHandler      *snapshothandler.EndpointSnapshotHandler
	directorySnapshotHandler     *snapshothandler.DirectorySnapshotHandler
	hostPortSnapshotHandler      *snapshothandler.HostPortSnapshotHandler
	screenshotSnapshotHandler    *snapshothandler.ScreenshotSnapshotHandler
	vulnerabilitySnapshotHandler *snapshothandler.VulnerabilitySnapshotHandler

	agentRepo    agentdomain.AgentRepository
	scanRepo     *scanrepo.ScanRepository
	scanTaskRepo scanrepo.ScanTaskRepository
	scanTaskSvc  *scanapp.ScanTaskFacade

	agentControlService     *agentservice.AgentControlLifecycleService
	agentTaskService        *agentservice.AgentTaskService
	agentLocationObserver   *agentservice.AgentLocationObserver
	serverLocationReader    *systemservice.ServerLocationReader
	serverLocationScheduler *systemservice.ServerLocationScheduler
	geolocationCoordinator  *geolocation.Coordinator

	executionProviderConfigSource *catalogservice.ExecutionProviderConfigSource
	executionWordlistSource       *catalogservice.ExecutionWordlistSource
	subdomainSnapshotRepo         *snapshotrepo.SubdomainSnapshotRepository
	hostPortSnapshotRepo          *snapshotrepo.HostPortSnapshotRepository
	websiteSnapshotRepo           *snapshotrepo.WebsiteSnapshotRepository
	endpointSnapshotRepo          *snapshotrepo.EndpointSnapshotRepository
	subdomainRepo                 *assetrepo.SubdomainRepository
	hostPortRepo                  *assetrepo.HostPortRepository
	websiteRepo                   *assetrepo.WebsiteRepository
	endpointRepo                  *assetrepo.EndpointRepository
	subdomainSnapshotService      *snapshotservice.SubdomainSnapshotFacade
	websiteSnapshotService        *snapshotservice.WebsiteSnapshotFacade
	endpointSnapshotService       *snapshotservice.EndpointSnapshotFacade
	hostPortSnapshotService       *snapshotservice.HostPortSnapshotFacade
	screenshotSnapshotService     *snapshotservice.ScreenshotSnapshotFacade
	resultIngestService           *resultingestapp.ResultIngestFacade
	runtimeStreamRegistry         *agentcontrol.AgentStreamRegistry
	taskProgressLogService        scanapp.TaskProgressLogApplicationService
}

type repositoryBundle struct {
	userRepo                      *identityrepo.UserRepository
	mcpKeyRepo                    *identityrepo.MCPKeyRepository
	orgRepo                       *identityrepo.OrganizationRepository
	targetRepo                    *catalogrepo.TargetRepository
	blacklistPolicyRepo           *blacklistrepo.PolicyRepository
	engineRepo                    *catalogrepo.EngineRepository
	wordlistRepo                  *catalogrepo.WordlistRepository
	scanWorkflowRepo              *catalogrepo.ScanWorkflowRepository
	websiteRepo                   *assetrepo.WebsiteRepository
	subdomainRepo                 *assetrepo.SubdomainRepository
	endpointRepo                  *assetrepo.EndpointRepository
	directoryRepo                 *assetrepo.DirectoryRepository
	hostPortRepo                  *assetrepo.HostPortRepository
	assetStatisticsRepo           *assetrepo.AssetStatisticsRepository
	screenshotRepo                *assetrepo.ScreenshotRepository
	vulnerabilityRepo             securityrepo.VulnerabilityRepository
	scanRepo                      *scanrepo.ScanRepository
	scheduledScanRepo             *scheduledscanrepo.ScheduledScanRepository
	targetCleanupRepo             *targetcleanuprepo.TargetCleanupRepository
	taskProgressLogRepo           *scanrepo.TaskProgressLogRepository
	subfinderProviderSettingsRepo *catalogrepo.SubfinderProviderSettingsRepository
	websiteSnapshotRepo           *snapshotrepo.WebsiteSnapshotRepository
	subdomainSnapshotRepo         *snapshotrepo.SubdomainSnapshotRepository
	endpointSnapshotRepo          *snapshotrepo.EndpointSnapshotRepository
	directorySnapshotRepo         *snapshotrepo.DirectorySnapshotRepository
	hostPortSnapshotRepo          *snapshotrepo.HostPortSnapshotRepository
	screenshotSnapshotRepo        *snapshotrepo.ScreenshotSnapshotRepository
	vulnerabilitySnapshotRepo     *snapshotrepo.VulnerabilitySnapshotRepository
	agentRepo                     agentdomain.AgentOperationalRepository
	agentLocationRepo             agentdomain.AgentLocationRepository
	registrationTokenRepo         agentdomain.RegistrationTokenRepository
	serverLocationRepo            systemdomain.ServerLocationRepository
	scanTaskRepo                  scanrepo.ScanTaskRepository
	fingerprintRepo               *fingerprintrepo.FingerprintRepository
	notificationProducer          *notificationrepo.ProducerWriter
	nucleiPocRepo                 *nucleipocRepo.NucleiPOCRepository
}

type identityModuleHandlers struct {
	authHandler        *identityhandler.AuthHandler
	userHandler        *identityhandler.UserHandler
	orgHandler         *identityhandler.OrganizationHandler
	organizationSvc    *identityservice.OrganizationFacade
	tokenVersionReader auth.TokenVersionReader
	mcpKeyService      *identityservice.MCPKeyLifecycleService
}

type catalogModuleHandlers struct {
	targetHandler                  *cataloghandler.TargetHandler
	engineCatalogHandler           *cataloghandler.EngineCatalogHandler
	scanWorkflowCatalogHandler     *cataloghandler.ScanWorkflowManagementHandler
	wordlistHandler                *cataloghandler.WordlistHandler
	subfinderAPIKeySettingsHandler *cataloghandler.SubfinderAPIKeySettingsHandler

	wordlistService               *catalogservice.WordlistFacade
	targetSvc                     *catalogservice.TargetFacade
	scanWorkflowService           *catalogservice.ScanWorkflowManagementService
	scanWorkflowProfileService    *catalogservice.ScanWorkflowProfileService
	engineCatalogService          *catalogservice.EngineCatalogFacade
	executionProviderConfigSource *catalogservice.ExecutionProviderConfigSource
	executionWordlistSource       *catalogservice.ExecutionWordlistSource
}

type blacklistModuleHandlers struct {
	blacklistPolicyHandler *blacklisthandler.BlacklistPolicyHandler
	blacklistPolicyService blacklistapp.BlacklistPolicyApplicationService
}

type assetModuleWiring struct {
	websiteHandler           *websitehandler.WebsiteHandler
	subdomainHandler         *subdomainhandler.SubdomainHandler
	endpointHandler          *endpointhandler.EndpointHandler
	directoryHandler         *directoryhandler.DirectoryHandler
	hostPortHandler          *hostporthandler.HostPortHandler
	screenshotHandler        *screenshothandler.ScreenshotHandler
	globalAssetSearchHandler *searchhandler.GlobalAssetSearchHandler
	assetStatisticsHandler   *assethandler.AssetStatisticsHandler

	websiteSvc    *assetservice.WebsiteFacade
	subdomainSvc  *assetservice.SubdomainFacade
	endpointSvc   *assetservice.EndpointFacade
	directorySvc  *assetservice.DirectoryFacade
	hostPortSvc   *assetservice.HostPortFacade
	screenshotSvc *assetservice.ScreenshotFacade
}

type fingerprintModuleWiring struct {
	handler   *fingerprinthandler.FingerprintHandler
	artifacts *fingerprintapp.FingerprintArtifactService
}

type securityModuleWiring struct {
	vulnerabilityHandler *securityhandler.VulnerabilityHandler
	vulnerabilitySvc     *securityservice.VulnerabilityFacade
}

type scanModuleWiring struct {
	scanHandler             *scanhandler.ScanHandler
	scheduledScanHandler    *scheduledscanhandler.ScheduledScanHandler
	taskProgressLogHandler  *scanhandler.TaskProgressLogHandler
	scanTaskBridge          agentservice.ScanTaskBridgePort
	scanTaskSvc             *scanapp.ScanTaskFacade
	taskProgressLogService  scanapp.TaskProgressLogApplicationService
	scanSvc                 *scanapp.ScanFacade
	scheduledScanController managedBackgroundJob
	occurrenceRetentionJob  managedBackgroundJob
}

type agentModuleHandlers struct {
	agentHandler               *agenthandler.AgentHandler
	agentLogHandler            *agenthandler.AgentLogHandler
	agentClusterSummaryHandler *agenthandler.AgentClusterSummaryHandler
	agentLocationMapHandler    *agenthandler.AgentLocationMapHandler

	agentControlService         *agentservice.AgentControlLifecycleService
	agentTaskService            *agentservice.AgentTaskService
	agentService                *agentservice.AgentFacade
	lokiLogQueryService         *agentservice.LokiLogQueryService
	agentLocationObserver       *agentservice.AgentLocationObserver
	registrationTokenCleanupJob runtimeMetricsJob
}

type systemModuleHandlers struct {
	serverLogHandler        *systemhandler.ServerLogHandler
	serverLogService        *systemservice.ServerLogService
	runtimeMetricsHandler   *systemhandler.RuntimeMetricsHandler
	runtimeMetricsJob       runtimeMetricsJob
	serverLocationReader    *systemservice.ServerLocationReader
	serverLocationScheduler *systemservice.ServerLocationScheduler
}

type runtimeMetricsJob interface {
	Start(ctx context.Context)
}

type managedBackgroundJob interface {
	Start(ctx context.Context)
	Done() <-chan struct{}
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
	directorySnapshotService     *snapshotservice.DirectorySnapshotFacade
	hostPortSnapshotService      *snapshotservice.HostPortSnapshotFacade
	screenshotSnapshotService    *snapshotservice.ScreenshotSnapshotFacade
	vulnerabilitySnapshotService *snapshotservice.VulnerabilitySnapshotFacade
}

func buildDependencies(infra *infra, cfg *config.Config) (*deps, error) {
	repos := newRepositoryBundle(infra, cfg)
	notificationWorkerOwner, err := notificationwiring.NewNotificationWorkerOwner()
	if err != nil {
		return nil, fmt.Errorf("create notification worker owner: %w", err)
	}
	notification, err := notificationwiring.NewNotificationModule(infra.db, notificationWorkerOwner)
	if err != nil {
		return nil, fmt.Errorf("wire notification module: %w", err)
	}
	loginVisual, err := loginvisualwiring.NewLoginVisualModule(infra.db, cfg.Storage.LoginVisualsBasePath)
	if err != nil {
		return nil, fmt.Errorf("wire login visual module: %w", err)
	}
	streamRegistry := agentcontrol.NewAgentStreamRegistry()
	grpcPublisher := agentcontrol.NewAgentControlEventPublisher(streamRegistry)

	identity := wireIdentityModule(repos, infra)
	catalog := wireCatalogModule(repos, cfg, infra)
	blacklist, err := wireBlacklistModule(repos)
	if err != nil {
		return nil, fmt.Errorf("wire blacklist module: %w", err)
	}
	asset := wireAssetModule(repos)
	fingerprint, err := wireFingerprintModule(repos, cfg)
	if err != nil {
		return nil, fmt.Errorf("wire fingerprint module: %w", err)
	}
	security := wireSecurityModule(repos)
	scan, err := wireScanModule(repos, infra, cfg, grpcPublisher, catalog.wordlistService, blacklist.blacklistPolicyService)
	if err != nil {
		return nil, fmt.Errorf("wire scan module: %w", err)
	}
	targetCleanupService, err := targetcleanupwiring.NewTargetcleanupApplicationService(
		repos.targetCleanupRepo,
		targetcleanupwiring.NewTargetcleanupScheduleCleanerAdapter(repos.scheduledScanRepo),
		targetcleanupwiring.NewTargetcleanupScanCancellerAdapter(repos.scanRepo),
		targetcleanupwiring.NewTargetcleanupTaskCancelPublisherAdapter(grpcPublisher),
	)
	if err != nil {
		return nil, fmt.Errorf("wire Target cleanup module: %w", err)
	}
	targetCleanupRunner, err := targetcleanupapp.NewTargetCleanupRunner(
		repos.targetCleanupRepo,
		targetCleanupService,
		targetcleanupapp.TargetCleanupRunOptions{
			AssetBatchSize:  cfg.TargetCleanup.BatchSize,
			MaxAssetBatches: cfg.TargetCleanup.MaxBatchesPerRun,
			MaxRunDuration:  cfg.TargetCleanup.MaxRunDuration,
		},
		targetcleanupinfra.NewTargetCleanupRunObserver(),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize Target cleanup Runner: %w", err)
	}
	locationCoordinator, err := geolocation.NewCoordinator(geolocation.NewFreeIPAPIClient(nil))
	if err != nil {
		return nil, fmt.Errorf("initialize geolocation coordinator: %w", err)
	}
	system, err := wireSystemModule(repos, infra, cfg, locationCoordinator)
	if err != nil {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = locationCoordinator.Shutdown(shutdownContext)
		return nil, fmt.Errorf("wire system module: %w", err)
	}
	serverLocationReader, err := agentwiring.NewAgentServerLocationReaderAdapter(system.serverLocationReader)
	if err != nil {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = locationCoordinator.Shutdown(shutdownContext)
		return nil, fmt.Errorf("wire Agent Server location reader: %w", err)
	}
	agent, err := wireAgentModule(repos, infra, cfg, scan.scanTaskBridge, grpcPublisher, locationCoordinator, serverLocationReader)
	if err != nil {
		shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = locationCoordinator.Shutdown(shutdownContext)
		return nil, fmt.Errorf("wire Agent module: %w", err)
	}
	snapshot := wireSnapshotModule(infra.db, repos, asset, security)
	scanHistoryRetentionJob := job.NewScanHistoryRetentionJob(
		snapshotrepo.NewScanHistoryPartitionLifecycle(infra.db),
		snapshotrepo.ScanHistoryRetentionMode(cfg.ScanHistoryRetention.Mode),
		cfg.ScanHistoryRetention.Interval,
		cfg.ScanHistoryRetention.Retention,
		snapshotrepo.ScanHistoryRetentionRunOptions{
			TaskDeleteBatchSize:          cfg.ScanHistoryRetention.TaskDeleteBatchSize,
			MaxTaskDeleteBatchesPerRange: cfg.ScanHistoryRetention.MaxTaskDeleteBatchesPerRange,
			MaxRangesPerRun:              cfg.ScanHistoryRetention.MaxRangesPerRun,
			MaxRunDuration:               cfg.ScanHistoryRetention.MaxRunDuration,
		},
	)
	resultIngestSummary := resultingestwiring.NewResultingestScanResultSummaryUpdaterAdapter(repos.scanRepo)
	resultIngestCoordinator := resultingestwiring.NewResultIngestMaterializationCoordinator(infra.db)
	resultIngest := resultingestapp.NewResultIngestFacade(resultingestapp.ResultIngestFacadeDependencies{Subdomains: snapshot.subdomainSnapshotService, ScanSummary: resultIngestSummary, HostPorts: snapshot.hostPortSnapshotService, Websites: snapshot.websiteSnapshotService, WebsiteTechnologies: asset.websiteSvc, Endpoints: snapshot.endpointSnapshotService, Directories: snapshot.directorySnapshotService, Screenshots: snapshot.screenshotSnapshotService, Vulnerabilities: snapshot.vulnerabilitySnapshotService, Materialization: resultIngestCoordinator})
	mcpRegistry := mcptools.NewRegistry(mcpadapters.NewReaders(mcpadapters.Dependencies{
		Targets:         catalog.targetSvc,
		Organizations:   identity.organizationSvc,
		Scans:           scan.scanSvc,
		Websites:        asset.websiteSvc,
		Subdomains:      asset.subdomainSvc,
		Endpoints:       asset.endpointSvc,
		Directories:     asset.directorySvc,
		HostPorts:       asset.hostPortSvc,
		Vulnerabilities: security.vulnerabilitySvc,
		Screenshots:     asset.screenshotSvc,
		Workflows:       catalog.scanWorkflowService,
		Profiles:        catalog.scanWorkflowProfileService,
		Engines:         catalog.engineCatalogService,
		Wordlists:       catalog.wordlistService,
		ServerLogs:      system.serverLogService,
		AgentLogs:       agent.lokiLogQueryService,
		Agents:          agent.agentService,
	}))
	mcpHandler := mcptransport.NewIdentityHandler(identity.mcpKeyService, mcpRegistry)

	nucleiPocResolver := nucleipocapp.NewPublicSourceResolver()
	nucleiPocWorkspace, err := nucleipocapp.NewLocalWorkspace(cfg.Storage.NucleiPocWorkspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("wire nuclei POC workspace: %w", err)
	}
	nucleiPocRunner := nucleipocapp.NewSyncRunner(repos.nucleiPocRepo, nucleiPocResolver, nucleiPocWorkspace, nucleipocapp.NewProcessGitRunner())
	nucleiPocService, err := nucleipocapp.NewPOCService(repos.nucleiPocRepo, nucleiPocResolver, nucleiPocRunner)
	if err != nil {
		return nil, fmt.Errorf("wire nuclei POC service: %w", err)
	}
	nucleiPocRetention := nucleipocapp.NewRetentionJob(repos.nucleiPocRepo, nucleiPocWorkspace, nucleipocapp.RetentionInterval)
	// Recover interrupted tasks before accepting a new request. This makes the
	// singleton active-task constraint deterministic across process restarts.
	if _, err := repos.nucleiPocRepo.RecoverInterruptedTasks(context.Background(), time.Now().UTC()); err != nil {
		return nil, fmt.Errorf("recover nuclei POC sync tasks: %w", err)
	}

	return &deps{
		tokenVersionReader:             identity.tokenVersionReader,
		mcpKeyService:                  identity.mcpKeyService,
		mcpHandler:                     mcpHandler,
		healthHandler:                  assethandler.NewHealthHandler(infra.db, infra.redisClient),
		authHandler:                    identity.authHandler,
		userHandler:                    identity.userHandler,
		orgHandler:                     identity.orgHandler,
		targetHandler:                  catalog.targetHandler,
		engineCatalogHandler:           catalog.engineCatalogHandler,
		scanWorkflowCatalogHandler:     catalog.scanWorkflowCatalogHandler,
		wordlistHandler:                catalog.wordlistHandler,
		subfinderAPIKeySettingsHandler: catalog.subfinderAPIKeySettingsHandler,
		blacklistPolicyHandler:         blacklist.blacklistPolicyHandler,
		websiteHandler:                 asset.websiteHandler,
		subdomainHandler:               asset.subdomainHandler,
		endpointHandler:                asset.endpointHandler,
		directoryHandler:               asset.directoryHandler,
		hostPortHandler:                asset.hostPortHandler,
		screenshotHandler:              asset.screenshotHandler,
		globalAssetSearchHandler:       asset.globalAssetSearchHandler,
		assetStatisticsHandler:         asset.assetStatisticsHandler,
		vulnerabilityHandler:           security.vulnerabilityHandler,
		scanHandler:                    scan.scanHandler,
		scheduledScanHandler:           scan.scheduledScanHandler,
		taskProgressLogHandler:         scan.taskProgressLogHandler,
		fingerprintHandler:             fingerprint.handler,
		fingerprintArtifacts:           fingerprint.artifacts,
		nucleiPocRepo:                  repos.nucleiPocRepo,

		agentHandler:                   agent.agentHandler,
		agentLogHandler:                agent.agentLogHandler,
		agentClusterSummaryHandler:     agent.agentClusterSummaryHandler,
		agentLocationMapHandler:        agent.agentLocationMapHandler,
		serverLogHandler:               system.serverLogHandler,
		runtimeMetricsHandler:          system.runtimeMetricsHandler,
		runtimeMetricsJob:              system.runtimeMetricsJob,
		scanHistoryRetentionJob:        scanHistoryRetentionJob,
		scheduledScanController:        scan.scheduledScanController,
		occurrenceRetentionJob:         scan.occurrenceRetentionJob,
		targetCleanupRunner:            targetCleanupRunner,
		registrationTokenCleanupJob:    agent.registrationTokenCleanupJob,
		notificationOutboxWorker:       notification.OutboxWorker,
		notificationDeliveryWorker:     notification.DeliveryWorker,
		notificationRetentionJob:       notification.RetentionJob,
		notificationInboxHandler:       notification.InboxHandler,
		notificationDestinationHandler: notification.DestinationHandler,
		notificationSSEHandler:         notification.SSEHandler,
		loginVisualHandler:             loginVisual.Handler,
		nucleiPocHandler:               nucleipocHandler.NewNucleiPOCHandler(nucleiPocService),
		nucleiPocRunner:                nucleiPocRunner,
		nucleiPocRetentionJob:          nucleiPocRetention,

		websiteSnapshotHandler:       snapshot.websiteSnapshotHandler,
		subdomainSnapshotHandler:     snapshot.subdomainSnapshotHandler,
		endpointSnapshotHandler:      snapshot.endpointSnapshotHandler,
		directorySnapshotHandler:     snapshot.directorySnapshotHandler,
		hostPortSnapshotHandler:      snapshot.hostPortSnapshotHandler,
		screenshotSnapshotHandler:    snapshot.screenshotSnapshotHandler,
		vulnerabilitySnapshotHandler: snapshot.vulnerabilitySnapshotHandler,

		agentRepo:    repos.agentRepo,
		scanRepo:     repos.scanRepo,
		scanTaskRepo: repos.scanTaskRepo,
		scanTaskSvc:  scan.scanTaskSvc,

		agentControlService:           agent.agentControlService,
		agentTaskService:              agent.agentTaskService,
		agentLocationObserver:         agent.agentLocationObserver,
		serverLocationReader:          system.serverLocationReader,
		serverLocationScheduler:       system.serverLocationScheduler,
		geolocationCoordinator:        locationCoordinator,
		executionProviderConfigSource: catalog.executionProviderConfigSource,
		executionWordlistSource:       catalog.executionWordlistSource,
		subdomainSnapshotRepo:         repos.subdomainSnapshotRepo,
		hostPortSnapshotRepo:          repos.hostPortSnapshotRepo,
		websiteSnapshotRepo:           repos.websiteSnapshotRepo,
		endpointSnapshotRepo:          repos.endpointSnapshotRepo,
		subdomainRepo:                 repos.subdomainRepo,
		hostPortRepo:                  repos.hostPortRepo,
		websiteRepo:                   repos.websiteRepo,
		endpointRepo:                  repos.endpointRepo,
		subdomainSnapshotService:      snapshot.subdomainSnapshotService,
		websiteSnapshotService:        snapshot.websiteSnapshotService,
		endpointSnapshotService:       snapshot.endpointSnapshotService,
		hostPortSnapshotService:       snapshot.hostPortSnapshotService,
		screenshotSnapshotService:     snapshot.screenshotSnapshotService,
		resultIngestService:           resultIngest,
		taskProgressLogService:        scan.taskProgressLogService,
		runtimeStreamRegistry:         streamRegistry,
	}, nil
}

func newRepositoryBundle(infra *infra, cfg *config.Config) *repositoryBundle {
	db := infra.db
	notificationOutbox := notificationrepo.NewOutboxRepository(db)
	notificationProducer := notificationrepo.NewProducerWriter(notificationOutbox, notificationdomain.VulnerabilitySeverity(cfg.Notification.VulnerabilityThreshold))
	return &repositoryBundle{
		userRepo:                      identityrepo.NewUserRepository(db),
		mcpKeyRepo:                    identityrepo.NewMCPKeyRepository(db),
		orgRepo:                       identityrepo.NewOrganizationRepository(db),
		targetRepo:                    catalogrepo.NewTargetRepository(db),
		blacklistPolicyRepo:           blacklistrepo.NewPolicyRepository(db),
		engineRepo:                    catalogrepo.NewEngineRepository(db),
		wordlistRepo:                  catalogrepo.NewWordlistRepository(db),
		scanWorkflowRepo:              catalogrepo.NewScanWorkflowRepository(db),
		websiteRepo:                   assetrepo.NewWebsiteRepository(db),
		subdomainRepo:                 assetrepo.NewSubdomainRepository(db),
		endpointRepo:                  assetrepo.NewEndpointRepository(db),
		directoryRepo:                 assetrepo.NewDirectoryRepository(db),
		hostPortRepo:                  assetrepo.NewHostPortRepository(db),
		assetStatisticsRepo:           assetrepo.NewAssetStatisticsRepository(db),
		screenshotRepo:                assetrepo.NewScreenshotRepository(db),
		vulnerabilityRepo:             securityrepo.NewVulnerabilityRepository(db),
		scanRepo:                      scanrepo.NewScanRepository(db, notificationProducer),
		scheduledScanRepo:             scheduledscanrepo.NewScheduledScanRepository(db),
		targetCleanupRepo:             targetcleanuprepo.NewTargetCleanupRepository(db),
		taskProgressLogRepo:           scanrepo.NewTaskProgressLogRepository(db),
		subfinderProviderSettingsRepo: catalogrepo.NewSubfinderProviderSettingsRepository(db),
		websiteSnapshotRepo:           snapshotrepo.NewWebsiteSnapshotRepository(db),
		subdomainSnapshotRepo:         snapshotrepo.NewSubdomainSnapshotRepository(db),
		endpointSnapshotRepo:          snapshotrepo.NewEndpointSnapshotRepository(db),
		directorySnapshotRepo:         snapshotrepo.NewDirectorySnapshotRepository(db),
		hostPortSnapshotRepo:          snapshotrepo.NewHostPortSnapshotRepository(db),
		screenshotSnapshotRepo:        snapshotrepo.NewScreenshotSnapshotRepository(db),
		vulnerabilitySnapshotRepo:     snapshotrepo.NewVulnerabilitySnapshotRepository(db),
		agentRepo:                     agentrepo.NewAgentRepository(db, notificationProducer),
		agentLocationRepo:             agentrepo.NewAgentLocationRepository(db),
		registrationTokenRepo:         agentrepo.NewRegistrationTokenRepository(db),
		serverLocationRepo:            systemrepo.NewServerLocationRepository(db),
		scanTaskRepo:                  scanrepo.NewScanTaskRepository(db),
		fingerprintRepo:               fingerprintrepo.NewFingerprintRepository(db),
		notificationProducer:          notificationProducer,
		nucleiPocRepo:                 nucleipocwiring.NewNucleiPOCRepository(db, notificationProducer),
	}
}

func wireIdentityModule(repos *repositoryBundle, infra *infra) identityModuleHandlers {
	identityUserQueryStore := identitywiring.NewIdentityUserQueryStoreAdapter(repos.userRepo)
	identityUserCommandStore := identitywiring.NewIdentityUserCommandStoreAdapter(repos.userRepo)
	identityOrgQueryStore := identitywiring.NewIdentityOrganizationQueryStoreAdapter(repos.orgRepo)
	identityOrgCommandStore := identitywiring.NewIdentityOrganizationCommandStoreAdapter(repos.orgRepo)
	identityAuthUserStore := identitywiring.NewIdentityAuthUserStoreAdapter(repos.userRepo)
	identityTokenVersionReader := identitywiring.NewIdentityTokenVersionReaderAdapter(repos.userRepo)
	mcpKeyService := identityservice.NewMCPKeyLifecycleService(repos.mcpKeyRepo, identityinfra.NewMCPKeySecretGenerator())

	userQueryService := identityservice.NewUserQueryService(identityUserQueryStore)
	userCommandService := identityservice.NewUserCommandService(identityUserCommandStore, identityservice.NewAuthPasswordHasher())
	orgQueryService := identityservice.NewOrganizationQueryService(identityOrgQueryStore)
	orgCommandService := identityservice.NewOrganizationCommandService(identityOrgCommandStore)
	authCommandService := identityservice.NewAuthCommandService(identityAuthUserStore, identityservice.NewAuthPasswordVerifier(), infra.jwtManager)

	userSvc := identityservice.NewUserFacade(userQueryService, userCommandService)
	orgSvc := identityservice.NewOrganizationFacade(orgQueryService, orgCommandService)
	authSvc := identityservice.NewAuthFacade(authCommandService)

	return identityModuleHandlers{
		authHandler:        identityhandler.NewAuthHandler(authSvc),
		userHandler:        identityhandler.NewUserHandler(userSvc, mcpKeyService),
		orgHandler:         identityhandler.NewOrganizationHandler(orgSvc),
		organizationSvc:    orgSvc,
		tokenVersionReader: identityTokenVersionReader,
		mcpKeyService:      mcpKeyService,
	}
}

func wireCatalogModule(repos *repositoryBundle, cfg *config.Config, infra *infra) catalogModuleHandlers {
	catalogTargetQueryStore := catalogwiring.NewCatalogTargetQueryStoreAdapter(repos.targetRepo)
	catalogTargetCommandStore := catalogwiring.NewCatalogTargetCommandStoreAdapter(repos.targetRepo)
	catalogWordlistQueryStore := catalogwiring.NewCatalogWordlistQueryStoreAdapter(repos.wordlistRepo)
	catalogWordlistCommandStore := catalogwiring.NewCatalogWordlistCommandStoreAdapter(repos.wordlistRepo)
	catalogOrganizationTargetBindingStore := catalogwiring.NewCatalogOrganizationTargetBindingStoreAdapter(repos.orgRepo)
	catalogTransactionCoordinator := catalogwiring.NewCatalogTransactionCoordinator(infra.db)
	engineCatalogQueryStore := catalogwiring.NewCatalogEngineCatalogQueryStoreAdapter(infra.installedEngineQuery)
	subfinderAPIKeySettingsStore := catalogwiring.NewCatalogSubfinderProviderSettingsStoreAdapter(repos.subfinderProviderSettingsRepo)
	executionProviderSettingsStore := catalogwiring.NewCatalogExecutionSubfinderProviderSettingsStoreAdapter(repos.subfinderProviderSettingsRepo)

	targetQueryService := catalogservice.NewTargetQueryService(catalogTargetQueryStore)
	targetCommandService := catalogservice.NewTargetCommandService(catalogTargetCommandStore, catalogOrganizationTargetBindingStore, catalogTransactionCoordinator)
	wordlistFileStore := catalogservice.NewLocalWordlistFileStore()
	wordlistQueryService := catalogservice.NewWordlistQueryService(catalogWordlistQueryStore, wordlistFileStore)
	wordlistCommandService := catalogservice.NewWordlistCommandService(catalogWordlistCommandStore, cfg.Storage.WordlistsBasePath, wordlistFileStore)

	targetSvc := catalogservice.NewTargetFacade(targetQueryService, targetCommandService)
	wordlistSvc := catalogservice.NewWordlistFacade(wordlistQueryService, wordlistCommandService)
	scanWorkflowCatalogSvc, err := catalogwiring.NewScanWorkflowManagementService(repos.scanWorkflowRepo, infra.installedEngineQuery)
	if err != nil {
		panic(fmt.Sprintf("initialize scan workflow management service: %v", err))
	}
	scanWorkflowProfileSvc, err := catalogwiring.NewScanWorkflowProfileService(repos.scanWorkflowRepo, infra.installedEngineQuery)
	if err != nil {
		panic(fmt.Sprintf("initialize scan workflow Profile service: %v", err))
	}
	engineCatalogSvc := catalogservice.NewEngineCatalogFacade(engineCatalogQueryStore)
	subfinderAPIKeySettingsSvc := catalogservice.NewSubfinderAPIKeySettingsService(subfinderAPIKeySettingsStore)
	executionProviderConfigSource := catalogservice.NewExecutionProviderConfigSource(executionProviderSettingsStore)
	executionWordlistSource := catalogservice.NewExecutionWordlistSource(wordlistQueryService)

	return catalogModuleHandlers{
		targetHandler:                  cataloghandler.NewTargetHandler(targetSvc),
		engineCatalogHandler:           cataloghandler.NewEngineCatalogHandler(engineCatalogSvc, infra.operatorEngineInstall),
		scanWorkflowCatalogHandler:     cataloghandler.NewScanWorkflowManagementHandler(scanWorkflowCatalogSvc, scanWorkflowProfileSvc),
		wordlistHandler:                cataloghandler.NewWordlistHandler(wordlistSvc),
		subfinderAPIKeySettingsHandler: cataloghandler.NewSubfinderAPIKeySettingsHandler(subfinderAPIKeySettingsSvc),
		wordlistService:                wordlistSvc,
		targetSvc:                      targetSvc,
		scanWorkflowService:            scanWorkflowCatalogSvc,
		scanWorkflowProfileService:     scanWorkflowProfileSvc,
		engineCatalogService:           engineCatalogSvc,
		executionProviderConfigSource:  executionProviderConfigSource,
		executionWordlistSource:        executionWordlistSource,
	}
}

func wireBlacklistModule(repos *repositoryBundle) (blacklistModuleHandlers, error) {
	policyStore := blacklistwiring.NewBlacklistPolicyStoreAdapter(repos.blacklistPolicyRepo)
	policyService, err := blacklistwiring.NewBlacklistPolicyApplicationService(policyStore)
	if err != nil {
		return blacklistModuleHandlers{}, err
	}
	policyHandler, err := blacklisthandler.NewBlacklistPolicyHandler(policyService)
	if err != nil {
		return blacklistModuleHandlers{}, err
	}
	return blacklistModuleHandlers{blacklistPolicyHandler: policyHandler, blacklistPolicyService: policyService}, nil
}

func wireAssetModule(repos *repositoryBundle) assetModuleWiring {
	assetTargetLookup := assetwiring.NewAssetTargetLookupAdapter(repos.targetRepo)
	assetWebsiteStore := assetwiring.NewAssetWebsiteStoreAdapter(repos.websiteRepo)
	assetSubdomainStore := assetwiring.NewAssetSubdomainStoreAdapter(repos.subdomainRepo)
	assetEndpointStore := assetwiring.NewAssetEndpointStoreAdapter(repos.endpointRepo)
	assetDirectoryStore := assetwiring.NewAssetDirectoryStoreAdapter(repos.directoryRepo)
	assetHostPortStore := assetwiring.NewAssetHostPortStoreAdapter(repos.hostPortRepo)
	assetScreenshotStore := assetwiring.NewAssetScreenshotStoreAdapter(repos.screenshotRepo)
	assetGlobalWebsiteSearchStore := assetwiring.NewAssetGlobalWebsiteSearchStoreAdapter(repos.websiteRepo)
	assetGlobalEndpointSearchStore := assetwiring.NewAssetGlobalEndpointSearchStoreAdapter(repos.endpointRepo)
	assetStatisticsStore := assetwiring.NewAssetStatisticsStoreAdapter(repos.assetStatisticsRepo)
	globalAssetSearchSvc := assetservice.NewGlobalAssetSearchService(assetGlobalWebsiteSearchStore, assetGlobalEndpointSearchStore)
	assetStatisticsSvc := assetservice.NewAssetStatisticsQueryService(assetStatisticsStore, time.Now)

	websiteSvc := assetservice.NewWebsiteFacade(
		assetservice.NewWebsiteQueryService(assetWebsiteStore, assetScreenshotStore, assetTargetLookup),
		assetservice.NewWebsiteCommandService(assetWebsiteStore, assetTargetLookup),
	)
	subdomainSvc := assetservice.NewSubdomainFacade(
		assetservice.NewSubdomainQueryService(assetSubdomainStore, assetTargetLookup),
		assetservice.NewSubdomainCommandService(assetSubdomainStore, assetTargetLookup),
	)
	endpointSvc := assetservice.NewEndpointFacade(
		assetservice.NewEndpointQueryService(assetEndpointStore, assetTargetLookup),
		assetservice.NewEndpointCommandService(assetEndpointStore, assetTargetLookup),
	)
	directorySvc := assetservice.NewDirectoryFacade(
		assetservice.NewDirectoryQueryService(assetDirectoryStore, assetTargetLookup),
		assetservice.NewDirectoryCommandService(assetDirectoryStore, assetTargetLookup),
	)
	hostPortSvc := assetservice.NewHostPortFacade(
		assetservice.NewHostPortQueryService(assetHostPortStore, assetTargetLookup),
		assetservice.NewHostPortCommandService(assetHostPortStore, assetTargetLookup),
	)
	screenshotSvc := assetservice.NewScreenshotFacade(
		assetservice.NewScreenshotQueryService(assetScreenshotStore, assetTargetLookup),
		assetservice.NewScreenshotCommandService(assetScreenshotStore, assetTargetLookup),
	)

	return assetModuleWiring{
		websiteHandler:           websitehandler.NewWebsiteHandler(websiteSvc),
		subdomainHandler:         subdomainhandler.NewSubdomainHandler(subdomainSvc),
		endpointHandler:          endpointhandler.NewEndpointHandler(endpointSvc),
		directoryHandler:         directoryhandler.NewDirectoryHandler(directorySvc),
		hostPortHandler:          hostporthandler.NewHostPortHandler(hostPortSvc),
		screenshotHandler:        screenshothandler.NewScreenshotHandler(screenshotSvc),
		globalAssetSearchHandler: searchhandler.NewGlobalAssetSearchHandler(globalAssetSearchSvc),
		assetStatisticsHandler:   assethandler.NewAssetStatisticsHandler(assetStatisticsSvc),
		websiteSvc:               websiteSvc,
		subdomainSvc:             subdomainSvc,
		endpointSvc:              endpointSvc,
		directorySvc:             directorySvc,
		hostPortSvc:              hostPortSvc,
		screenshotSvc:            screenshotSvc,
	}
}

func wireFingerprintModule(repos *repositoryBundle, cfg *config.Config) (fingerprintModuleWiring, error) {
	artifacts, err := fingerprintapp.NewFingerprintArtifactService(repos.fingerprintRepo, cfg.Storage.FingerprintArtifactsBasePath)
	if err != nil {
		return fingerprintModuleWiring{}, err
	}
	service := fingerprintapp.NewFacadeWithArtifacts(repos.fingerprintRepo, artifacts)
	return fingerprintModuleWiring{handler: fingerprinthandler.NewFingerprintHandler(service), artifacts: artifacts}, nil
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

func wireScanModule(repos *repositoryBundle, infra *infra, cfg *config.Config, notifier agentservice.AgentMessagePublisher, wordlistCatalog scaninfra.PlanTaskWordlistCatalog, blacklistPolicyService blacklistapp.BlacklistPolicyApplicationService) (scanModuleWiring, error) {
	scanQueryStore := scanwiring.NewScanQueryStoreAdapter(repos.scanRepo)
	effectivePolicyResolver, err := scanwiring.NewEffectiveBlacklistPolicyResolverAdapter(blacklistPolicyService)
	if err != nil {
		return scanModuleWiring{}, err
	}
	scanCommandStore := scanwiring.NewScanCommandStoreAdapter(repos.scanRepo, effectivePolicyResolver)
	if scanCommandStore == nil {
		return scanModuleWiring{}, fmt.Errorf("initialize Scan command store")
	}
	scanDomainRepository := scanwiring.NewScanDomainRepositoryAdapter(repos.scanRepo)
	scanTaskStore := scanwiring.NewScanTaskStoreAdapter(repos.scanTaskRepo)
	scanTaskRuntimeScanStore := scanwiring.NewScanTaskRuntimeScanStoreAdapter(repos.scanRepo)
	taskProgressLogQueryStore := taskprogresslogwiring.NewTaskProgressLogQueryStoreAdapter(repos.taskProgressLogRepo)
	taskProgressLogCommandStore := taskprogresslogwiring.NewTaskProgressLogCommandStoreAdapter(repos.taskProgressLogRepo)
	scanStopStore := scanwiring.NewScanStopStoreAdapter(repos.scanRepo)
	scanTargetLookup := scanwiring.NewScanTargetLookupAdapter(
		repos.targetRepo,
		catalogservice.NewTargetCommandService(catalogwiring.NewCatalogTargetCommandStoreAdapter(repos.targetRepo), nil),
		repos.orgRepo,
	)
	taskProgressLogLookup := taskprogresslogwiring.NewTaskProgressLogScanLookupAdapter(repos.scanRepo)
	configResourceResolver, configResourceValidator, err := scanwiring.NewConfigResourceValidationComponents(infra.installedEngineQuery, wordlistCatalog)
	if err != nil {
		return scanModuleWiring{}, err
	}
	scanSvc, err := scanwiring.NewScanApplicationService(
		scanQueryStore,
		scanCommandStore,
		scanDomainRepository,
		scanStopStore,
		notifier,
		scanTargetLookup,
		scanwiring.NewScanCreateAgentLookupAdapter(repos.agentRepo),
		repos.scanWorkflowRepo,
		repos.engineRepo,
		infra.installedEngineQuery,
		configResourceResolver,
		cfg.EngineInstall.CFAcceleration,
	)
	if err != nil {
		return scanModuleWiring{}, err
	}
	scheduledScanSvc := scheduledscanapp.NewScheduledScanService(repos.scheduledScanRepo, repos.scanWorkflowRepo).
		WithAgentLookup(scanwiring.NewScanCreateAgentLookupAdapter(repos.agentRepo)).
		WithConfigResourceValidator(configResourceValidator)
	scheduledScanController := scheduledscanapp.NewSchedulerController(
		repos.scheduledScanRepo,
		scheduledscanapp.NewOccurrenceDispatcher(scanSvc),
	)
	occurrenceRetentionJob := scheduledscanapp.NewOccurrenceRetentionJob(repos.scheduledScanRepo)
	scanTaskSvc := scanwiring.NewScanTaskApplicationService(scanTaskStore, scanTaskRuntimeScanStore, repos.subdomainSnapshotRepo, repos.hostPortSnapshotRepo)
	taskProgressLogSvc := taskprogresslogwiring.NewTaskProgressLogApplicationService(taskProgressLogQueryStore, taskProgressLogCommandStore, taskProgressLogLookup)

	return scanModuleWiring{
		scanHandler: scanhandler.NewScanHandler(scanSvc, scanhandler.ScanHistoryRetentionPolicy{
			MinimumRetentionSeconds: int64(cfg.ScanHistoryRetention.Retention.Seconds()),
			AutomaticCleanupEnabled: cfg.ScanHistoryRetention.Mode == "enforce",
		}),
		scheduledScanHandler:    scheduledscanhandler.NewScheduledScanHandler(scheduledScanSvc),
		taskProgressLogHandler:  scanhandler.NewTaskProgressLogHandler(taskProgressLogSvc),
		scanTaskBridge:          scanTaskSvc,
		scanTaskSvc:             scanTaskSvc,
		taskProgressLogService:  taskProgressLogSvc,
		scanSvc:                 scanSvc,
		scheduledScanController: scheduledScanController,
		occurrenceRetentionJob:  occurrenceRetentionJob,
	}, nil
}

func wireAgentModule(
	repos *repositoryBundle,
	infra *infra,
	cfg *config.Config,
	scanTaskBridge agentservice.ScanTaskBridgePort,
	messageBus agentservice.AgentMessagePublisher,
	locationCoordinator *geolocation.Coordinator,
	serverLocationReader agentservice.ServerLocationReader,
) (agentModuleHandlers, error) {
	agentClock := agentinfra.NewSystemClock()
	agentTokenGenerator := agentinfra.NewCryptoTokenGenerator()
	agentSvc := agentservice.NewAgentFacade(
		agentservice.NewAgentQueryService(repos.agentRepo),
		agentservice.NewAgentCommandService(repos.agentRepo).WithDeletionLifecycle(repos.scanRepo),
		agentservice.NewAgentRegistrationService(repos.agentRepo, repos.registrationTokenRepo, agentClock, agentTokenGenerator),
	)
	agentControlSvc := agentservice.NewAgentControlLifecycleService(
		repos.agentRepo,
		infra.heartbeatCache,
		messageBus,
		agentClock,
		infra.agentVersion,
		infra.agentImageRef,
	)
	agentTaskSvc := agentservice.NewAgentTaskService(scanTaskBridge)
	locationLookup, err := agentinfra.NewCoordinatorAgentLocationLookup(locationCoordinator)
	if err != nil {
		return agentModuleHandlers{}, err
	}
	locationObserver, err := agentservice.NewAgentLocationObserver(repos.agentLocationRepo, locationLookup, agentClock)
	if err != nil {
		return agentModuleHandlers{}, err
	}
	clusterSummaryService, err := agentservice.NewAgentClusterSummaryService(repos.agentRepo, agentClock)
	if err != nil {
		return agentModuleHandlers{}, err
	}
	locationMapService, err := agentservice.NewAgentLocationMapService(repos.agentRepo, serverLocationReader, agentClock)
	if err != nil {
		return agentModuleHandlers{}, err
	}
	lokiLogQuerySvc := agentservice.NewLokiLogQueryService(infra.lokiClient, cfg.JWT.Secret)

	return agentModuleHandlers{
		agentHandler: agenthandler.NewAgentHandler(
			agentSvc,
			agentControlSvc,
			infra.agentVersion,
			cfg.PublicURL,
			fmt.Sprintf("http://server:%d", cfg.Server.GRPCPort),
			infra.agentImageRef,
			infra.sharedDataVolumeBind,
			infra.heartbeatCache,
		),
		agentLogHandler:            agenthandler.NewAgentLogHandler(agentSvc, lokiLogQuerySvc),
		agentClusterSummaryHandler: agenthandler.NewAgentClusterSummaryHandler(clusterSummaryService),
		agentLocationMapHandler:    agenthandler.NewAgentLocationMapHandler(locationMapService),

		agentControlService:         agentControlSvc,
		agentTaskService:            agentTaskSvc,
		agentService:                agentSvc,
		lokiLogQueryService:         lokiLogQuerySvc,
		agentLocationObserver:       locationObserver,
		registrationTokenCleanupJob: agentservice.NewRegistrationTokenCleanupJob(repos.registrationTokenRepo, agentClock, time.Hour),
	}, nil
}

func wireSystemModule(
	repos *repositoryBundle,
	infra *infra,
	cfg *config.Config,
	locationCoordinator *geolocation.Coordinator,
) (systemModuleHandlers, error) {
	serverLogSvc := systemservice.NewServerLogService(infra.lokiClient, cfg.JWT.Secret)
	// Runtime metrics describe the server shared-data boundary, not any optional
	// business subdirectory that may be created lazily by a specific module.
	runtimeMetricsSvc := systemservice.NewRuntimeMetricsService(
		systemservice.NewOSRuntimeMetricsSampler(sharedstorage.SharedDataRoot),
		systemservice.NewOSRuntimeMetricsMetadataProvider(sharedstorage.SharedDataRoot),
	)
	serverLocationRuntime := systeminfra.NewServerLocationRuntime()
	serverLocationLookup, err := systeminfra.NewCoordinatorServerLocationLookup(locationCoordinator)
	if err != nil {
		return systemModuleHandlers{}, err
	}
	serverLocationReader, err := systemservice.NewServerLocationReader(repos.serverLocationRepo, serverLocationRuntime)
	if err != nil {
		return systemModuleHandlers{}, err
	}
	serverLocationScheduler, err := systemservice.NewServerLocationScheduler(repos.serverLocationRepo, serverLocationLookup, serverLocationRuntime)
	if err != nil {
		return systemModuleHandlers{}, err
	}
	return systemModuleHandlers{
		serverLogHandler:        systemhandler.NewServerLogHandler(serverLogSvc),
		serverLogService:        serverLogSvc,
		runtimeMetricsHandler:   systemhandler.NewRuntimeMetricsHandler(runtimeMetricsSvc),
		runtimeMetricsJob:       runtimeMetricsSvc,
		serverLocationReader:    serverLocationReader,
		serverLocationScheduler: serverLocationScheduler,
	}, nil
}

func wireSnapshotModule(
	db *gorm.DB,
	repos *repositoryBundle,
	asset assetModuleWiring,
	security securityModuleWiring,
) snapshotModuleHandlers {
	materializationCoordinator := snapshotwiring.NewSnapshotMaterializationCoordinator(db, repos.scanRepo)
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
	vulnerabilityNotificationWriter := snapshotwiring.NewSnapshotVulnerabilityNotificationOccurrenceWriter(db, repos.notificationProducer)

	websiteSnapshotSvc := snapshotwiring.NewSnapshotWebsiteApplicationService(websiteSnapshotQueryStore, websiteSnapshotCommandStore, snapshotScanLookup, websiteAssetSync)
	subdomainSnapshotSvc := snapshotwiring.NewSnapshotSubdomainApplicationService(subdomainSnapshotQueryStore, subdomainSnapshotCommandStore, snapshotScanLookup, subdomainAssetSync)
	endpointSnapshotSvc := snapshotwiring.NewSnapshotEndpointApplicationService(endpointSnapshotQueryStore, endpointSnapshotCommandStore, snapshotScanLookup, endpointAssetSync)
	directorySnapshotSvc := snapshotwiring.NewSnapshotDirectoryApplicationService(directorySnapshotQueryStore, directorySnapshotCommandStore, snapshotScanLookup, directoryAssetSync, materializationCoordinator)
	hostPortSnapshotSvc := snapshotwiring.NewSnapshotHostPortApplicationService(hostPortSnapshotQueryStore, hostPortSnapshotCommandStore, snapshotScanLookup, hostPortAssetSync)
	screenshotSnapshotSvc := snapshotwiring.NewSnapshotScreenshotApplicationService(screenshotSnapshotQueryStore, screenshotSnapshotCommandStore, snapshotScanLookup, screenshotAssetSync, materializationCoordinator)
	vulnerabilitySnapshotSvc := snapshotwiring.NewSnapshotVulnerabilityApplicationService(vulnerabilitySnapshotQueryStore, vulnerabilitySnapshotCommandStore, snapshotScanLookup, vulnerabilityAssetSync, vulnerabilityRawOutputCodec, materializationCoordinator, vulnerabilityNotificationWriter)

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
		directorySnapshotService:     directorySnapshotSvc,
		hostPortSnapshotService:      hostPortSnapshotSvc,
		screenshotSnapshotService:    screenshotSnapshotSvc,
		vulnerabilitySnapshotService: vulnerabilitySnapshotSvc,
	}
}
