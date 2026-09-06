package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/contracts/versioning"
	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	agentinfra "github.com/yyhuni/lunafox/server/internal/modules/agent/infrastructure"
	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
)

type agentBootstrapCredentials struct {
	InstanceID          string `json:"instanceId"`
	DisplayName         string `json:"displayName"`
	AuthenticationToken string `json:"agentAuthenticationToken"`
}

// RunAgentBootstrap creates the first in-network Agent using the existing
// registration protocol. The persisted credential remains the current
// eight-character hexadecimal bearer baseline; this bootstrap does not add
// TLS, rotation, or revocation semantics.
func RunAgentBootstrap(ctx context.Context, databaseConfig *config.DatabaseConfig, credentialsPath, hostname, agentVersion string) error {
	if ctx == nil {
		return errors.New("agent bootstrap context is required")
	}
	if databaseConfig == nil {
		return errors.New("agent bootstrap database configuration is required")
	}
	credentialsPath = strings.TrimSpace(credentialsPath)
	hostname = strings.TrimSpace(hostname)
	agentVersion = versioning.Normalize(agentVersion)
	if credentialsPath == "" || !filepath.IsAbs(credentialsPath) {
		return errors.New("agent bootstrap credentials path must be absolute")
	}
	if hostname == "" || hostname != strings.TrimSpace(hostname) {
		return errors.New("agent bootstrap hostname is required")
	}
	if !versioning.IsValidSemVer(agentVersion) {
		return errors.New(versioning.SemVerFieldMessage("agent bootstrap version"))
	}

	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		return fmt.Errorf("connect database for agent bootstrap: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access database connection for agent bootstrap: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	registrationService := agentapp.NewAgentRegistrationService(
		agentrepo.NewAgentRepository(db),
		agentrepo.NewRegistrationTokenRepository(db),
		agentinfra.NewSystemClock(),
		agentinfra.NewCryptoTokenGenerator(),
	)
	registration, err := registrationService.CreateRegistrationToken(ctx)
	if err != nil {
		return fmt.Errorf("create bootstrap agent registration: %w", err)
	}
	agent, err := registrationService.RegisterAgent(ctx, registration.Token, hostname, agentVersion, agentdomain.AgentRegistrationOptions{})
	if err != nil {
		return fmt.Errorf("register bootstrap agent: %w", err)
	}
	if err := writeAgentBootstrapCredentials(credentialsPath, agent); err != nil {
		return err
	}
	return nil
}

func writeAgentBootstrapCredentials(path string, agent *agentdomain.Agent) error {
	if agent == nil {
		return errors.New("bootstrap agent is required")
	}
	if strings.TrimSpace(agent.InstanceID) == "" || strings.TrimSpace(agent.DisplayName) == "" || strings.TrimSpace(agent.AuthenticationToken) == "" {
		return errors.New("bootstrap agent credentials are incomplete")
	}
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return errors.New("agent bootstrap credentials path must be absolute")
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("agent bootstrap credentials path already exists: %s", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect agent bootstrap credentials path: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create agent bootstrap credentials directory: %w", err)
	}

	payload, err := json.Marshal(agentBootstrapCredentials{
		InstanceID:          agent.InstanceID,
		DisplayName:         agent.DisplayName,
		AuthenticationToken: agent.AuthenticationToken,
	})
	if err != nil {
		return fmt.Errorf("encode agent bootstrap credentials: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".agent-bootstrap-")
	if err != nil {
		return fmt.Errorf("create temporary agent bootstrap credentials: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("restrict agent bootstrap credentials: %w", err)
	}
	if _, err := temporary.Write(append(payload, '\n')); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write agent bootstrap credentials: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close agent bootstrap credentials: %w", err)
	}
	// Link, rather than rename, preserves the no-overwrite contract if a second
	// bootstrap process races this one. Credential replacement is reset-only.
	if err := os.Link(temporaryPath, path); err != nil {
		return fmt.Errorf("publish agent bootstrap credentials: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("restrict published agent bootstrap credentials: %w", err)
	}
	return nil
}
