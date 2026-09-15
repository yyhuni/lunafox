package bootstrap

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"gorm.io/gorm"
)

type agentBootstrapCredentials struct {
	AgentID             int    `json:"agentId"`
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

	// The transaction lock serializes the filesystem/DB handoff. A crash after
	// publishing the file but before commit deliberately leaves a mismatch that
	// requires repair; it must never silently register another Agent.
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(0x4c46424f4f545354)).Error; err != nil {
			return err
		}
		var bindings []struct{ InstanceID string }
		if err := tx.Raw("SELECT instance_id FROM deployment_agent_bootstrap WHERE singleton = TRUE").Scan(&bindings).Error; err != nil {
			return err
		}
		bound := ""
		if len(bindings) == 1 {
			bound = bindings[0].InstanceID
		}
		repository := agentrepo.NewAgentRepository(tx)
		return ensureAgentBootstrapCredentials(credentialsPath, bound,
			func(token string) (*agentdomain.Agent, error) {
				return repository.FindByAuthenticationToken(ctx, token)
			},
			func() (*agentdomain.Agent, error) {
				_, count, err := repository.List(ctx, 1, 1, "", "")
				if err != nil {
					return nil, err
				}
				if count != 0 {
					return nil, errors.New("existing Agent data has no Compose bootstrap binding; explicit migration is required")
				}
				service := agentapp.NewAgentRegistrationService(repository, agentrepo.NewRegistrationTokenRepository(tx), agentinfra.NewSystemClock(), agentinfra.NewCryptoTokenGenerator())
				registration, err := service.CreateRegistrationToken(ctx)
				if err != nil {
					return nil, fmt.Errorf("create bootstrap registration: %w", err)
				}
				agent, err := service.RegisterAgent(ctx, registration.Token, hostname, agentVersion, agentdomain.AgentRegistrationOptions{})
				if err != nil {
					return nil, fmt.Errorf("register bootstrap agent: %w", err)
				}
				if err := tx.Exec("INSERT INTO deployment_agent_bootstrap (singleton, instance_id) VALUES (TRUE, ?)", agent.InstanceID).Error; err != nil {
					return nil, err
				}
				return agent, nil
			})
	})
}

func ensureAgentBootstrapCredentials(path, boundInstance string, find func(string) (*agentdomain.Agent, error), create func() (*agentdomain.Agent, error)) error {
	credentials, err := readAgentBootstrapCredentials(path)
	if err != nil {
		return err
	}
	if credentials != nil {
		if boundInstance == "" || boundInstance != credentials.InstanceID {
			return errors.New("bootstrap credentials do not match the database binding; explicit repair is required")
		}
		agent, err := find(credentials.AuthenticationToken)
		if err != nil {
			return fmt.Errorf("validate persisted bootstrap identity: %w", err)
		}
		if agent == nil || agent.ID != credentials.AgentID || agent.InstanceID != credentials.InstanceID || subtle.ConstantTimeCompare([]byte(agent.AuthenticationToken), []byte(credentials.AuthenticationToken)) != 1 {
			return errors.New("bootstrap credentials do not match the registered Agent; explicit repair is required")
		}
		return nil
	}
	if boundInstance != "" {
		return errors.New("bootstrap credentials are missing for an existing database binding; explicit repair is required")
	}
	agent, err := create()
	if err != nil {
		return err
	}
	return writeAgentBootstrapCredentials(path, agent)
}

func readAgentBootstrapCredentials(path string) (*agentBootstrapCredentials, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect bootstrap credentials: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() > 4096 {
		return nil, errors.New("bootstrap credentials must be a mode-0600 regular file of at most 4096 bytes")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bootstrap credentials: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var credentials agentBootstrapCredentials
	if err := decoder.Decode(&credentials); err != nil {
		return nil, errors.New("invalid bootstrap credentials JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("unexpected trailing bootstrap credentials data")
	}
	if credentials.AgentID <= 0 || strings.TrimSpace(credentials.InstanceID) == "" || strings.TrimSpace(credentials.DisplayName) == "" || len(credentials.AuthenticationToken) != 8 || strings.Trim(credentials.AuthenticationToken, "0123456789abcdef") != "" {
		return nil, errors.New("bootstrap credentials are incomplete or invalid")
	}
	return &credentials, nil
}

func writeAgentBootstrapCredentials(path string, agent *agentdomain.Agent) error {
	if agent == nil {
		return errors.New("bootstrap agent is required")
	}
	if agent.ID <= 0 || strings.TrimSpace(agent.InstanceID) == "" || strings.TrimSpace(agent.DisplayName) == "" || strings.TrimSpace(agent.AuthenticationToken) == "" {
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
		AgentID:             agent.ID,
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
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync bootstrap credentials: %w", err)
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
