package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/config"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
)

const upgradeProbeTimeout = 5 * time.Second

// newUpgradeVerifier binds the application verifier to the dependencies that
// already have lifecycle ownership in bootstrap. The verifier observes only;
// it does not restart services or mutate database state.
func newUpgradeVerifier(infra *infra, cfg *config.Config) (upgradeapp.UpgradeVerifier, error) {
	if infra == nil || cfg == nil {
		return nil, fmt.Errorf("upgrade verifier infrastructure is required")
	}
	publicURL := strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/")
	probes := map[string]upgradeapp.HealthProbe{
		"postgres": func(ctx context.Context) error {
			if infra.db == nil {
				return fmt.Errorf("postgres is not configured")
			}
			db, err := infra.db.DB()
			if err != nil {
				return err
			}
			probeCtx, cancel := context.WithTimeout(ctx, upgradeProbeTimeout)
			defer cancel()
			return db.PingContext(probeCtx)
		},
		"redis": func(ctx context.Context) error {
			if infra.redisClient == nil {
				return fmt.Errorf("redis is not configured")
			}
			probeCtx, cancel := context.WithTimeout(ctx, upgradeProbeTimeout)
			defer cancel()
			return infra.redisClient.Ping(probeCtx).Err()
		},
		"loki": func(ctx context.Context) error {
			if infra.lokiClient == nil {
				return fmt.Errorf("loki is not configured")
			}
			probeCtx, cancel := context.WithTimeout(ctx, upgradeProbeTimeout)
			defer cancel()
			return infra.lokiClient.CheckReady(probeCtx)
		},
		"server": httpHealthProbe(serverHealthURL(cfg.Server.Port)),
		"api":    httpHealthProbe(serverHealthURL(cfg.Server.Port)),
		"frontend": func(ctx context.Context) error {
			if publicURL == "" {
				return fmt.Errorf("public frontend URL is not configured")
			}
			return httpHealthProbe(publicURL)(ctx)
		},
		"nginx": func(ctx context.Context) error {
			if publicURL == "" {
				return fmt.Errorf("public edge URL is not configured")
			}
			return httpHealthProbe(publicURL + "/healthChecks/readiness")(ctx)
		},
		"publicFrontend": func(ctx context.Context) error {
			if publicURL == "" {
				return fmt.Errorf("public frontend URL is not configured")
			}
			return httpHealthProbe(publicURL)(ctx)
		},
	}
	verifier, err := upgradeapp.NewCompositeVerifier(upgradeapp.CompositeVerifierConfig{
		Probes: probes,
		RequiredProbes: []string{
			"postgres", "redis", "loki", "server", "frontend", "nginx", "api", "publicFrontend",
		},
		RequiredDigests: []string{"server", "frontend", "nginx"},
	})
	if err != nil {
		return nil, err
	}
	return verifier, nil
}

func serverHealthURL(port int) string {
	if port <= 0 {
		port = 8080
	}
	return "http://127.0.0.1:" + strconv.Itoa(port) + "/healthChecks/readiness"
}

func httpHealthProbe(rawURL string) upgradeapp.HealthProbe {
	return func(ctx context.Context) error {
		rawURL = strings.TrimSpace(rawURL)
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("health probe URL is invalid")
		}
		probeCtx, cancel := context.WithTimeout(ctx, upgradeProbeTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, parsed.String(), nil)
		if err != nil {
			return err
		}
		response, err := (&http.Client{Timeout: upgradeProbeTimeout}).Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("health probe returned HTTP %d", response.StatusCode)
		}
		return nil
	}
}
