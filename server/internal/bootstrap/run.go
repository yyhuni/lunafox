package bootstrap

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/config"
	runtimegrpc "github.com/yyhuni/lunafox/server/internal/grpc/runtime/server"
	runtimesvc "github.com/yyhuni/lunafox/server/internal/grpc/runtime/service"
	"github.com/yyhuni/lunafox/server/internal/job"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	d := buildDependencies(infra, cfg)

	runtimeServer, err := newRuntimeServer(cfg, d)
	if err != nil {
		pkg.Fatal("Failed to initialize runtime gRPC server", zap.Error(err))
	}

	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	engine.Use(middleware.Recovery())
	engine.Use(middleware.Logger())

	registerRoutes(engine, d, middleware.AuthMiddleware(infra.jwtManager))

	jobCtx, jobCancel := context.WithCancel(context.Background())
	defer jobCancel()
	agentMonitor := job.NewAgentMonitor(d.agentRepo, d.scanTaskRepo, time.Minute, 120*time.Second)
	go agentMonitor.Run(jobCtx)

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
		pkg.Info("Runtime gRPC listening", zap.String("addr", runtimeServer.Addr()))
		if err := runtimeServer.Serve(); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			pkg.Fatal("Failed to start runtime gRPC server", zap.Error(err))
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
	if err := runtimeServer.Shutdown(shutdownCtx); err != nil {
		pkg.Error("Runtime gRPC forced to shutdown", zap.Error(err))
	}

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

func newRuntimeServer(cfg *config.Config, d *deps) (*runtimegrpc.Server, error) {
	return runtimegrpc.New(
		fmt.Sprintf(":%d", cfg.Server.GRPCPort),
		runtimesvc.NewAgentRuntimeServiceWithDeps(
			d.agentRepo,
			d.agentRuntimeService,
			d.agentTaskService,
			d.runtimeStreamRegistry,
		),
		runtimesvc.NewAgentDataProxyServiceWithDeps(
			d.workerProviderConfigService,
			d.wordlistService,
			runtimesvc.AssetBatchUpsertRuntimes{
				Subdomains: d.subdomainSnapshotService,
				Websites:   d.websiteSnapshotService,
				Endpoints:  d.endpointSnapshotService,
				HostPorts:  d.hostPortSnapshotService,
			},
		).WithAgentFinder(d.agentRepo),
	)
}
