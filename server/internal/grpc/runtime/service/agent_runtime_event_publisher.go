package service

import (
	"sync"

	runtimev1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/runtime/v1"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// defaultAgentStreamBufferSize is the per-connection outbound queue size.
// When full, Publish drops events instead of blocking runtime goroutines.
const defaultAgentStreamBufferSize = 16

type AgentStreamRegistry struct {
	mu      sync.RWMutex
	streams map[int]map[uint64]chan *runtimev1.AgentRuntimeEvent
	nextID  uint64
}

func NewAgentStreamRegistry() *AgentStreamRegistry {
	return &AgentStreamRegistry{
		streams: make(map[int]map[uint64]chan *runtimev1.AgentRuntimeEvent),
	}
}

// Register creates an outbound event queue for one connected stream of an agent.
// Caller must invoke unregister when the stream lifecycle ends.
func (registry *AgentStreamRegistry) Register(agentID int) (<-chan *runtimev1.AgentRuntimeEvent, func()) {
	channel := make(chan *runtimev1.AgentRuntimeEvent, defaultAgentStreamBufferSize)
	if registry == nil || agentID <= 0 {
		return channel, func() {}
	}

	registry.mu.Lock()
	registry.nextID++
	streamID := registry.nextID
	agentStreams := registry.streams[agentID]
	if agentStreams == nil {
		agentStreams = make(map[uint64]chan *runtimev1.AgentRuntimeEvent)
		registry.streams[agentID] = agentStreams
	}
	agentStreams[streamID] = channel
	registry.mu.Unlock()

	var once sync.Once
	unregister := func() {
		once.Do(func() {
			registry.mu.Lock()
			defer registry.mu.Unlock()

			agentStreams := registry.streams[agentID]
			if agentStreams == nil {
				return
			}
			delete(agentStreams, streamID)
			if len(agentStreams) == 0 {
				delete(registry.streams, agentID)
			}
		})
	}
	return channel, unregister
}

// Publish fans out one runtime event to all active streams of an agent.
// Delivery is best-effort and non-blocking: a full channel drops the event.
// It returns true if at least one stream accepted the event.
func (registry *AgentStreamRegistry) Publish(agentID int, event *runtimev1.AgentRuntimeEvent) bool {
	if registry == nil || agentID <= 0 || event == nil {
		return false
	}

	registry.mu.RLock()
	agentStreams := registry.streams[agentID]
	if len(agentStreams) == 0 {
		registry.mu.RUnlock()
		return false
	}
	targets := make([]chan *runtimev1.AgentRuntimeEvent, 0, len(agentStreams))
	for _, channel := range agentStreams {
		targets = append(targets, channel)
	}
	registry.mu.RUnlock()

	delivered := false
	for _, channel := range targets {
		select {
		case channel <- event:
			delivered = true
		default:
			// Intentionally drop on backpressure to keep publish path non-blocking.
		}
	}
	return delivered
}

type AgentRuntimeEventPublisher struct {
	registry *AgentStreamRegistry
}

func NewAgentRuntimeEventPublisher(registry *AgentStreamRegistry) *AgentRuntimeEventPublisher {
	return &AgentRuntimeEventPublisher{registry: registry}
}

// SendConfigUpdate publishes a config_update event as best-effort downlink.
func (publisher *AgentRuntimeEventPublisher) SendConfigUpdate(agentID int, payload agentdomain.ConfigUpdatePayload) {
	if publisher == nil || publisher.registry == nil {
		return
	}
	publisher.registry.Publish(agentID, &runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_ConfigUpdate{
			ConfigUpdate: toRuntimeConfigUpdate(payload),
		},
	})
}

// SendUpdateRequired publishes an update_required event and reports whether at
// least one stream accepted it.
func (publisher *AgentRuntimeEventPublisher) SendUpdateRequired(agentID int, payload agentdomain.UpdateRequiredPayload) bool {
	if publisher == nil || publisher.registry == nil {
		return false
	}
	return publisher.registry.Publish(agentID, &runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_UpdateRequired{
			UpdateRequired: &runtimev1.UpdateRequired{
				AgentVersion:   payload.AgentVersion,
				AgentImageRef:  payload.AgentImageRef,
				WorkerImageRef: payload.WorkerImageRef,
				WorkerVersion:  payload.WorkerVersion,
			},
		},
	})
}

// SendTaskCancel publishes a task_cancel event as best-effort downlink.
func (publisher *AgentRuntimeEventPublisher) SendTaskCancel(agentID, taskID int) {
	if publisher == nil || publisher.registry == nil {
		return
	}
	publisher.registry.Publish(agentID, &runtimev1.AgentRuntimeEvent{
		Payload: &runtimev1.AgentRuntimeEvent_TaskCancel{
			TaskCancel: &runtimev1.TaskCancel{
				TaskId: int32(taskID),
			},
		},
	})
}

func toRuntimeConfigUpdate(payload agentdomain.ConfigUpdatePayload) *runtimev1.ConfigUpdate {
	return &runtimev1.ConfigUpdate{
		MaxTasks:      intPtrToInt32Ptr(payload.MaxTasks),
		CpuThreshold:  intPtrToInt32Ptr(payload.CPUThreshold),
		MemThreshold:  intPtrToInt32Ptr(payload.MemThreshold),
		DiskThreshold: intPtrToInt32Ptr(payload.DiskThreshold),
	}
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	casted := int32(*value)
	return &casted
}
