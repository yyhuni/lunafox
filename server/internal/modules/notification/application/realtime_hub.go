package application

import "sync"

// RealtimeHub is an in-memory, best-effort invalidation broker. Durable inbox
// queries remain the only notification source of truth, so it stores no event
// history, cursors, payloads, or replay state.
type RealtimeHub struct {
	mu          sync.RWMutex
	subscribers map[int]map[uint64]chan struct{}
	nextID      uint64
}

// NewRealtimeHub creates a broker for post-commit refresh hints.
func NewRealtimeHub() *RealtimeHub {
	return &RealtimeHub{subscribers: make(map[int]map[uint64]chan struct{})}
}

// Subscribe returns a lightweight user-scoped change channel and an idempotent
// remover. The channel is deliberately not closed to avoid a send/close race
// with concurrent materializer notifications.
func (hub *RealtimeHub) Subscribe(userID int) (<-chan struct{}, func()) {
	if userID <= 0 {
		return nil, func() {}
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.nextID++
	id := hub.nextID
	updates := make(chan struct{}, 1)
	if hub.subscribers[userID] == nil {
		hub.subscribers[userID] = make(map[uint64]chan struct{})
	}
	hub.subscribers[userID][id] = updates
	var once sync.Once
	return updates, func() {
		once.Do(func() {
			hub.mu.Lock()
			defer hub.mu.Unlock()
			delete(hub.subscribers[userID], id)
			if len(hub.subscribers[userID]) == 0 {
				delete(hub.subscribers, userID)
			}
		})
	}
}

// NotifyUser coalesces refresh hints per stream and never blocks database work.
func (hub *RealtimeHub) NotifyUser(userID int) {
	if hub == nil || userID <= 0 {
		return
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, updates := range hub.subscribers[userID] {
		select {
		case updates <- struct{}{}:
		default:
		}
	}
}

var _ RealtimeNotifier = (*RealtimeHub)(nil)
