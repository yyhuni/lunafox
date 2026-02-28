package config

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	legacyDefaultJWTSecret   = "change-me-in-production-use-a-long-random-string"
	legacyDefaultWorkerToken = "change-me-worker-token"
)

// validateConfig enforces required runtime contracts and blocks weak placeholder values.
func validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	jwtSecret := strings.TrimSpace(cfg.JWT.Secret)
	if jwtSecret == "" || jwtSecret == legacyDefaultJWTSecret {
		return fmt.Errorf("JWT_SECRET is required and cannot use weak placeholder value")
	}

	workerToken := strings.TrimSpace(cfg.Worker.Token)
	if workerToken == "" || workerToken == legacyDefaultWorkerToken {
		return fmt.Errorf("WORKER_TOKEN is required and cannot use weak placeholder value")
	}

	publicURL := strings.TrimSpace(cfg.PublicURL)
	if publicURL == "" {
		return fmt.Errorf("PUBLIC_URL is required")
	}
	parsed, err := url.Parse(publicURL)
	if err != nil {
		return fmt.Errorf("PUBLIC_URL is invalid: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("PUBLIC_URL must use https scheme")
	}
	if parsed.Host == "" {
		return fmt.Errorf("PUBLIC_URL host is required")
	}

	return nil
}
