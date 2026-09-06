package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	scanwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/scan"
	"github.com/yyhuni/lunafox/server/internal/config"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	agentdata "github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	agentserver "github.com/yyhuni/lunafox/server/internal/grpc/agentserver"
	"github.com/yyhuni/lunafox/server/internal/job"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// Run wires dependencies and starts the HTTP server.
func Run(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) {
	pkg.Info(
		"Starting server",
		zap.Int("port", cfg.Server.Port),
		zap.Int("server.grpc.port", cfg.Server.GRPCPort),
		zap.String("mode", cfg.Server.Mode),
	)

	infra := initInfra(cfg, migrationsFS)
	d, err := buildDependencies(infra, cfg)
	if err != nil {
		pkg.Fatal("Failed to initialize application dependencies", zap.Error(err))
	}

	agentPlaneServer, err := newAgentPlaneServer(cfg, d)
	if err != nil {
		pkg.Fatal("Failed to initialize agent plane gRPC server", zap.Error(err))
	}

	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Logger())

	registerRoutes(engine, d, middleware.AuthMiddleware(infra.jwtManager, d.tokenVersionReader))

	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()
	agentMonitor := job.NewAgentMonitor(d.agentRepo, d.scanTaskRepo, d.scanTaskSvc, time.Minute, agentcontrol.DefaultSessionHeartbeatTimeout)
	go agentMonitor.Run(jobCtx)
	if d.runtimeMetricsJob != nil {
		d.runtimeMetricsJob.Start(jobCtx)
	}
	if d.scanHistoryRetentionJob != nil {
		d.scanHistoryRetentionJob.Start(jobCtx)
	}
	if d.registrationTokenCleanupJob != nil {
		d.registrationTokenCleanupJob.Start(jobCtx)
	}
	d.scheduledScanController.Start(jobCtx)
	d.occurrenceRetentionJob.Start(jobCtx)
	d.targetCleanupRunner.Start(jobCtx)
	d.serverLocationScheduler.Start(jobCtx)
	d.notificationOutboxWorker.Start(jobCtx)
	d.notificationDeliveryWorker.Start(jobCtx)
	d.notificationRetentionJob.Start(jobCtx)
	if d.nucleiPocRetentionJob != nil {
		d.nucleiPocRetentionJob.Start(jobCtx)
	}
	if d.nucleiPocRunner != nil {
		d.nucleiPocRunner.Start(jobCtx)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      middleware.NormalizeTrailingSlash(engine),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		pkg.Info("Server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pkg.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	go func() {
		pkg.Info("Agent plane gRPC listening", zap.String("addr", agentPlaneServer.Addr()))
		if err := agentPlaneServer.Serve(); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			pkg.Fatal("Failed to start agent plane gRPC server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	pkg.Info("Shutting down server...")
	jobCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		pkg.Error("Server forced to shutdown", zap.Error(err))
	}
	if err := agentPlaneServer.Shutdown(shutdownCtx); err != nil {
		pkg.Error("Agent plane gRPC forced to shutdown", zap.Error(err))
	}
	if d.agentLocationObserver != nil {
		if err := d.agentLocationObserver.Shutdown(shutdownCtx); err != nil {
			pkg.Error("Agent location observer forced to shutdown", zap.Error(err))
		}
	}
	waitForManagedBackgroundJob(shutdownCtx, "Server location scheduler", d.serverLocationScheduler)
	if d.geolocationCoordinator != nil {
		if err := d.geolocationCoordinator.Shutdown(shutdownCtx); err != nil {
			pkg.Error("Geolocation coordinator forced to shutdown", zap.Error(err))
		}
	}
	waitForManagedBackgroundJob(shutdownCtx, "scheduled scan controller", d.scheduledScanController)
	waitForManagedBackgroundJob(shutdownCtx, "scheduled scan occurrence retention", d.occurrenceRetentionJob)
	waitForManagedBackgroundJob(shutdownCtx, "Target cleanup runner", d.targetCleanupRunner)
	waitForManagedBackgroundJob(shutdownCtx, "notification outbox worker", d.notificationOutboxWorker)
	waitForManagedBackgroundJob(shutdownCtx, "notification delivery worker", d.notificationDeliveryWorker)
	waitForManagedBackgroundJob(shutdownCtx, "notification retention job", d.notificationRetentionJob)
	waitForManagedBackgroundJob(shutdownCtx, "nuclei POC retention job", d.nucleiPocRetentionJob)
	waitForManagedBackgroundJob(shutdownCtx, "nuclei POC sync runner", d.nucleiPocRunner)

	if sqlDB, err := infra.db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			pkg.Error("Failed to close database connection", zap.Error(err))
		}
	}
	if infra.redisClient != nil {
		if err := infra.redisClient.Close(); err != nil {
			pkg.Error("Failed to close Redis connection", zap.Error(err))
		}
	}

	pkg.Info("Server exited")
}

func waitForManagedBackgroundJob(ctx context.Context, name string, backgroundJob managedBackgroundJob) {
	if backgroundJob == nil {
		return
	}
	select {
	case <-backgroundJob.Done():
		return
	case <-ctx.Done():
		pkg.Error("Managed background job did not stop before Server shutdown deadline",
			zap.String("job", name),
			zap.Error(ctx.Err()),
		)
	}
}

func newAgentPlaneServer(cfg *config.Config, d *deps) (*agentserver.Server, error) {
	if cfg == nil || d == nil {
		return nil, fmt.Errorf("agent plane configuration and dependencies are required")
	}
	if d.agentTaskService == nil {
		return nil, fmt.Errorf("agent task service is required")
	}
	if _, ok := any(d.agentTaskService).(agentcontrol.EngineDiagnosticScanTaskBridge); !ok {
		return nil, fmt.Errorf("agent task service must implement the diagnostic terminal bridge")
	}
	sessions := agentcontrol.NewActiveSessionRegistry()
	controlPlane := agentcontrol.NewControlPlaneService(
		d.agentRepo,
		d.agentControlService,
		d.agentTaskService,
		d.runtimeStreamRegistry,
	).WithActiveSessionRegistry(sessions).
		WithReadyConnectionObserver(d.agentLocationObserver)
	artifactResolver, err := agentdata.NewServerExecutionArtifactResolver(agentdata.ServerExecutionArtifactResolverDependencies{
		Tasks:                 d.scanTaskRepo,
		Sessions:              sessions,
		DNSNames:              d.subdomainSnapshotRepo,
		HostPorts:             scanwiring.NewHostPortCursorAdapter(d.hostPortSnapshotRepo),
		WebsiteURLs:           d.websiteSnapshotRepo,
		EndpointURLs:          scanwiring.NewEndpointURLCursorAdapter(d.endpointSnapshotRepo),
		InventoryDNSNames:     scanwiring.NewTargetInventoryDNSNameCursorAdapter(d.subdomainRepo),
		InventoryHostPorts:    scanwiring.NewTargetInventoryHostPortCursorAdapter(d.hostPortRepo),
		InventoryWebsiteURLs:  scanwiring.NewTargetInventoryWebsiteURLCursorAdapter(d.websiteRepo),
		InventoryEndpointURLs: scanwiring.NewTargetInventoryEndpointURLCursorAdapter(d.endpointRepo),
		BlacklistSnapshots:    agentdata.NewExecutionInputBlacklistSnapshotSource(scanwiring.NewScanBlacklistSnapshotStoreAdapter(d.scanRepo)),
		Wordlists:             d.executionWordlistSource,
		ProviderConfig:        d.executionProviderConfigSource,
		FingerprintArtifacts:  d.fingerprintArtifacts,
		NucleiTemplates:       d.nucleiPocRepo,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize execution artifact resolver: %w", err)
	}
	return agentserver.New(
		fmt.Sprintf(":%d", cfg.Server.GRPCPort),
		controlPlane,
		agentdata.NewDataPlaneService(
			d.agentRepo,

			agentdata.ResultIngestDataPlanes{
				TaskScopes: agentdata.NewResultTaskScopeDataPlane(d.scanTaskRepo, d.scanRepo),
				Ingest:     d.resultIngestService,
			}).
			WithTaskProgressLogDataPlane(agentdata.NewTaskProgressLogDataPlane(d.taskProgressLogService, d.scanTaskRepo)).WithAgentSessionReader(sessions),
		agentdata.NewExecutionArtifactService(d.agentRepo, artifactResolver),
		grpc.MaxRecvMsgSize(agentdata.MaxDataPlaneRecvMessageBytes),
		// gRPC keepalive: send HTTP/2 PINGs to keep long-lived agent streams
		// alive through Docker bridge networks and other intermediate devices.
		// Must be configured on both server and client sides.
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    10 * time.Second,
			Timeout: 20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
}
