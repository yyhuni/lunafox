package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/agentproto"
)

const (
	LogStreamEventStatus = "status"
	LogStreamEventLog    = "log"
	LogStreamEventError  = "error"
	LogStreamEventDone   = "done"
	LogStreamEventPing   = "ping"

	LogStreamErrorAgentTimeout = "agent_timeout"
	LogStreamErrorWSSendFailed = "ws_send_failed"

	defaultLogSessionBuffer = 1024
	defaultLogPingInterval  = 20 * time.Second
	defaultLogSweepInterval = 10 * time.Second
	defaultLogStartTimeout  = 15 * time.Second
	defaultLogMaxSessionAge = 6 * time.Hour
)

var (
	ErrLogStreamSendFailed = errors.New("log stream ws send failed")
)

// LogStreamOpenRequest describes one stream open request from HTTP layer.
type LogStreamOpenRequest struct {
	AgentID    int
	Container  string
	Tail       int
	Follow     bool
	Since      *time.Time
	Timestamps bool
}

// LogStreamSubscription is returned after opening a stream session.
type LogStreamSubscription struct {
	RequestID string
	Events    <-chan LogStreamEvent
}

// LogStreamEvent is the normalized internal stream event consumed by SSE handler.
type LogStreamEvent struct {
	Type      string
	RequestID string
	Status    string
	TS        time.Time
	Stream    string
	Line      string
	Truncated bool
	Code      string
	Message   string
	Reason    string
}

type logStreamSession struct {
	requestID   string
	agentID     int
	events      chan LogStreamEvent
	openedAt    time.Time
	lastEventAt time.Time
	lastPingAt  time.Time
	started     bool
	closed      bool
}

// LogStreamService manages transient log stream sessions bridging HTTP SSE and agent WS.
type LogStreamService struct {
	messageBus AgentMessagePublisher
	clock      Clock

	mu       sync.Mutex
	sessions map[string]*logStreamSession

	pingInterval  time.Duration
	sweepInterval time.Duration
	startTimeout  time.Duration
	maxSessionAge time.Duration

	stopCh chan struct{}
}

func NewLogStreamService(messageBus AgentMessagePublisher, clock Clock) *LogStreamService {
	if clock == nil {
		panic("clock is required")
	}
	service := &LogStreamService{
		messageBus:    messageBus,
		clock:         clock,
		sessions:      make(map[string]*logStreamSession),
		pingInterval:  defaultLogPingInterval,
		sweepInterval: defaultLogSweepInterval,
		startTimeout:  defaultLogStartTimeout,
		maxSessionAge: defaultLogMaxSessionAge,
		stopCh:        make(chan struct{}),
	}
	go service.runSweepLoop()
	return service
}

func (service *LogStreamService) Open(ctx context.Context, req LogStreamOpenRequest) (*LogStreamSubscription, error) {
	if service == nil {
		return nil, errors.New("log stream service is nil")
	}
	if req.AgentID <= 0 || strings.TrimSpace(req.Container) == "" {
		return nil, errors.New("invalid log stream request")
	}

	now := service.clock.NowUTC()
	requestID := uuid.NewString()
	session := &logStreamSession{
		requestID:   requestID,
		agentID:     req.AgentID,
		events:      make(chan LogStreamEvent, defaultLogSessionBuffer),
		openedAt:    now,
		lastEventAt: now,
		lastPingAt:  now,
	}

	service.mu.Lock()
	service.sessions[requestID] = session
	service.mu.Unlock()

	payload := agentproto.LogOpenPayload{
		RequestID:  requestID,
		Container:  req.Container,
		Tail:       req.Tail,
		Follow:     req.Follow,
		Since:      req.Since,
		Timestamps: req.Timestamps,
	}
	if service.messageBus == nil || !service.messageBus.SendLogOpen(req.AgentID, payload) {
		service.closeSession(requestID, false, nil)
		return nil, ErrLogStreamSendFailed
	}

	if ctx != nil {
		go func(id string, done <-chan struct{}) {
			select {
			case <-done:
				service.Cancel(id, "client_closed")
			case <-service.stopCh:
			}
		}(requestID, ctx.Done())
	}

	return &LogStreamSubscription{
		RequestID: requestID,
		Events:    session.events,
	}, nil
}

func (service *LogStreamService) Cancel(requestID, reason string) {
	if service == nil || strings.TrimSpace(requestID) == "" {
		return
	}
	if strings.TrimSpace(reason) == "" {
		reason = "cancelled"
	}
	done := LogStreamEvent{Type: LogStreamEventDone, RequestID: requestID, Reason: reason}
	service.closeSession(requestID, true, []LogStreamEvent{done})
}

func (service *LogStreamService) HandleLogStarted(payload LogStartedItem) {
	if service == nil || strings.TrimSpace(payload.RequestID) == "" {
		return
	}
	now := service.clock.NowUTC()
	service.mu.Lock()
	defer service.mu.Unlock()
	session := service.sessions[payload.RequestID]
	if session == nil || session.closed {
		return
	}
	session.started = true
	session.lastEventAt = now
	service.tryEmitLocked(session, LogStreamEvent{
		Type:      LogStreamEventStatus,
		RequestID: payload.RequestID,
		Status:    "started",
	})
}

func (service *LogStreamService) HandleLogChunk(payload LogChunkItem) {
	if service == nil || strings.TrimSpace(payload.RequestID) == "" {
		return
	}
	now := service.clock.NowUTC()
	ts := payload.TS.UTC()
	if ts.IsZero() {
		ts = now
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	session := service.sessions[payload.RequestID]
	if session == nil || session.closed {
		return
	}
	session.lastEventAt = now
	service.tryEmitLocked(session, LogStreamEvent{
		Type:      LogStreamEventLog,
		RequestID: payload.RequestID,
		TS:        ts,
		Stream:    payload.Stream,
		Line:      payload.Line,
		Truncated: payload.Truncated,
	})
}

func (service *LogStreamService) HandleLogEnd(payload LogEndItem) {
	if service == nil || strings.TrimSpace(payload.RequestID) == "" {
		return
	}
	done := LogStreamEvent{Type: LogStreamEventDone, RequestID: payload.RequestID, Reason: payload.Reason}
	service.closeSession(payload.RequestID, false, []LogStreamEvent{done})
}

func (service *LogStreamService) HandleLogError(payload LogErrorItem) {
	if service == nil || strings.TrimSpace(payload.RequestID) == "" {
		return
	}
	code := strings.TrimSpace(payload.Code)
	if code == "" {
		code = "internal_error"
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = "log stream error"
	}
	errorEvent := LogStreamEvent{
		Type:      LogStreamEventError,
		RequestID: payload.RequestID,
		Code:      code,
		Message:   message,
	}
	done := LogStreamEvent{Type: LogStreamEventDone, RequestID: payload.RequestID, Reason: "error"}
	service.closeSession(payload.RequestID, false, []LogStreamEvent{errorEvent, done})
}

func (service *LogStreamService) Close() {
	if service == nil {
		return
	}
	select {
	case <-service.stopCh:
		return
	default:
		close(service.stopCh)
	}
}

func (service *LogStreamService) runSweepLoop() {
	ticker := time.NewTicker(service.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-service.stopCh:
			return
		case <-ticker.C:
			service.sweepSessions()
		}
	}
}

func (service *LogStreamService) sweepSessions() {
	now := service.clock.NowUTC()
	var cleanupIDs []string

	service.mu.Lock()
	for requestID, session := range service.sessions {
		if session == nil || session.closed {
			cleanupIDs = append(cleanupIDs, requestID)
			continue
		}

		if now.Sub(session.lastPingAt) >= service.pingInterval {
			service.tryEmitLocked(session, LogStreamEvent{
				Type:      LogStreamEventPing,
				RequestID: requestID,
				TS:        now,
			})
			session.lastPingAt = now
		}

		if !session.started && now.Sub(session.openedAt) >= service.startTimeout {
			cleanupIDs = append(cleanupIDs, requestID)
			continue
		}
		if now.Sub(session.openedAt) >= service.maxSessionAge {
			cleanupIDs = append(cleanupIDs, requestID)
		}
	}
	service.mu.Unlock()

	for _, requestID := range cleanupIDs {
		errorEvent := LogStreamEvent{
			Type:      LogStreamEventError,
			RequestID: requestID,
			Code:      LogStreamErrorAgentTimeout,
			Message:   "log stream startup timeout",
		}
		done := LogStreamEvent{Type: LogStreamEventDone, RequestID: requestID, Reason: "timeout"}
		service.closeSession(requestID, true, []LogStreamEvent{errorEvent, done})
	}
}

func (service *LogStreamService) closeSession(requestID string, sendCancel bool, events []LogStreamEvent) {
	if strings.TrimSpace(requestID) == "" {
		return
	}

	var agentID int
	service.mu.Lock()
	session := service.sessions[requestID]
	if session == nil || session.closed {
		service.mu.Unlock()
		return
	}
	delete(service.sessions, requestID)
	agentID = session.agentID
	for _, event := range events {
		service.tryEmitLocked(session, event)
	}
	session.closed = true
	close(session.events)
	service.mu.Unlock()

	if sendCancel && service.messageBus != nil && agentID > 0 {
		service.messageBus.SendLogCancel(agentID, agentproto.LogCancelPayload{RequestID: requestID})
	}
}

func (service *LogStreamService) tryEmitLocked(session *logStreamSession, event LogStreamEvent) bool {
	if session == nil || session.closed {
		return false
	}
	select {
	case session.events <- event:
		return true
	default:
		// Slow consumers should not block agent websocket ingestion.
		return false
	}
}
