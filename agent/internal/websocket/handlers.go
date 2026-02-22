package websocket

import (
	"encoding/json"

	"github.com/yyhuni/lunafox/agent/internal/protocol"
)

// Handler routes incoming WebSocket messages.
type Handler struct {
	onTaskAvailable func()
	onTaskCancel    func(int)
	onConfigUpdate  func(protocol.ConfigUpdatePayload)
	onUpdateReq     func(protocol.UpdateRequiredPayload)
	onLogOpen       func(protocol.LogOpenPayload)
	onLogCancel     func(protocol.LogCancelPayload)
}

// NewHandler creates a message handler.
func NewHandler() *Handler {
	return &Handler{}
}

// OnTaskAvailable registers a callback for task_available messages.
func (h *Handler) OnTaskAvailable(fn func()) {
	h.onTaskAvailable = fn
}

// OnTaskCancel registers a callback for task_cancel messages.
func (h *Handler) OnTaskCancel(fn func(int)) {
	h.onTaskCancel = fn
}

// OnConfigUpdate registers a callback for config_update messages.
func (h *Handler) OnConfigUpdate(fn func(protocol.ConfigUpdatePayload)) {
	h.onConfigUpdate = fn
}

// OnUpdateRequired registers a callback for update_required messages.
func (h *Handler) OnUpdateRequired(fn func(protocol.UpdateRequiredPayload)) {
	h.onUpdateReq = fn
}

// OnLogOpen registers a callback for log_open messages.
func (h *Handler) OnLogOpen(fn func(protocol.LogOpenPayload)) {
	h.onLogOpen = fn
}

// OnLogCancel registers a callback for log_cancel messages.
func (h *Handler) OnLogCancel(fn func(protocol.LogCancelPayload)) {
	h.onLogCancel = fn
}

// Handle processes a raw message.
func (h *Handler) Handle(raw []byte) {
	var msg struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Type {
	case protocol.MessageTypeTaskAvailable:
		if h.onTaskAvailable != nil {
			h.onTaskAvailable()
		}
	case protocol.MessageTypeTaskCancel:
		if h.onTaskCancel == nil {
			return
		}
		var payload protocol.TaskCancelPayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		if payload.TaskID > 0 {
			h.onTaskCancel(payload.TaskID)
		}
	case protocol.MessageTypeConfigUpdate:
		if h.onConfigUpdate == nil {
			return
		}
		var payload protocol.ConfigUpdatePayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		h.onConfigUpdate(payload)
	case protocol.MessageTypeUpdateRequired:
		if h.onUpdateReq == nil {
			return
		}
		var payload protocol.UpdateRequiredPayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		if payload.Version == "" || payload.ImageRef == "" {
			return
		}
		h.onUpdateReq(payload)
	case protocol.MessageTypeLogOpen:
		if h.onLogOpen == nil {
			return
		}
		var payload protocol.LogOpenPayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		if payload.RequestID == "" || payload.Container == "" {
			return
		}
		h.onLogOpen(payload)
	case protocol.MessageTypeLogCancel:
		if h.onLogCancel == nil {
			return
		}
		var payload protocol.LogCancelPayload
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		if payload.RequestID == "" {
			return
		}
		h.onLogCancel(payload)
	}
}
