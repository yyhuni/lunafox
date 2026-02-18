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
	if message.Type != agentproto.MessageTypeHeartbeat {
		return input, nil
	}

	var payload agentproto.HeartbeatPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return agentapp.RuntimeMessageInput{}, err
	}
	input.Heartbeat = toHeartbeatItem(payload)
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
