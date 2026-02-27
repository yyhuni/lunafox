package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yyhuni/lunafox/agent/internal/app"
	"github.com/yyhuni/lunafox/agent/internal/config"
	"github.com/yyhuni/lunafox/agent/internal/logger"
	"go.uber.org/zap"
)

func main() {
	if err := logger.Init(os.Getenv("LOG_LEVEL")); err != nil {
		fmt.Fprintf(os.Stderr, "logger init failed: %v\n", err)
	}
	defer logger.Sync()

	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		logger.Log.Fatal("failed to load config", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, *cfg); err != nil {
		logger.Log.Fatal("agent stopped", zap.Error(err))
	}
}
