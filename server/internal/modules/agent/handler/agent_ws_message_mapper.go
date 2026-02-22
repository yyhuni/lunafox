package handler

import (
	"encoding/json"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentproto"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

func toRuntimeMessageInput(raw []byte) (agentapp.RuntimeMessageInput, error) {
	var message agentproto.Message
	if err := json.Unmarshal(raw, &message); err != nil {
		return agentapp.RuntimeMessageInput{}, err
	}

	input := agentapp.RuntimeMessageInput{Type: message.Type}
	switch message.Type {
	case agentproto.MessageTypeHeartbeat:
		var payload agentproto.HeartbeatPayload
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return agentapp.RuntimeMessageInput{}, err
		}
		input.Heartbeat = toHeartbeatItem(payload)
	case agentproto.MessageTypeLogStarted:
		var payload agentproto.LogStartedPayload
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return agentapp.RuntimeMessageInput{}, err
		}
		input.LogStarted = &agentapp.LogStartedItem{RequestID: payload.RequestID}
	case agentproto.MessageTypeLogChunk:
		var payload agentproto.LogChunkPayload
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return agentapp.RuntimeMessageInput{}, err
		}
		input.LogChunk = &agentapp.LogChunkItem{
			RequestID: payload.RequestID,
			TS:        payload.TS.UTC(),
			Stream:    payload.Stream,
			Line:      payload.Line,
			Truncated: payload.Truncated,
		}
	case agentproto.MessageTypeLogEnd:
		var payload agentproto.LogEndPayload
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return agentapp.RuntimeMessageInput{}, err
		}
		input.LogEnd = &agentapp.LogEndItem{
			RequestID: payload.RequestID,
			Reason:    payload.Reason,
		}
	case agentproto.MessageTypeLogError:
		var payload agentproto.LogErrorPayload
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			return agentapp.RuntimeMessageInput{}, err
		}
		input.LogError = &agentapp.LogErrorItem{
			RequestID: payload.RequestID,
			Code:      payload.Code,
			Message:   payload.Message,
		}
	}
	return input, nil
}

func toHeartbeatItem(payload agentproto.HeartbeatPayload) *agentapp.HeartbeatItem {
	item := &agentapp.HeartbeatItem{
		CPU:      payload.CPU,
		Mem:      payload.Mem,
		Disk:     payload.Disk,
		Tasks:    payload.Tasks,
		Version:  payload.Version,
		Hostname: payload.Hostname,
		Uptime:   payload.Uptime,
	}

	if payload.Health == nil {
		return item
	}
	var since *time.Time
	if payload.Health.Since != nil {
		value := payload.Health.Since.UTC()
		since = &value
	}
	item.Health = &agentapp.HeartbeatHealthItem{
		State:   payload.Health.State,
		Reason:  payload.Health.Reason,
		Message: payload.Health.Message,
		Since:   since,
	}
	return item
}
