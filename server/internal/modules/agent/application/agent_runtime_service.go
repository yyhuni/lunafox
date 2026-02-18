package application

import (
	"context"
	"time"

	"github.com/yyhuni/lunafox/server/internal/cache"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.uber.org/zap"
)

// AgentRuntimeService orchestrates WebSocket runtime events.
type AgentRuntimeService struct {
	agentRepo       agentdomain.AgentRepository
	heartbeatCache  HeartbeatCachePort
	messageBus      AgentMessagePublisher
	clock           Clock
	serverVersion   string
	agentImageRef   string
	updateNotifier  *updateNotifier
	messageHandlers map[string]runtimeMessageHandler
}

func NewAgentRuntimeService(
	agentRepo agentdomain.AgentRepository,
	heartbeatCache HeartbeatCachePort,
	messageBus AgentMessagePublisher,
	clock Clock,
	serverVersion, agentImageRef string,
) *AgentRuntimeService {
	if clock == nil {
		panic("clock is required")
	}

	return &AgentRuntimeService{
		agentRepo:       agentRepo,
		heartbeatCache:  heartbeatCache,
		messageBus:      messageBus,
		clock:           clock,
		serverVersion:   serverVersion,
		agentImageRef:   agentImageRef,
		updateNotifier:  newUpdateNotifier(messageBus, serverVersion, agentImageRef),
		messageHandlers: newRuntimeMessageDispatcher(),
	}
}

func (service *AgentRuntimeService) OnConnected(ctx context.Context, agent *agentdomain.Agent, ipAddress string) error {
	now := service.clock.NowUTC()
	agent.Status = "online"
	agent.ConnectedAt = &now
	agent.LastHeartbeat = &now
	agent.IPAddress = ipAddress

	if err := service.agentRepo.Update(ctx, agent); err != nil {
		return err
	}

	service.SendConfigUpdate(agent)
	return nil
}

func (service *AgentRuntimeService) OnDisconnected(ctx context.Context, agentID int) error {
	if err := service.agentRepo.UpdateStatus(ctx, agentID, "offline"); err != nil {
		return err
	}
	if service.heartbeatCache != nil {
		if err := service.heartbeatCache.Delete(ctx, agentID); err != nil {
			pkg.Warn("Failed to clear heartbeat cache on disconnect", zap.Int("agent_id", agentID), zap.Error(err))
		}
	}
	return nil
}

func (service *AgentRuntimeService) SendConfigUpdate(agent *agentdomain.Agent) {
	if service == nil || service.messageBus == nil || agent == nil {
		return
	}
	service.messageBus.SendConfigUpdate(agent.ID, BuildConfigUpdatePayload(agent))
}

func (service *AgentRuntimeService) HandleMessage(ctx context.Context, agentID int, message RuntimeMessageInput) error {
	if agentID <= 0 {
		return nil
	}

	handlers := service.messageHandlers
	if handlers == nil {
		// Keep a safe fallback for partially-initialized instances used in tests.
		handlers = newRuntimeMessageDispatcher()
		service.messageHandlers = handlers
	}

	handler, ok := handlers[message.Type]
	if !ok || handler == nil {
		// Unknown message types are intentionally ignored for forward compatibility.
		return nil
	}
	return handler(service, ctx, agentID, message)
}

func (service *AgentRuntimeService) handleHeartbeat(ctx context.Context, agentID int, payload HeartbeatItem) error {
	// Processing contract: persist heartbeat first, then update cache best-effort,
	// then evaluate whether upgrade notification should be sent.
	update := toAgentHeartbeatUpdate(service.clock.NowUTC(), payload)

	if err := service.agentRepo.UpdateHeartbeat(ctx, agentID, update); err != nil {
		return err
	}

	if service.heartbeatCache != nil {
		cachePayload := toHeartbeatCacheData(payload)
		if err := service.heartbeatCache.Set(ctx, agentID, cachePayload); err != nil {
			pkg.Warn("Failed to cache heartbeat", zap.Error(err))
		}
	}

	notifier := service.updateNotifier
	if notifier == nil {
		notifier = newUpdateNotifier(service.messageBus, service.serverVersion, service.agentImageRef)
		service.updateNotifier = notifier
	}
	notifier.maybeSendUpdateRequired(agentID, payload.Version)
	return nil
}

func toAgentHeartbeatUpdate(now time.Time, payload HeartbeatItem) agentdomain.AgentHeartbeatUpdate {
	update := agentdomain.AgentHeartbeatUpdate{
		LastHeartbeat: now,
		Version:       payload.Version,
		Hostname:      payload.Hostname,
	}

	if payload.Health == nil {
		return update
	}

	update.HasHealth = true
	update.HealthState = payload.Health.State
	update.HealthReason = payload.Health.Reason
	update.HealthMessage = payload.Health.Message
	if payload.Health.Since != nil {
		since := payload.Health.Since.UTC()
		update.HealthSince = &since
	}
	return update
}

func toHeartbeatCacheData(payload HeartbeatItem) *cache.HeartbeatData {
	cachePayload := &cache.HeartbeatData{
		CPU:      payload.CPU,
		Mem:      payload.Mem,
		Disk:     payload.Disk,
		Tasks:    payload.Tasks,
		Version:  payload.Version,
		Hostname: payload.Hostname,
		Uptime:   payload.Uptime,
	}

	if payload.Health == nil {
		return cachePayload
	}

	var since *time.Time
	if payload.Health.Since != nil {
		value := payload.Health.Since.UTC()
		since = &value
	}
	cachePayload.Health = &cache.HealthStatus{
		State:   payload.Health.State,
		Reason:  payload.Health.Reason,
		Message: payload.Health.Message,
		Since:   since,
	}
	return cachePayload
}
