package application

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/cache"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// AgentControlLifecycleService orchestrates runtime lifecycle actions for agent heartbeats.
type AgentControlLifecycleService struct {
	agentRepo           agentdomain.AgentRepository
	heartbeatCache      HeartbeatCachePort
	messageBus          AgentMessagePublisher
	clock               Clock
	desiredAgentVersion string
	agentImageRef       string
	updateNotifier      *updateNotifier
}

func NewAgentControlLifecycleService(
	agentRepo agentdomain.AgentRepository,
	heartbeatCache HeartbeatCachePort,
	messageBus AgentMessagePublisher,
	clock Clock,
	desiredAgentVersion, agentImageRef string,
) *AgentControlLifecycleService {
	if clock == nil {
		panic("clock is required")
	}

	return &AgentControlLifecycleService{
		agentRepo:           agentRepo,
		heartbeatCache:      heartbeatCache,
		messageBus:          messageBus,
		clock:               clock,
		desiredAgentVersion: desiredAgentVersion,
		agentImageRef:       agentImageRef,
		updateNotifier:      newUpdateNotifier(messageBus, desiredAgentVersion, agentImageRef),
	}
}

// OnConnected persists the Server-observed current connection address before
// SessionReady. It is presentation/GeoIP data only and must never affect
// authentication, authorization, scheduling, or policy.
func (service *AgentControlLifecycleService) OnConnected(ctx context.Context, agent *agentdomain.Agent, connectionIP string) error {
	if agent == nil || agent.ID <= 0 {
		return fmt.Errorf("agent identity is required")
	}
	normalizedConnection, err := normalizeObservedConnectionSource(connectionIP)
	if err != nil {
		return err
	}
	now := service.clock.NowUTC()
	observation, err := service.agentRepo.RecordConnection(ctx, agent.ID, normalizedConnection, now)
	if err != nil {
		return err
	}
	agent.Status = "online"
	agent.ConnectedAt = &now
	agent.ConnectionIP = observation.ConnectionIP
	agent.ObservedSourceIP = observation.SourceIP
	agent.ObservedIPGeneration = observation.Generation
	if observation.SourceChanged && agent.Location != nil {
		agent.Location.ForcedExpired = true
	}
	return nil
}

func normalizeObservedConnectionSource(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	address, err := netip.ParseAddr(trimmed)
	if err != nil {
		return "", fmt.Errorf("observed connection source is invalid: %w", err)
	}
	return address.Unmap().WithZone("").String(), nil
}

func (service *AgentControlLifecycleService) OnDisconnected(ctx context.Context, agentID int) error {
	if err := service.agentRepo.UpdateStatus(ctx, agentID, "offline"); err != nil {
		return err
	}
	if service.heartbeatCache != nil {
		if err := service.heartbeatCache.Delete(ctx, agentID); err != nil {
			pkg.Warn("Failed to clear heartbeat cache on disconnect", zap.Int("agent.id", agentID), zap.Error(err))
		}
	}
	return nil
}

// OnControlConnectionDetached clears the physical connection address without
// changing liveness: the session registry permits a short reconnect window.
func (service *AgentControlLifecycleService) OnControlConnectionDetached(ctx context.Context, agentID int) error {
	if agentID <= 0 {
		return fmt.Errorf("agent identity is required")
	}
	return service.agentRepo.ClearConnectionIP(ctx, agentID)
}

func (service *AgentControlLifecycleService) SendConfigUpdate(agent *agentdomain.Agent) {
	if service == nil || service.messageBus == nil || agent == nil {
		return
	}
	service.messageBus.SendConfigUpdate(agent.ID, BuildConfigUpdatePayload(agent))
}

func (service *AgentControlLifecycleService) RecordHeartbeat(ctx context.Context, agentID int, event agentdomain.AgentHeartbeatEvent) error {
	if agentID <= 0 {
		return nil
	}
	// Processing contract: persist heartbeat first, then update cache best-effort,
	// then evaluate whether upgrade notification should be sent.
	update := toAgentHeartbeatUpdate(service.clock.NowUTC(), event)

	if err := service.agentRepo.UpdateHeartbeat(ctx, agentID, update); err != nil {
		return err
	}

	if service.heartbeatCache != nil {
		cachePayload := toHeartbeatCacheData(event)
		if err := service.heartbeatCache.Set(ctx, agentID, cachePayload); err != nil {
			pkg.Warn("Failed to cache heartbeat", zap.Error(err))
		}
	}

	notifier := service.updateNotifier
	if notifier == nil {
		notifier = newUpdateNotifier(service.messageBus, service.desiredAgentVersion, service.agentImageRef)
		service.updateNotifier = notifier
	}
	notifier.maybeSendUpdateRequired(agentID, event.AgentVersion)
	return nil
}

func toAgentHeartbeatUpdate(now time.Time, event agentdomain.AgentHeartbeatEvent) agentdomain.AgentHeartbeatUpdate {
	update := agentdomain.AgentHeartbeatUpdate{
		LastHeartbeat:            now,
		InstanceID:               event.InstanceID,
		SessionID:                event.SessionID,
		SessionEpoch:             event.SessionEpoch,
		ObservedHostname:         event.ObservedHostname,
		AgentVersion:             event.AgentVersion,
		OperatingSystem:          event.OperatingSystem,
		Architecture:             event.Architecture,
		ContainerRuntimeReady:    event.ContainerRuntimeReady,
		SupportedEngineAPIMajors: append([]uint32(nil), event.SupportedEngineAPIMajors...),
		CPU:                      event.CPU,
		Mem:                      event.Mem,
		Disk:                     event.Disk,
		RunningTasks:             event.RunningTasks,
		TaskSlotsUsed:            event.TaskSlotsUsed,
		Uptime:                   event.Uptime,
	}

	if event.Health == nil {
		return update
	}

	update.HasHealth = true
	update.HealthState = event.Health.State
	update.HealthReason = event.Health.Reason
	update.HealthMessage = event.Health.Message
	if event.Health.Since != nil {
		since := event.Health.Since.UTC()
		update.HealthSince = &since
	}
	return update
}

func toHeartbeatCacheData(event agentdomain.AgentHeartbeatEvent) *cache.HeartbeatData {
	cachePayload := &cache.HeartbeatData{
		InstanceID:               event.InstanceID,
		SessionID:                event.SessionID,
		SessionEpoch:             event.SessionEpoch,
		ObservedHostname:         event.ObservedHostname,
		CPU:                      event.CPU,
		Mem:                      event.Mem,
		Disk:                     event.Disk,
		RunningTasks:             event.RunningTasks,
		TaskSlotsUsed:            event.TaskSlotsUsed,
		AgentVersion:             event.AgentVersion,
		OperatingSystem:          event.OperatingSystem,
		Architecture:             event.Architecture,
		ContainerRuntimeReady:    event.ContainerRuntimeReady,
		SupportedEngineAPIMajors: append([]uint32(nil), event.SupportedEngineAPIMajors...),
		Uptime:                   event.Uptime,
	}

	if event.Health == nil {
		return cachePayload
	}

	var since *time.Time
	if event.Health.Since != nil {
		value := event.Health.Since.UTC()
		since = &value
	}
	cachePayload.Health = &cache.HealthStatus{
		State:   event.Health.State,
		Reason:  event.Health.Reason,
		Message: event.Health.Message,
		Since:   since,
	}
	return cachePayload
}
