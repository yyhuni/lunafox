package agentcontrol

import (
	"reflect"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
)

func TestActiveSessionRegistryExecutionSnapshotFailsClosedWhenMissingEmptyStaleOrNotReady(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(time.Minute, func() time.Time { return now })
	registry.executionSnapshotFreshness = 10 * time.Second
	session, _ := registry.Register(31, 401, "session-31")
	if _, ok := registry.MarkReadyIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID); !ok {
		t.Fatal("mark ready failed")
	}
	read := func() bool {
		_, ok := registry.CurrentExecutionSnapshotIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID)
		return ok
	}
	if read() {
		t.Fatal("missing execution snapshot must fail closed")
	}
	if !registry.UpdateExecutionSnapshotIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID, agentdomain.AgentExecutionCapabilitySnapshot{OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true}) {
		t.Fatal("attach empty snapshot failed")
	}
	if read() {
		t.Fatal("empty supported-major set must fail closed")
	}
	if !registry.UpdateExecutionSnapshotIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID, agentdomain.AgentExecutionCapabilitySnapshot{OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: false, SupportedEngineAPIMajors: []uint32{2}}) {
		t.Fatal("attach not-ready snapshot failed")
	}
	if read() {
		t.Fatal("container runtime not ready must fail closed")
	}
	if !registry.UpdateExecutionSnapshotIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID, agentdomain.AgentExecutionCapabilitySnapshot{OperatingSystem: "linux", Architecture: "amd64", ContainerRuntimeReady: true, SupportedEngineAPIMajors: []uint32{2}}) {
		t.Fatal("attach ready snapshot failed")
	}
	snapshot, ok := registry.CurrentExecutionSnapshotIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID)
	if !ok || len(snapshot.SupportedEngineAPIMajors) != 1 || snapshot.SupportedEngineAPIMajors[0] != 2 {
		t.Fatalf("fresh ready snapshot = %#v, %v", snapshot, ok)
	}
	snapshot.SupportedEngineAPIMajors[0] = 99
	if again, ok := registry.CurrentExecutionSnapshotIfCurrent(31, session.SessionID, session.SessionEpoch, session.StreamID); !ok || again.SupportedEngineAPIMajors[0] != 2 {
		t.Fatalf("returned snapshot was not detached: %#v, %v", again, ok)
	}
	now = now.Add(11 * time.Second)
	if read() {
		t.Fatal("stale execution snapshot must fail closed")
	}
}

func TestActiveSessionRegistryRejectsUnsupportedContainerDaemonPlatform(t *testing.T) {
	registry := NewActiveSessionRegistry()
	session, _ := registry.Register(32, 402, "session-32")
	if _, ok := registry.MarkReadyIfCurrent(32, session.SessionID, session.SessionEpoch, session.StreamID); !ok {
		t.Fatal("mark ready failed")
	}
	if !registry.UpdateExecutionSnapshotIfCurrent(32, session.SessionID, session.SessionEpoch, session.StreamID, agentdomain.AgentExecutionCapabilitySnapshot{
		OperatingSystem: "darwin", Architecture: "arm64", ContainerRuntimeReady: true, SupportedEngineAPIMajors: []uint32{2},
	}) {
		t.Fatal("attach unsupported platform snapshot failed")
	}
	if _, reason := registry.currentExecutionSnapshotIfCurrent(32, session.SessionID, session.SessionEpoch, session.StreamID); reason != executionSnapshotAdmissionUnsupportedPlatform {
		t.Fatalf("unsupported daemon platform reason = %q", reason)
	}
}

func TestActiveSessionRegistryCurrentLeaseSessionAllowsFreshDetachedButRejectsPendingAndExpired(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(time.Minute, func() time.Time { return now })
	session, _ := registry.Register(33, 403, "session-33")
	if _, ok := registry.CurrentLeaseSession(33); ok {
		t.Fatal("pending-ready session authorized an operational lease")
	}
	if _, ok := registry.MarkReadyIfCurrent(33, session.SessionID, session.SessionEpoch, session.StreamID); !ok {
		t.Fatal("mark ready failed")
	}
	if current, ok := registry.CurrentLeaseSession(33); !ok || current.SessionID != session.SessionID || current.SessionEpoch != session.SessionEpoch {
		t.Fatalf("ready lease session = %#v, %v", current, ok)
	}
	if !registry.DetachStreamIfCurrent(33, session.SessionID, session.SessionEpoch, session.StreamID) {
		t.Fatal("detach current stream failed")
	}
	if current, ok := registry.CurrentLeaseSession(33); !ok || current.Phase != ControlSessionPhaseDetachedRecoverable || current.StreamID != 0 {
		t.Fatalf("detached lease session = %#v, %v", current, ok)
	}
	now = now.Add(time.Minute + time.Nanosecond)
	if _, ok := registry.CurrentLeaseSession(33); ok {
		t.Fatal("expired detached session authorized an operational lease")
	}
}

func TestActiveSessionRegistryExpiredAttachedSessionCannotRefreshOrRemainCurrent(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(time.Minute, func() time.Time { return now })
	first, _ := registry.Register(34, 404, "session-34")
	if _, ok := registry.MarkReadyIfCurrent(34, first.SessionID, first.SessionEpoch, first.StreamID); !ok {
		t.Fatal("mark ready failed")
	}
	now = now.Add(time.Minute + time.Nanosecond)
	if registry.MatchesCurrentSession(34, first.SessionID, first.SessionEpoch, first.StreamID) {
		t.Fatal("expired attached session remained operational")
	}
	if registry.RefreshLastSeenIfCurrent(34, first.SessionID, first.SessionEpoch, first.StreamID) {
		t.Fatal("expired attached session refreshed itself")
	}
	second, replaced := registry.Register(34, 405, "session-34")
	if replaced == nil || second.SessionEpoch <= first.SessionEpoch {
		t.Fatalf("expired same-process registration reused epoch: first=%#v second=%#v replaced=%#v", first, second, replaced)
	}
}

func TestActiveControlSessionDoesNotExposeConnectedAt(t *testing.T) {
	sessionType := reflect.TypeOf(ActiveControlSession{})
	if _, ok := sessionType.FieldByName("ConnectedAt"); ok {
		t.Fatal("expected ActiveControlSession to omit ConnectedAt")
	}
}

func TestActiveControlSessionIsRegistered(t *testing.T) {
	if (ActiveControlSession{}).IsRegistered() {
		t.Fatal("expected zero-value session not to be considered registered")
	}

	session := ActiveControlSession{
		AgentID:      1,
		SessionID:    "session-1",
		SessionEpoch: 1,
		StreamID:     101,
		Phase:        ControlSessionPhaseReadyAttached,
	}
	if !session.IsRegistered() {
		t.Fatalf("expected populated session to be considered registered: %+v", session)
	}
}

func TestNewActiveSessionRegistryAppliesDefaults(t *testing.T) {
	registry := newActiveSessionRegistry(0, nil)

	if registry.heartbeatTimeout != DefaultSessionHeartbeatTimeout {
		t.Fatalf("expected default heartbeat timeout, got %s", registry.heartbeatTimeout)
	}
	if registry.now == nil {
		t.Fatal("expected default clock to be initialized")
	}
}

func TestActiveSessionRegistryRegisterRejectsInvalidInputs(t *testing.T) {
	registry := NewActiveSessionRegistry()
	var nilRegistry *ActiveSessionRegistry

	testCases := []struct {
		name      string
		registry  *ActiveSessionRegistry
		agentID   int
		streamID  uint64
		sessionID string
	}{
		{name: "nil registry", registry: nilRegistry, agentID: 1, streamID: 1, sessionID: "session-1"},
		{name: "invalid agent id", registry: registry, agentID: 0, streamID: 1, sessionID: "session-1"},
		{name: "invalid stream id", registry: registry, agentID: 1, streamID: 0, sessionID: "session-1"},
		{name: "blank session id", registry: registry, agentID: 1, streamID: 1, sessionID: " \t "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			session, replaced := tc.registry.Register(tc.agentID, tc.streamID, tc.sessionID)
			if session != (ActiveControlSession{}) {
				t.Fatalf("expected zero-value session, got %+v", session)
			}
			if replaced != nil {
				t.Fatalf("expected no replaced session, got %+v", replaced)
			}
		})
	}
}

func TestActiveSessionRegistryReusesEpochForSameSessionReconnect(t *testing.T) {
	registry := NewActiveSessionRegistry()

	first, replaced := registry.Register(7, 101, "session-7")
	if replaced != nil {
		t.Fatalf("expected first registration not to replace session, got %+v", replaced)
	}

	second, replaced := registry.Register(7, 202, "session-7")
	if replaced == nil {
		t.Fatalf("expected same-session reconnect to report previous attachment")
	}
	if second.SessionEpoch != first.SessionEpoch {
		t.Fatalf("expected same-session reconnect to reuse epoch, first=%d second=%d", first.SessionEpoch, second.SessionEpoch)
	}
	if second.StreamID == first.StreamID {
		t.Fatalf("expected same-session reconnect to replace stream id")
	}
}

func TestActiveSessionRegistryRegisterStartsSessionInPendingReadyPhase(t *testing.T) {
	registry := NewActiveSessionRegistry()

	session, _ := registry.Register(21, 301, "session-21")

	if session.Phase != ControlSessionPhasePendingReady {
		t.Fatalf("expected registered session to start pending ready, got %q", session.Phase)
	}
	if session.IsDownlinkEligible() {
		t.Fatal("expected pending ready session not to be downlink eligible")
	}
}

func TestActiveSessionRegistryMarkReadyIfCurrentTransitionsToReadyAttached(t *testing.T) {
	registry := NewActiveSessionRegistry()

	session, _ := registry.Register(22, 302, "session-22")
	readySession, ok := registry.MarkReadyIfCurrent(22, "session-22", session.SessionEpoch, 302)
	if !ok {
		t.Fatal("expected current pending session to become ready")
	}
	if readySession.Phase != ControlSessionPhaseReadyAttached {
		t.Fatalf("expected ready attached phase, got %q", readySession.Phase)
	}
	if !readySession.IsDownlinkEligible() {
		t.Fatal("expected ready attached session to be downlink eligible")
	}
}

func TestActiveSessionRegistryMarkReadyIfCurrentRejectsInvalidOrStaleSession(t *testing.T) {
	registry := NewActiveSessionRegistry()
	var nilRegistry *ActiveSessionRegistry

	session, _ := registry.Register(24, 304, "session-24")

	testCases := []struct {
		name         string
		registry     *ActiveSessionRegistry
		agentID      int
		sessionID    string
		sessionEpoch int64
		streamID     uint64
	}{
		{name: "nil registry", registry: nilRegistry, agentID: 24, sessionID: "session-24", sessionEpoch: session.SessionEpoch, streamID: 304},
		{name: "invalid agent id", registry: registry, agentID: 0, sessionID: "session-24", sessionEpoch: session.SessionEpoch, streamID: 304},
		{name: "blank session id", registry: registry, agentID: 24, sessionID: "   ", sessionEpoch: session.SessionEpoch, streamID: 304},
		{name: "invalid epoch", registry: registry, agentID: 24, sessionID: "session-24", sessionEpoch: 0, streamID: 304},
		{name: "invalid stream", registry: registry, agentID: 24, sessionID: "session-24", sessionEpoch: session.SessionEpoch, streamID: 0},
		{name: "missing current session", registry: registry, agentID: 99, sessionID: "session-99", sessionEpoch: 1, streamID: 1},
		{name: "mismatched session id", registry: registry, agentID: 24, sessionID: "other-session", sessionEpoch: session.SessionEpoch, streamID: 304},
		{name: "mismatched epoch", registry: registry, agentID: 24, sessionID: "session-24", sessionEpoch: session.SessionEpoch + 1, streamID: 304},
		{name: "mismatched stream", registry: registry, agentID: 24, sessionID: "session-24", sessionEpoch: session.SessionEpoch, streamID: 999},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if ready, ok := tc.registry.MarkReadyIfCurrent(tc.agentID, tc.sessionID, tc.sessionEpoch, tc.streamID); ok || ready != (ActiveControlSession{}) {
				t.Fatalf("expected invalid or stale mark-ready request to fail, ready=%+v ok=%v", ready, ok)
			}
		})
	}
}

func TestActiveSessionRegistryAllocatesNewEpochForSessionTakeover(t *testing.T) {
	registry := NewActiveSessionRegistry()

	first, _ := registry.Register(9, 101, "session-old")
	second, replaced := registry.Register(9, 202, "session-new")

	if replaced == nil {
		t.Fatalf("expected session takeover to report replaced session")
	}
	if second.SessionEpoch <= first.SessionEpoch {
		t.Fatalf("expected new session takeover to allocate newer epoch, first=%d second=%d", first.SessionEpoch, second.SessionEpoch)
	}
	if replaced.SessionID != "session-old" || replaced.SessionEpoch != first.SessionEpoch {
		t.Fatalf("unexpected replaced session: %+v", replaced)
	}
}

func TestActiveSessionRegistrySeedLastEpochPreventsColdStartEpochReuse(t *testing.T) {
	registry := NewActiveSessionRegistry()
	registry.SeedLastEpoch(9, 2)

	session, replaced := registry.Register(9, 202, "session-new")

	if replaced != nil {
		t.Fatalf("expected cold-start seed not to create replaced session, got %+v", replaced)
	}
	if session.SessionEpoch != 3 {
		t.Fatalf("expected epoch 3 after persisted seed 2, got %d", session.SessionEpoch)
	}
}

func TestActiveSessionRegistryRestoresOnlyFreshSameProcessSession(t *testing.T) {
	now := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time { return now })
	lastHeartbeat := now.Add(-time.Minute)
	if !registry.RestorePersistedSession(9, "session-live", 2, &lastHeartbeat) {
		t.Fatal("fresh persisted session was not restored")
	}

	same, _ := registry.Register(9, 202, "session-live")
	if same.SessionEpoch != 2 {
		t.Fatalf("same process epoch = %d, want restored 2", same.SessionEpoch)
	}
	different, _ := registry.Register(9, 303, "session-new")
	if different.SessionEpoch != 3 {
		t.Fatalf("different process epoch = %d, want fenced 3", different.SessionEpoch)
	}
}

func TestActiveSessionRegistryStalePersistedSessionOnlySeedsEpochFloor(t *testing.T) {
	now := time.Date(2026, 7, 22, 14, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time { return now })
	lastHeartbeat := now.Add(-2*time.Minute - time.Nanosecond)
	if registry.RestorePersistedSession(9, "session-stale", 2, &lastHeartbeat) {
		t.Fatal("stale persisted session must not be restored")
	}

	session, _ := registry.Register(9, 202, "session-stale")
	if session.SessionEpoch != 3 {
		t.Fatalf("stale process epoch = %d, want fenced 3", session.SessionEpoch)
	}
}

func TestActiveSessionRegistryMatchesCurrentSession(t *testing.T) {
	registry := NewActiveSessionRegistry()

	current, _ := registry.Register(13, 101, " session-13 ")
	if registry.MatchesCurrentSession(13, "session-13", current.SessionEpoch, 101) {
		t.Fatal("expected pending ready session not to match operational current session")
	}
	current, ok := registry.MarkReadyIfCurrent(13, "session-13", current.SessionEpoch, 101)
	if !ok {
		t.Fatal("expected current session to transition to ready state")
	}
	if !registry.MatchesCurrentSession(13, "session-13", current.SessionEpoch, 101) {
		t.Fatal("expected current session to match with trimmed session id")
	}
	if registry.MatchesCurrentSession(13, "session-13", current.SessionEpoch, 202) {
		t.Fatal("expected mismatched stream id not to match current session")
	}
	if registry.MatchesCurrentSession(13, "session-13", current.SessionEpoch+1, 101) {
		t.Fatal("expected mismatched epoch not to match current session")
	}
	if registry.MatchesCurrentSession(13, "other-session", current.SessionEpoch, 101) {
		t.Fatal("expected mismatched session id not to match current session")
	}
	if registry.MatchesCurrentSession(99, "session-13", current.SessionEpoch, 101) {
		t.Fatal("expected missing agent not to match current session")
	}
}

func TestActiveSessionRegistryCurrentReadySessionReturnsOnlyFreshReadySession(t *testing.T) {
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(time.Minute, func() time.Time { return now })
	registered, _ := registry.Register(17, 9, "session-17")
	if _, ok := registry.CurrentReadySession(17); ok {
		t.Fatal("pending session must not be visible to data-plane authorization")
	}
	ready, ok := registry.MarkReadyIfCurrent(17, registered.SessionID, registered.SessionEpoch, registered.StreamID)
	if !ok {
		t.Fatal("mark ready failed")
	}
	current, ok := registry.CurrentReadySession(17)
	if !ok || current != ready {
		t.Fatalf("current ready session = %#v, %v; want %#v", current, ok, ready)
	}
	now = now.Add(time.Minute + time.Nanosecond)
	if _, ok := registry.CurrentReadySession(17); ok {
		t.Fatal("expired session must not authorize data-plane requests")
	}
}

func TestActiveSessionRegistryMatchesCurrentSessionRejectsInvalidInputs(t *testing.T) {
	registry := NewActiveSessionRegistry()
	var nilRegistry *ActiveSessionRegistry

	testCases := []struct {
		name         string
		registry     *ActiveSessionRegistry
		agentID      int
		sessionID    string
		sessionEpoch int64
		streamID     uint64
	}{
		{name: "nil registry", registry: nilRegistry, agentID: 1, sessionID: "session-1", sessionEpoch: 1, streamID: 1},
		{name: "invalid agent id", registry: registry, agentID: 0, sessionID: "session-1", sessionEpoch: 1, streamID: 1},
		{name: "blank session id", registry: registry, agentID: 1, sessionID: " \n ", sessionEpoch: 1, streamID: 1},
		{name: "invalid session epoch", registry: registry, agentID: 1, sessionID: "session-1", sessionEpoch: 0, streamID: 1},
		{name: "invalid stream id", registry: registry, agentID: 1, sessionID: "session-1", sessionEpoch: 1, streamID: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.registry.MatchesCurrentSession(tc.agentID, tc.sessionID, tc.sessionEpoch, tc.streamID) {
				t.Fatal("expected invalid input not to match current session")
			}
		})
	}
}

func TestActiveSessionRegistryRefreshLastSeenIfCurrent(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time {
		return now
	})

	current, _ := registry.Register(15, 101, "session-15")
	if _, ok := registry.MarkReadyIfCurrent(15, "session-15", current.SessionEpoch, 101); !ok {
		t.Fatal("expected current session to become ready before refresh")
	}

	now = now.Add(30 * time.Second)
	if !registry.RefreshLastSeenIfCurrent(15, " session-15 ", current.SessionEpoch, 101) {
		t.Fatal("expected current session to refresh last-seen timestamp")
	}
	refreshed := registry.sessions[15]
	if !refreshed.LastSeenAt.Equal(now) {
		t.Fatalf("expected refreshed last-seen timestamp %s, got %s", now, refreshed.LastSeenAt)
	}

	now = now.Add(30 * time.Second)
	if registry.RefreshLastSeenIfCurrent(15, "session-15", current.SessionEpoch, 202) {
		t.Fatal("expected stale stream not to refresh last-seen timestamp")
	}
	unchanged := registry.sessions[15]
	if !unchanged.LastSeenAt.Equal(refreshed.LastSeenAt) {
		t.Fatalf("expected mismatched refresh to keep last-seen timestamp %s, got %s", refreshed.LastSeenAt, unchanged.LastSeenAt)
	}
}

func TestActiveSessionRegistryRefreshLastSeenIfCurrentRejectsInvalidOrMissingSession(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time {
		return now
	})
	var nilRegistry *ActiveSessionRegistry

	testCases := []struct {
		name         string
		registry     *ActiveSessionRegistry
		agentID      int
		sessionID    string
		sessionEpoch int64
		streamID     uint64
	}{
		{name: "nil registry", registry: nilRegistry, agentID: 1, sessionID: "session-1", sessionEpoch: 1, streamID: 1},
		{name: "invalid agent id", registry: registry, agentID: 0, sessionID: "session-1", sessionEpoch: 1, streamID: 1},
		{name: "blank session id", registry: registry, agentID: 1, sessionID: " \t ", sessionEpoch: 1, streamID: 1},
		{name: "invalid session epoch", registry: registry, agentID: 1, sessionID: "session-1", sessionEpoch: 0, streamID: 1},
		{name: "invalid stream id", registry: registry, agentID: 1, sessionID: "session-1", sessionEpoch: 1, streamID: 0},
		{name: "missing current session", registry: registry, agentID: 99, sessionID: "session-99", sessionEpoch: 1, streamID: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.registry.RefreshLastSeenIfCurrent(tc.agentID, tc.sessionID, tc.sessionEpoch, tc.streamID) {
				t.Fatal("expected invalid or missing session not to refresh last-seen timestamp")
			}
		})
	}
}

func TestActiveSessionRegistryRefreshLastSeenIfCurrentRejectsPendingSession(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time {
		return now
	})

	session, _ := registry.Register(26, 306, "session-26")
	if registry.RefreshLastSeenIfCurrent(26, "session-26", session.SessionEpoch, 306) {
		t.Fatal("expected pending session not to refresh last-seen timestamp")
	}
}

func TestActiveSessionRegistryAllocatesNewEpochAfterRecoveryWindowExpires(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time {
		return now
	})

	first, _ := registry.Register(11, 101, "session-11")
	if _, ok := registry.MarkReadyIfCurrent(11, "session-11", first.SessionEpoch, 101); !ok {
		t.Fatal("expected first session to become ready")
	}
	if !registry.DetachStreamIfCurrent(11, "session-11", first.SessionEpoch, 101) {
		t.Fatalf("expected first session to detach cleanly")
	}

	now = now.Add(3 * time.Minute)
	second, _ := registry.Register(11, 202, "session-11")

	if second.SessionEpoch <= first.SessionEpoch {
		t.Fatalf("expected expired session to allocate newer epoch, first=%d second=%d", first.SessionEpoch, second.SessionEpoch)
	}
}

func TestActiveSessionRegistryReusesEpochAtRecoveryWindowBoundary(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time {
		return now
	})

	first, _ := registry.Register(17, 101, "session-17")
	if _, ok := registry.MarkReadyIfCurrent(17, "session-17", first.SessionEpoch, 101); !ok {
		t.Fatal("expected first session to become ready")
	}
	if !registry.DetachStreamIfCurrent(17, "session-17", first.SessionEpoch, 101) {
		t.Fatalf("expected first session to detach cleanly")
	}

	now = now.Add(2 * time.Minute)
	second, replaced := registry.Register(17, 202, "session-17")

	if replaced == nil {
		t.Fatal("expected reconnect at recovery boundary to report previous attachment")
	}
	if second.SessionEpoch != first.SessionEpoch {
		t.Fatalf("expected reconnect at recovery boundary to reuse epoch, first=%d second=%d", first.SessionEpoch, second.SessionEpoch)
	}
}

func TestActiveSessionRegistryExpirationTreatsMissingLastSeenAsExpired(t *testing.T) {
	now := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	registry := newActiveSessionRegistry(2*time.Minute, func() time.Time {
		return now
	})

	if !registry.isExpiredLocked(ActiveControlSession{}, now) {
		t.Fatal("expected zero-value session to be treated as expired")
	}
}

func TestActiveSessionRegistryDetachStreamIfCurrentRejectsInvalidOrMissingSession(t *testing.T) {
	registry := NewActiveSessionRegistry()
	var nilRegistry *ActiveSessionRegistry

	testCases := []struct {
		name         string
		registry     *ActiveSessionRegistry
		agentID      int
		sessionID    string
		sessionEpoch int64
		streamID     uint64
	}{
		{name: "nil registry", registry: nilRegistry, agentID: 1, sessionID: "session-1", sessionEpoch: 1, streamID: 1},
		{name: "invalid agent id", registry: registry, agentID: 0, sessionID: "session-1", sessionEpoch: 1, streamID: 1},
		{name: "blank session id", registry: registry, agentID: 1, sessionID: " \t ", sessionEpoch: 1, streamID: 1},
		{name: "invalid session epoch", registry: registry, agentID: 1, sessionID: "session-1", sessionEpoch: 0, streamID: 1},
		{name: "invalid stream id", registry: registry, agentID: 1, sessionID: "session-1", sessionEpoch: 1, streamID: 0},
		{name: "missing current session", registry: registry, agentID: 99, sessionID: "session-99", sessionEpoch: 1, streamID: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.registry.DetachStreamIfCurrent(tc.agentID, tc.sessionID, tc.sessionEpoch, tc.streamID) {
				t.Fatal("expected invalid or missing session not to detach stream")
			}
		})
	}
}

func TestActiveSessionRegistryDetachStreamIfCurrentTransitionsToDetachedRecoverable(t *testing.T) {
	registry := NewActiveSessionRegistry()

	session, _ := registry.Register(23, 303, "session-23")
	if _, ok := registry.MarkReadyIfCurrent(23, "session-23", session.SessionEpoch, 303); !ok {
		t.Fatal("expected session to become ready before detach")
	}
	if !registry.DetachStreamIfCurrent(23, "session-23", session.SessionEpoch, 303) {
		t.Fatal("expected ready session to detach")
	}

	detached := registry.sessions[23]
	if detached.Phase != ControlSessionPhaseDetachedRecoverable {
		t.Fatalf("expected detached recoverable phase, got %q", detached.Phase)
	}
	if detached.StreamID != 0 {
		t.Fatalf("expected detached session stream id cleared, got %d", detached.StreamID)
	}
	if detached.IsDownlinkEligible() {
		t.Fatal("expected detached session not to be downlink eligible")
	}
}

func TestActiveSessionRegistryExpirationTreatsNilRegistryAsExpired(t *testing.T) {
	var nilRegistry *ActiveSessionRegistry

	if !nilRegistry.isExpiredLocked(ActiveControlSession{LastSeenAt: time.Now().UTC()}, time.Now().UTC()) {
		t.Fatal("expected nil registry to treat session as expired")
	}
}
