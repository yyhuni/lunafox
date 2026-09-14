package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	upgradeapp "github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	upgradedomain "github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
)

const (
	// Heartbeat freshness is intentionally shorter than the verification
	// deadline. A stale row must not be mistaken for an Agent that survived the
	// host restart and is ready to claim work.
	upgradeAgentHeartbeatFreshness = 90 * time.Second
	upgradeAgentPageSize           = 200
)

// upgradeAgentSource adapts the persisted Agent runtime projection and the
// existing control-plane publisher to the upgrade application port. It never
// claims work and never copies authentication tokens into the Operation.
type upgradeAgentSource struct {
	repository agentdomain.AgentOperationalRepository
	publisher  agentapp.AgentMessagePublisher
	now        func() time.Time
	freshness  time.Duration
}

func newUpgradeAgentSource(repository agentdomain.AgentOperationalRepository, publisher agentapp.AgentMessagePublisher, now func() time.Time) *upgradeAgentSource {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &upgradeAgentSource{
		repository: repository,
		publisher:  publisher,
		now:        now,
		freshness:  upgradeAgentHeartbeatFreshness,
	}
}

func (source *upgradeAgentSource) Snapshot(ctx context.Context, target upgradeapp.AgentUpgradeTarget) ([]upgradedomain.AgentExpectation, error) {
	if source == nil || source.repository == nil {
		return nil, fmt.Errorf("agent upgrade source is not configured")
	}
	if strings.TrimSpace(target.Version) == "" || strings.TrimSpace(target.Digest) == "" || strings.TrimSpace(target.ImageRef) == "" {
		return nil, fmt.Errorf("agent upgrade target is incomplete")
	}
	now := source.now().UTC()
	all := make([]upgradedomain.AgentExpectation, 0)
	for page := 1; ; page++ {
		agents, total, err := source.repository.List(ctx, page, upgradeAgentPageSize, "", "createdAt asc")
		if err != nil {
			return nil, fmt.Errorf("list Agents for upgrade snapshot: %w", err)
		}
		for _, agent := range agents {
			if agent == nil || agent.ID <= 0 {
				continue
			}
			all = append(all, source.expectation(agent, target, now))
		}
		if len(agents) == 0 || len(all) >= int(total) || len(agents) < upgradeAgentPageSize {
			break
		}
	}
	return all, nil
}

// Reconcile is an optional extension consumed by the application service after
// the host is back. The expected set is immutable: newly registered Agents are
// deliberately excluded from this Operation.
func (source *upgradeAgentSource) Reconcile(ctx context.Context, target upgradeapp.AgentUpgradeTarget, expected []upgradedomain.AgentExpectation) ([]upgradedomain.AgentExpectation, error) {
	if source == nil || source.repository == nil {
		return nil, fmt.Errorf("agent upgrade source is not configured")
	}
	now := source.now().UTC()
	current, err := source.Snapshot(ctx, target)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]upgradedomain.AgentExpectation, len(current))
	for _, item := range current {
		byID[item.AgentID] = item
	}
	result := make([]upgradedomain.AgentExpectation, 0, len(expected))
	for _, wanted := range expected {
		observed, ok := byID[wanted.AgentID]
		if !ok {
			wanted.Connected = false
			wanted.Healthy = false
			wanted.ClaimReady = false
			wanted.ObservedVersion = ""
			wanted.ObservedDigest = ""
			wanted.LastObservedAt = nil
			wanted.Diagnostic = "Agent is no longer registered"
			result = append(result, wanted)
			continue
		}
		observed.DesiredVersion = wanted.DesiredVersion
		observed.TargetDigest = wanted.TargetDigest
		result = append(result, observed)
	}
	// Keep the timestamp read above meaningful if a test clock returns a zero
	// value; expectation() normalizes it, while this guard documents that the
	// reconciliation is based on one observation instant.
	_ = now
	return result, nil
}

func (source *upgradeAgentSource) NotifyUpdateRequired(ctx context.Context, target upgradeapp.AgentUpgradeTarget, expectations []upgradedomain.AgentExpectation) error {
	if source == nil || source.publisher == nil {
		return nil
	}
	if strings.TrimSpace(target.Version) == "" || strings.TrimSpace(target.ImageRef) == "" {
		return fmt.Errorf("agent update target is incomplete")
	}
	payload := agentdomain.UpdateRequiredPayload{AgentVersion: target.Version, AgentImageRef: target.ImageRef}
	failed := 0
	for _, expectation := range expectations {
		if err := ctx.Err(); err != nil {
			return err
		}
		if expectation.AgentID <= 0 {
			continue
		}
		if !source.publisher.SendUpdateRequired(expectation.AgentID, payload) {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("update_required could not reach %d Agent(s)", failed)
	}
	return nil
}

func (source *upgradeAgentSource) expectation(agent *agentdomain.Agent, target upgradeapp.AgentUpgradeTarget, now time.Time) upgradedomain.AgentExpectation {
	lastHeartbeat := cloneTime(agent.LastHeartbeat)
	connected := strings.EqualFold(strings.TrimSpace(agent.Status), "online") && heartbeatFresh(lastHeartbeat, now, source.freshness)
	healthState := strings.ToLower(strings.TrimSpace(agent.HealthState))
	healthy := healthState == "healthy"
	paused := healthState == "paused"
	claimReady := connected && agent.ContainerRuntimeReady && strings.EqualFold(strings.TrimSpace(agent.OperatingSystem), "linux") && supportedAgentArchitecture(agent.Architecture) && len(agent.SupportedEngineAPIMajors) > 0 && strings.TrimSpace(agent.SessionID) != "" && agent.SessionEpoch > 0
	diagnostic := ""
	switch {
	case !connected:
		diagnostic = "Agent heartbeat is missing or stale"
	case paused:
		diagnostic = "Agent is paused"
	case !healthy:
		diagnostic = firstNonEmpty(agent.HealthReason, "Agent health is not healthy")
	case strings.TrimSpace(agent.AgentVersion) != target.Version:
		diagnostic = "Agent is running a different version"
	case !claimReady:
		diagnostic = "Agent runtime is not ready to claim work"
	}
	return upgradedomain.AgentExpectation{
		AgentID:         agent.ID,
		DesiredVersion:  target.Version,
		TargetDigest:    target.Digest,
		ObservedVersion: strings.TrimSpace(agent.AgentVersion),
		// The current heartbeat contract has no image digest. Leave this empty
		// rather than treating a configured target as observed evidence.
		ObservedDigest: "",
		Connected:      connected,
		Healthy:        healthy,
		Paused:         paused,
		ClaimReady:     claimReady,
		LastObservedAt: lastHeartbeat,
		Diagnostic:     diagnostic,
	}
}

func heartbeatFresh(observed *time.Time, now time.Time, freshness time.Duration) bool {
	if observed == nil || observed.IsZero() {
		return false
	}
	if freshness <= 0 {
		freshness = upgradeAgentHeartbeatFreshness
	}
	age := now.UTC().Sub(observed.UTC())
	return age >= 0 && age <= freshness
}

func supportedAgentArchitecture(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "amd64", "x86_64", "arm64", "aarch64":
		return true
	default:
		return false
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

var _ upgradeapp.AgentUpgradeSource = (*upgradeAgentSource)(nil)
