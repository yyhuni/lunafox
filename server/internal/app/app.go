package app

import (
	"context"
	"embed"

	"github.com/yyhuni/lunafox/server/internal/bootstrap"
	"github.com/yyhuni/lunafox/server/internal/config"
)

// Run delegates to bootstrap package for dependency wiring and server lifecycle.
func Run(ctx context.Context, cfg *config.Config, migrationsFS embed.FS) {
	bootstrap.Run(ctx, cfg, migrationsFS)
}
