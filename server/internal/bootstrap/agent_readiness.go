package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/config"
	"github.com/yyhuni/lunafox/server/internal/database"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
	"gorm.io/gorm"
)

// ResidentAgentReadiness is the deployment-facing readiness of the Agent bound
// to this installation. It reuses the Server's existing database trust boundary
// instead of a user session, because a lifecycle command has no administrator
// password and must not authenticate as one.
//
// Readiness here means "can this Agent accept work now". It deliberately omits
// the Upgrade Operation's DesiredVersion and TargetDigest: matching an upgrade
// target is an Operation concern, and requiring it would make an ordinary start
// fail whenever the Agent runs a different release than a pending upgrade.
type ResidentAgentReadiness struct {
	InstanceID  string
	DisplayName string
	Connected   bool
	Healthy     bool
	Paused      bool
	ClaimReady  bool
	Diagnostic  string
}

// Ready reports whether the bound Agent satisfies every lifecycle condition.
func (readiness ResidentAgentReadiness) Ready() bool {
	return readiness.Connected && readiness.Healthy && !readiness.Paused && readiness.ClaimReady
}

type residentAgentReadinessStore interface {
	boundInstanceID(ctx context.Context) (string, error)
	findAgent(ctx context.Context, instanceID string) (*agentdomain.Agent, error)
}

// RunResidentAgentReadiness reports whether the resident Agent can accept work.
// It opens the bounded database configuration only: no JWT, Redis, Loki, or
// administrator credentials are involved.
func RunResidentAgentReadiness(ctx context.Context, databaseConfig *config.DatabaseConfig) (ResidentAgentReadiness, error) {
	if ctx == nil {
		return ResidentAgentReadiness{}, errors.New("resident Agent readiness context is required")
	}
	if databaseConfig == nil {
		return ResidentAgentReadiness{}, errors.New("resident Agent readiness database configuration is required")
	}
	db, err := database.NewDatabase(databaseConfig)
	if err != nil {
		return ResidentAgentReadiness{}, fmt.Errorf("connect database for resident Agent readiness: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return ResidentAgentReadiness{}, fmt.Errorf("access database connection for resident Agent readiness: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	return residentAgentReadiness(ctx, databaseResidentAgentReadinessStore{db: db.WithContext(ctx)}, time.Now().UTC())
}

type databaseResidentAgentReadinessStore struct {
	db *gorm.DB
}

func (store databaseResidentAgentReadinessStore) boundInstanceID(ctx context.Context) (string, error) {
	var bindings []struct{ InstanceID string }
	if err := store.db.WithContext(ctx).Raw("SELECT instance_id FROM deployment_agent_bootstrap WHERE singleton = TRUE").Scan(&bindings).Error; err != nil {
		return "", err
	}
	if len(bindings) != 1 {
		return "", nil
	}
	return strings.TrimSpace(bindings[0].InstanceID), nil
}

func (store databaseResidentAgentReadinessStore) findAgent(ctx context.Context, instanceID string) (*agentdomain.Agent, error) {
	repository := agentrepo.NewAgentRepository(store.db.WithContext(ctx))
	agent, err := repository.GetByInstanceID(ctx, instanceID)
	if err != nil {
		if errors.Is(err, agentdomain.ErrAgentNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return agent, nil
}

// residentAgentReadiness is the testable core: it resolves the bootstrap
// binding, observes the bound Agent, and reports one verdict.
func residentAgentReadiness(ctx context.Context, store residentAgentReadinessStore, now time.Time) (ResidentAgentReadiness, error) {
	if store == nil {
		return ResidentAgentReadiness{}, errors.New("resident Agent readiness store is required")
	}
	instanceID, err := store.boundInstanceID(ctx)
	if err != nil {
		return ResidentAgentReadiness{}, fmt.Errorf("read the deployment Agent binding: %w", err)
	}
	if instanceID == "" {
		return ResidentAgentReadiness{Diagnostic: "the deployment has no Agent binding yet"}, nil
	}
	agent, err := store.findAgent(ctx, instanceID)
	if err != nil {
		return ResidentAgentReadiness{}, fmt.Errorf("read the bound Agent: %w", err)
	}
	if agent == nil {
		return ResidentAgentReadiness{
			InstanceID: instanceID,
			Diagnostic: "the bound Agent is not registered",
		}, nil
	}
	observation := observeAgent(agent, now.UTC(), upgradeAgentHeartbeatFreshness)
	readiness := ResidentAgentReadiness{
		InstanceID:  instanceID,
		DisplayName: strings.TrimSpace(agent.DisplayName),
		Connected:   observation.Connected,
		Healthy:     observation.Healthy,
		Paused:      observation.Paused,
		ClaimReady:  observation.ClaimReady,
		Diagnostic:  observation.Diagnostic,
	}
	if !readiness.Ready() && readiness.Diagnostic == "" {
		readiness.Diagnostic = "the resident Agent is not ready"
	}
	return readiness, nil
}

// Describe renders one operator-facing line. It never includes credentials,
// tokens, or environment values.
func (readiness ResidentAgentReadiness) Describe() string {
	if readiness.Ready() {
		if readiness.DisplayName != "" {
			return fmt.Sprintf("resident Agent %s (%s) is claim-ready", readiness.DisplayName, readiness.InstanceID)
		}
		return fmt.Sprintf("resident Agent %s is claim-ready", readiness.InstanceID)
	}
	detail := readiness.Diagnostic
	if detail == "" {
		detail = "the resident Agent is not ready"
	}
	if readiness.InstanceID == "" {
		return fmt.Sprintf("resident Agent is not claim-ready: %s", detail)
	}
	return fmt.Sprintf("resident Agent %s is not claim-ready: %s", readiness.InstanceID, detail)
}

var _ residentAgentReadinessStore = databaseResidentAgentReadinessStore{}
