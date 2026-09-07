package agentcontrol

import (
	"sync"

	agentcontrolv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/control/v1"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// defaultAgentStreamBufferSize is the per-connection outbound queue size.
// When full, Publish drops events instead of blocking runtime goroutines.
const defaultAgentStreamBufferSize = 16

type AgentStreamRegistry struct {
	mu                  sync.RWMutex
	streams             map[int]map[uint64]chan *agentcontrolv1.ConnectResponse
	activeStreamByAgent map[int]uint64
	nextID              uint64
}

func NewAgentStreamRegistry() *AgentStreamRegistry {
	return &AgentStreamRegistry{
		streams:             make(map[int]map[uint64]chan *agentcontrolv1.ConnectResponse),
		activeStreamByAgent: make(map[int]uint64),
	}
}

// RegisterStream creates an outbound event queue for one connected stream of an agent.
// Caller must invoke unregister when the stream lifecycle ends.
func (registry *AgentStreamRegistry) RegisterStream(agentID int) (uint64, <-chan *agentcontrolv1.ConnectResponse, func()) {
	channel := make(chan *agentcontrolv1.ConnectResponse, defaultAgentStreamBufferSize)
	if registry == nil || agentID <= 0 {
		return 0, channel, func() {}
	}

	registry.mu.Lock()
	registry.nextID++
	streamID := registry.nextID
	agentStreams := registry.streams[agentID]
	if agentStreams == nil {
		agentStreams = make(map[uint64]chan *agentcontrolv1.ConnectResponse)
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
			if registry.activeStreamByAgent[agentID] == streamID {
				delete(registry.activeStreamByAgent, agentID)
			}
			if len(agentStreams) == 0 {
				delete(registry.streams, agentID)
			}
		})
	}
	return streamID, channel, unregister
}

func (registry *AgentStreamRegistry) SetActiveStream(agentID int, streamID uint64) {
	if registry == nil || agentID <= 0 || streamID == 0 {
		return
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if streams := registry.streams[agentID]; len(streams) > 0 {
		if _, ok := streams[streamID]; ok {
			registry.activeStreamByAgent[agentID] = streamID
		}
	}
}

func (registry *AgentStreamRegistry) ClearActiveStream(agentID int, streamID uint64) {
	if registry == nil || agentID <= 0 || streamID == 0 {
		return
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.activeStreamByAgent[agentID] == streamID {
		delete(registry.activeStreamByAgent, agentID)
	}
}

// Publish delivers one control-plane event to the active stream of an agent.
// Delivery is best-effort and non-blocking: a full channel drops the event.
// It returns true if the active stream accepted the event.
func (registry *AgentStreamRegistry) Publish(agentID int, event *agentcontrolv1.ConnectResponse) bool {
	if registry == nil || agentID <= 0 || event == nil {
		return false
	}

	registry.mu.RLock()
	agentStreams := registry.streams[agentID]
	activeStreamID := registry.activeStreamByAgent[agentID]
	if len(agentStreams) == 0 || activeStreamID == 0 {
		registry.mu.RUnlock()
		return false
	}
	channel, ok := agentStreams[activeStreamID]
	registry.mu.RUnlock()
	if !ok {
		return false
	}

	select {
	case channel <- event:
		return true
	default:
		// Intentionally drop on backpressure to keep publish path non-blocking.
		return false
	}
}

type AgentControlEventPublisher struct {
	registry *AgentStreamRegistry
}

func NewAgentControlEventPublisher(registry *AgentStreamRegistry) *AgentControlEventPublisher {
	return &AgentControlEventPublisher{registry: registry}
}

// SendConfigUpdate publishes agent control-plane scheduling/resource threshold changes for
// the active agent connection as best-effort downlink. It refreshes execution
// config and does not imply an Agent image upgrade.
func (publisher *AgentControlEventPublisher) SendConfigUpdate(agentID int, payload agentdomain.ConfigUpdatePayload) {
	if publisher == nil || publisher.registry == nil {
		return
	}
	publisher.registry.Publish(agentID, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_ConfigUpdate{
			ConfigUpdate: toAgentControlConfigUpdate(payload),
		},
	})
}

// SendUpdateRequired publishes the target Agent version and image ref to the
// active Agent connection as best-effort downlink. It signals that the current
// Agent control plane no longer matches the desired release and should upgrade.
// The return value reports whether the active stream accepted the notification.
func (publisher *AgentControlEventPublisher) SendUpdateRequired(agentID int, payload agentdomain.UpdateRequiredPayload) bool {
	if publisher == nil || publisher.registry == nil {
		return false
	}
	return publisher.registry.Publish(agentID, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_UpdateRequired{
			UpdateRequired: &agentcontrolv1.UpdateRequired{
				AgentVersion: payload.AgentVersion,
				AgentImage:   payload.AgentImageRef,
			},
		},
	})
}

// SendTaskCancel publishes a task_cancel event as best-effort downlink.
func (publisher *AgentControlEventPublisher) SendTaskCancel(agentID, scanID, taskID int) {
	_ = publisher.TrySendTaskCancel(agentID, scanID, taskID)
}

// TrySendTaskCancel publishes a task_cancel event as best-effort downlink and
// reports whether the currently active stream accepted it. Callers must treat
// false as an observation only: cancellation state is persisted before this
// process-local notification and is never rolled back for delivery.
func (publisher *AgentControlEventPublisher) TrySendTaskCancel(agentID, scanID, taskID int) bool {
	if publisher == nil || publisher.registry == nil {
		return false
	}
	return publisher.registry.Publish(agentID, &agentcontrolv1.ConnectResponse{
		Payload: &agentcontrolv1.ConnectResponse_TaskCancel{
			TaskCancel: &agentcontrolv1.TaskCancel{
				Task: resourcenames.Task(scanID, taskID),
			},
		},
	})
}

func toAgentControlConfigUpdate(payload agentdomain.ConfigUpdatePayload) *agentcontrolv1.ConfigUpdate {
	return &agentcontrolv1.ConfigUpdate{
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
