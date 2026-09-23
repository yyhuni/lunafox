package bootstrap

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/config"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
)

const (
	upgradeProbeTimeout = 5 * time.Second
	composeEdgeProbeURL = "https://nginx"
)

// newUpgradeVerifier binds the application verifier to the dependencies that
// already have lifecycle ownership in bootstrap. The verifier observes only;
// it does not restart services or mutate database state.
func newUpgradeVerifier(infra *infra, cfg *config.Config) (upgradeapp.UpgradeVerifier, error) {
	if infra == nil || cfg == nil {
		return nil, fmt.Errorf("upgrade verifier infrastructure is required")
	}
	publicHost, err := composeEdgePublicHost(cfg.PublicURL)
	if err != nil {
		return nil, fmt.Errorf("configure internal Compose edge probe: %w", err)
	}
	frontendProbe := composeEdgeHealthProbe(publicHost, "/")
	nginxProbe := composeEdgeHealthProbe(publicHost, "/healthChecks/readiness")
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
		"server":         httpHealthProbe(serverHealthURL(cfg.Server.Port)),
		"api":            httpHealthProbe(serverHealthURL(cfg.Server.Port)),
		"frontend":       frontendProbe,
		"nginx":          nginxProbe,
		"publicFrontend": frontendProbe,
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

// composeEdgePublicHost derives the external Host routing value while keeping
// the network destination separate. PUBLIC_URL identifies the operator-facing
// address, which can be a host-only port mapping that is unreachable from the
// Server container itself.
func composeEdgePublicHost(rawPublicURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawPublicURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("PUBLIC_URL is invalid for Compose edge verification")
	}
	return parsed.Host, nil
}

// composeEdgeHealthProbe reaches the fixed Nginx service through Compose DNS.
// It deliberately never resolves PUBLIC_URL from inside Server: that address
// belongs to the operator-facing ingress and may point to the Docker host.
func composeEdgeHealthProbe(publicHost, requestPath string) upgradeapp.HealthProbe {
	return composeEdgeHealthProbeWithClient(composeEdgeProbeURL, publicHost, requestPath, newComposeEdgeHTTPClient())
}

func composeEdgeHealthProbeWithClient(rawEndpoint, publicHost, requestPath string, client *http.Client) upgradeapp.HealthProbe {
	return func(ctx context.Context) error {
		endpoint, err := url.Parse(strings.TrimSpace(rawEndpoint))
		if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
			return fmt.Errorf("compose edge probe endpoint is invalid")
		}
		if strings.TrimSpace(publicHost) == "" || !strings.HasPrefix(requestPath, "/") || client == nil {
			return fmt.Errorf("compose edge probe configuration is invalid")
		}
		endpoint.Path = requestPath
		endpoint.RawPath = ""
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return err
		}
		req.Host = publicHost
		response, err := client.Do(req)
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

func newComposeEdgeHTTPClient() *http.Client {
	// The target is the fixed Compose service, not an operator-provided URL.
	// cert-init creates a self-signed certificate for PUBLIC_HOST, while Docker
	// DNS resolves nginx. This readiness signal stays within the trusted Compose
	// network and has no external TLS-identity role.
	transport := &http.Transport{
		Proxy:             nil,
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{ // #nosec G402 -- fixed internal Compose endpoint with generated self-signed TLS.
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
	}
	return &http.Client{Timeout: upgradeProbeTimeout, Transport: transport}
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
