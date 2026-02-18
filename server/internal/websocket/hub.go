package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yyhuni/lunafox/server/internal/agentproto"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

// Client represents a WebSocket client connection
type Client struct {
	AgentID int
	Conn    *websocket.Conn
	Send    chan []byte
	Hub     *Hub
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients (agentID -> Client)
	clients map[int]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast message to all clients
	broadcast chan []byte

	// Send message to specific client
	sendTo chan *SendToMessage

	// Mutex for thread-safe access
	mu sync.RWMutex
}

// SendToMessage represents a message to send to a specific agent
type SendToMessage struct {
	AgentID int
	Message []byte
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		sendTo:     make(chan *SendToMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.AgentID] = client
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.AgentID]; ok {
				delete(h.clients, client.AgentID)
				close(client.Send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.AgentID)
				}
			}
			h.mu.Unlock()

		case msg := <-h.sendTo:
			h.mu.Lock()
			if client, ok := h.clients[msg.AgentID]; ok {
				select {
				case client.Send <- msg.Message:
				default:
					close(client.Send)
					delete(h.clients, msg.AgentID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// Register registers a new client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// SendTo sends a message to a specific agent
func (h *Hub) SendTo(agentID int, message []byte) {
	h.SendToWithResult(agentID, message)
}

// SendTaskCancel sends a task_cancel message to a specific agent.
func (h *Hub) SendTaskCancel(agentID, taskID int) {
	payload, err := json.Marshal(agentproto.TaskCancelPayload{TaskID: taskID})
	if err != nil {
		return
	}
	msg := agentproto.Message{
		Type:      agentproto.MessageTypeTaskCancel,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.SendTo(agentID, data)
}

// SendUpdateRequired sends an update_required message and returns whether sending succeeded.
func (h *Hub) SendUpdateRequired(agentID int, payload agentproto.UpdateRequiredPayload) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	msg := agentproto.Message{
		Type:      agentproto.MessageTypeUpdateRequired,
		Payload:   data,
		Timestamp: time.Now().UTC(),
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		return false
	}
	return h.SendToWithResult(agentID, encoded)
}

// SendConfigUpdate sends a config_update message to a specific agent.
func (h *Hub) SendConfigUpdate(agentID int, payload agentproto.ConfigUpdatePayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	msg := agentproto.Message{
		Type:      agentproto.MessageTypeConfigUpdate,
		Payload:   data,
		Timestamp: time.Now().UTC(),
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.SendTo(agentID, encoded)
}

// NewAgentMessagePublisher exposes a typed publisher adapter for agent runtime services.
func NewAgentMessagePublisher(hub *Hub) agentapp.AgentMessagePublisher {
	return &agentMessagePublisher{hub: hub}
}

type agentMessagePublisher struct {
	hub *Hub
}

func (publisher *agentMessagePublisher) SendConfigUpdate(agentID int, payload agentproto.ConfigUpdatePayload) {
	if publisher == nil || publisher.hub == nil {
		return
	}
	publisher.hub.SendConfigUpdate(agentID, payload)
}

func (publisher *agentMessagePublisher) SendUpdateRequired(agentID int, payload agentproto.UpdateRequiredPayload) bool {
	if publisher == nil || publisher.hub == nil {
		return false
	}
	return publisher.hub.SendUpdateRequired(agentID, payload)
}

func (publisher *agentMessagePublisher) SendTaskCancel(agentID, taskID int) {
	if publisher == nil || publisher.hub == nil {
		return
	}
	publisher.hub.SendTaskCancel(agentID, taskID)
}

// SendToWithResult sends a message to a specific agent and returns success.
func (h *Hub) SendToWithResult(agentID int, message []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.clients[agentID]
	if !ok {
		return false
	}
	select {
	case client.Send <- message:
		return true
	default:
		close(client.Send)
		delete(h.clients, agentID)
		return false
	}
}

// IsConnected checks if an agent is currently connected
func (h *Hub) IsConnected(agentID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[agentID]
	return ok
}

// GetConnectedAgents returns a list of connected agent IDs
func (h *Hub) GetConnectedAgents() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	agentIDs := make([]int, 0, len(h.clients))
	for agentID := range h.clients {
		agentIDs = append(agentIDs, agentID)
	}
	return agentIDs
}
