package agentcontrol

import (
	"strings"
	"sync"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

// DefaultSessionHeartbeatTimeout defines how long a detached logical session
// may be recovered before the server treats it as expired and allocates a new
// fencing epoch. It is intentionally shared with the agent monitor timeout.
const DefaultSessionHeartbeatTimeout = 120 * time.Second

// DefaultExecutionSnapshotFreshness is independent from the longer session
// recovery window. Scheduler admission must stop promptly when node readiness
// is no longer being observed.
const DefaultExecutionSnapshotFreshness = 15 * time.Second

type executionSnapshotAdmissionReason string

const (
	executionSnapshotAdmissionInvalidSession      executionSnapshotAdmissionReason = "session_unavailable"
	executionSnapshotAdmissionMissing             executionSnapshotAdmissionReason = "snapshot_missing"
	executionSnapshotAdmissionRuntimeNotReady     executionSnapshotAdmissionReason = "container_runtime_not_ready"
	executionSnapshotAdmissionMissingAPISupport   executionSnapshotAdmissionReason = "engine_api_support_missing"
	executionSnapshotAdmissionUnsupportedPlatform executionSnapshotAdmissionReason = "unsupported_daemon_platform"
	executionSnapshotAdmissionStale               executionSnapshotAdmissionReason = "snapshot_stale"
)

type ControlSessionPhase string

const (
	ControlSessionPhasePendingReady        ControlSessionPhase = "pending_ready"
	ControlSessionPhaseReadyAttached       ControlSessionPhase = "ready_attached"
	ControlSessionPhaseDetachedRecoverable ControlSessionPhase = "detached_recoverable"
)

// ActiveControlSession records the control-plane session currently fenced as active
// for an agent on the server side.
// SessionID is the agent-generated process session identifier, while
// SessionEpoch is the server-issued fencing token for the active connection.
type ActiveControlSession struct {
	AgentID   int
	SessionID string
	// SessionEpoch is incremented by the server each time a new active runtime
	// session replaces the previous one for the same agent.
	SessionEpoch      int64
	StreamID          uint64
	LastSeenAt        time.Time
	Phase             ControlSessionPhase
	ExecutionSnapshot *agentdomain.AgentExecutionCapabilitySnapshot
}

func (session ActiveControlSession) IsRegistered() bool {
	return session.AgentID > 0 && session.SessionID != "" && session.SessionEpoch > 0
}

func (session ActiveControlSession) IsDownlinkEligible() bool {
	return session.Phase == ControlSessionPhaseReadyAttached && session.StreamID != 0
}

func (session ActiveControlSession) CanAcceptOperationalMessages() bool {
	return session.Phase == ControlSessionPhaseReadyAttached && session.StreamID != 0
}

type ActiveSessionRegistry struct {
	mu                         sync.RWMutex
	sessions                   map[int]ActiveControlSession
	lastEpoch                  map[int]int64
	heartbeatTimeout           time.Duration
	executionSnapshotFreshness time.Duration
	now                        func() time.Time
}

func NewActiveSessionRegistry() *ActiveSessionRegistry {
	return newActiveSessionRegistry(DefaultSessionHeartbeatTimeout, time.Now)
}

func newActiveSessionRegistry(timeout time.Duration, now func() time.Time) *ActiveSessionRegistry {
	if timeout <= 0 {
		timeout = DefaultSessionHeartbeatTimeout
	}
	if now == nil {
		now = time.Now
	}
	return &ActiveSessionRegistry{
		sessions:                   make(map[int]ActiveControlSession),
		lastEpoch:                  make(map[int]int64),
		heartbeatTimeout:           timeout,
		executionSnapshotFreshness: DefaultExecutionSnapshotFreshness,
		now:                        now,
	}
}

// UpdateExecutionSnapshotIfCurrent attaches a server-observed capability
// snapshot only to the authenticated fenced session that supplied it.
func (registry *ActiveSessionRegistry) UpdateExecutionSnapshotIfCurrent(agentID int, sessionID string, sessionEpoch int64, streamID uint64, snapshot agentdomain.AgentExecutionCapabilitySnapshot) bool {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 || streamID == 0 {
		return false
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	current, ok := registry.sessions[agentID]
	if !ok || current.SessionID != sessionID || current.SessionEpoch != sessionEpoch || current.StreamID != streamID {
		return false
	}
	copy := snapshot.Clone()
	copy.ObservedAt = registry.now().UTC()
	current.ExecutionSnapshot = &copy
	registry.sessions[agentID] = current
	return true
}

// CurrentExecutionSnapshotIfCurrent returns a detached, fresh snapshot for
// this exact ready session. Missing, empty, or stale snapshots fail closed.
func (registry *ActiveSessionRegistry) CurrentExecutionSnapshotIfCurrent(agentID int, sessionID string, sessionEpoch int64, streamID uint64) (agentdomain.AgentExecutionCapabilitySnapshot, bool) {
	snapshot, reason := registry.currentExecutionSnapshotIfCurrent(agentID, sessionID, sessionEpoch, streamID)
	return snapshot, reason == ""
}

func (registry *ActiveSessionRegistry) currentExecutionSnapshotIfCurrent(agentID int, sessionID string, sessionEpoch int64, streamID uint64) (agentdomain.AgentExecutionCapabilitySnapshot, executionSnapshotAdmissionReason) {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 || streamID == 0 {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionInvalidSession
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	now := registry.now().UTC()
	current, ok := registry.sessions[agentID]
	if !ok || !current.CanAcceptOperationalMessages() || registry.isExpiredLocked(current, now) ||
		current.SessionID != sessionID || current.SessionEpoch != sessionEpoch || current.StreamID != streamID {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionInvalidSession
	}
	if current.ExecutionSnapshot == nil || current.ExecutionSnapshot.ObservedAt.IsZero() {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionMissing
	}
	if !current.ExecutionSnapshot.ContainerRuntimeReady {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionRuntimeNotReady
	}
	if len(current.ExecutionSnapshot.SupportedEngineAPIMajors) == 0 {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionMissingAPISupport
	}
	if !supportedContainerDaemonPlatform(current.ExecutionSnapshot.OperatingSystem, current.ExecutionSnapshot.Architecture) {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionUnsupportedPlatform
	}
	freshness := registry.executionSnapshotFreshness
	if freshness <= 0 {
		freshness = DefaultExecutionSnapshotFreshness
	}
	age := now.Sub(current.ExecutionSnapshot.ObservedAt)
	if age < 0 || age > freshness {
		return agentdomain.AgentExecutionCapabilitySnapshot{}, executionSnapshotAdmissionStale
	}
	return current.ExecutionSnapshot.Clone(), ""
}

func supportedContainerDaemonPlatform(operatingSystem, architecture string) bool {
	if operatingSystem != "linux" {
		return false
	}
	return architecture == "amd64" || architecture == "arm64"
}

func (registry *ActiveSessionRegistry) Register(agentID int, streamID uint64, sessionID string) (ActiveControlSession, *ActiveControlSession) {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || streamID == 0 || sessionID == "" {
		return ActiveControlSession{}, nil
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	now := registry.now().UTC()
	var replaced *ActiveControlSession
	var reusedEpoch int64
	// Reuse the epoch only when the agent re-registers the same live session.
	// A different or expired session must get a new epoch so stale stream messages
	// cannot be treated as belonging to the current control session.
	if current, ok := registry.sessions[agentID]; ok {
		copied := current
		replaced = &copied
		if current.SessionID == sessionID && !registry.isExpiredLocked(current, now) {
			reusedEpoch = current.SessionEpoch
		}
	}

	nextEpoch := reusedEpoch
	if nextEpoch <= 0 {
		nextEpoch = registry.lastEpoch[agentID] + 1
		registry.lastEpoch[agentID] = nextEpoch
	}

	session := ActiveControlSession{
		AgentID:      agentID,
		SessionID:    sessionID,
		SessionEpoch: nextEpoch,
		StreamID:     streamID,
		LastSeenAt:   now,
		Phase:        ControlSessionPhasePendingReady,
	}
	registry.sessions[agentID] = session
	return session, replaced
}

// SeedLastEpoch raises the in-memory fencing lower bound from persisted runtime
// state after a server restart. It never lowers an existing epoch.
func (registry *ActiveSessionRegistry) SeedLastEpoch(agentID int, persistedEpoch int64) {
	if registry == nil || agentID <= 0 || persistedEpoch <= 0 {
		return
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	// Server restarts must not allow the next control session to reuse an epoch
	// that could still be attached to persisted running task leases.
	if registry.lastEpoch[agentID] < persistedEpoch {
		registry.lastEpoch[agentID] = persistedEpoch
	}
}

// RestorePersistedSession reconstructs only a fresh detached process session.
// Stale or incomplete persisted state still raises the epoch floor, but cannot
// authorize reuse after a Server restart.
func (registry *ActiveSessionRegistry) RestorePersistedSession(agentID int, sessionID string, persistedEpoch int64, lastHeartbeat *time.Time) bool {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || persistedEpoch <= 0 {
		return false
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.lastEpoch[agentID] < persistedEpoch {
		registry.lastEpoch[agentID] = persistedEpoch
	}
	if _, exists := registry.sessions[agentID]; exists || sessionID == "" || lastHeartbeat == nil || lastHeartbeat.IsZero() {
		return false
	}
	now := registry.now().UTC()
	observed := lastHeartbeat.UTC()
	age := now.Sub(observed)
	if age < 0 || age > registry.heartbeatTimeout {
		return false
	}
	registry.sessions[agentID] = ActiveControlSession{
		AgentID:      agentID,
		SessionID:    sessionID,
		SessionEpoch: persistedEpoch,
		LastSeenAt:   observed,
		Phase:        ControlSessionPhaseDetachedRecoverable,
	}
	return true
}

func (registry *ActiveSessionRegistry) MarkReadyIfCurrent(agentID int, sessionID string, sessionEpoch int64, streamID uint64) (ActiveControlSession, bool) {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 || streamID == 0 {
		return ActiveControlSession{}, false
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	current, ok := registry.sessions[agentID]
	if !ok {
		return ActiveControlSession{}, false
	}
	if current.SessionID != sessionID || current.SessionEpoch != sessionEpoch || current.StreamID != streamID {
		return ActiveControlSession{}, false
	}
	current.Phase = ControlSessionPhaseReadyAttached
	registry.sessions[agentID] = current
	return current, true
}

func (registry *ActiveSessionRegistry) MatchesCurrentSession(agentID int, sessionID string, sessionEpoch int64, streamID uint64) bool {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 || streamID == 0 {
		return false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()

	current, ok := registry.sessions[agentID]
	if !ok {
		return false
	}
	if !current.CanAcceptOperationalMessages() {
		return false
	}
	if registry.isExpiredLocked(current, registry.now().UTC()) {
		return false
	}
	return current.SessionID == sessionID && current.SessionEpoch == sessionEpoch && current.StreamID == streamID
}

// CurrentReadySession returns a detached snapshot of the active operational
// session. Data-plane authorization uses this to compare a persisted task
// lease epoch without accepting caller-supplied session identity.
func (registry *ActiveSessionRegistry) CurrentReadySession(agentID int) (ActiveControlSession, bool) {
	if registry == nil || agentID <= 0 {
		return ActiveControlSession{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()
	current, ok := registry.sessions[agentID]
	if !ok || !current.CanAcceptOperationalMessages() || registry.isExpiredLocked(current, registry.now().UTC()) {
		return ActiveControlSession{}, false
	}
	if current.ExecutionSnapshot != nil {
		copy := current.ExecutionSnapshot.Clone()
		current.ExecutionSnapshot = &copy
	}
	return current, true
}

// CurrentLeaseSession returns the current session tuple that may authorize an
// already claimed operational data-plane call. A short detached interval is
// allowed; pending-ready and expired sessions remain fenced out.
func (registry *ActiveSessionRegistry) CurrentLeaseSession(agentID int) (ActiveControlSession, bool) {
	if registry == nil || agentID <= 0 {
		return ActiveControlSession{}, false
	}

	registry.mu.RLock()
	defer registry.mu.RUnlock()
	current, ok := registry.sessions[agentID]
	if !ok || !current.IsRegistered() || registry.isExpiredLocked(current, registry.now().UTC()) {
		return ActiveControlSession{}, false
	}
	if current.Phase != ControlSessionPhaseReadyAttached && current.Phase != ControlSessionPhaseDetachedRecoverable {
		return ActiveControlSession{}, false
	}
	if current.ExecutionSnapshot != nil {
		copy := current.ExecutionSnapshot.Clone()
		current.ExecutionSnapshot = &copy
	}
	return current, true
}

// RefreshLastSeenIfCurrent refreshes LastSeenAt only when the caller still
// matches the current fenced session for the agent. Stale streams must not
// extend liveness.
func (registry *ActiveSessionRegistry) RefreshLastSeenIfCurrent(agentID int, sessionID string, sessionEpoch int64, streamID uint64) bool {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 || streamID == 0 {
		return false
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	current, ok := registry.sessions[agentID]
	if !ok {
		return false
	}
	if !current.CanAcceptOperationalMessages() {
		return false
	}
	if registry.isExpiredLocked(current, registry.now().UTC()) {
		return false
	}
	if current.SessionID != sessionID || current.SessionEpoch != sessionEpoch || current.StreamID != streamID {
		return false
	}
	current.LastSeenAt = registry.now().UTC()
	registry.sessions[agentID] = current
	return true
}

func (registry *ActiveSessionRegistry) DetachStreamIfCurrent(agentID int, sessionID string, sessionEpoch int64, streamID uint64) bool {
	sessionID = strings.TrimSpace(sessionID)
	if registry == nil || agentID <= 0 || sessionID == "" || sessionEpoch <= 0 || streamID == 0 {
		return false
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	current, ok := registry.sessions[agentID]
	if !ok {
		return false
	}
	if current.SessionID != sessionID || current.SessionEpoch != sessionEpoch || current.StreamID != streamID {
		return false
	}
	current.StreamID = 0
	current.Phase = ControlSessionPhaseDetachedRecoverable
	registry.sessions[agentID] = current
	return true
}

func (registry *ActiveSessionRegistry) isExpiredLocked(session ActiveControlSession, now time.Time) bool {
	if registry == nil {
		return true
	}
	if session.LastSeenAt.IsZero() {
		return true
	}
	return now.Sub(session.LastSeenAt) > registry.heartbeatTimeout
}
