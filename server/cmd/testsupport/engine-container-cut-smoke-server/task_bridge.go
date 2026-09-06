package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/contracts/agentexecution"
	agentexecutionv1 "github.com/yyhuni/lunafox/contracts/gen/lunafox/agent/execution/v1"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

const smokeAgentID = 1

type smokeCounters struct {
	planAssignments         int
	noTaskAssignments       int
	claimRecoveryFailures   int
	claimRecoveryReplays    int
	productionClaimCalls    int
	artifactHost            int
	artifactWeb             int
	artifactConfig          int
	artifactPlatform        int
	artifactNucleiTemplates int
	progressMessages        int
	resultBatches           int
	terminalAcks            int
	recoveryReplays         int
	terminalRecoveryFails   int
	productionTerminalCalls int
}

type smokeClaimKey struct {
	agentID      int
	sessionEpoch int64
	requestID    string
}

type smokeCommittedClaim struct {
	sessionID       string
	task            string
	planBytes       []byte
	failedAfterSave bool
	returned        bool
}

type smokeTerminalSnapshot struct {
	agentID        int
	sessionID      string
	sessionEpoch   int64
	taskID         int
	result         string
	failurePresent bool
	failureKind    string
	failureMessage string
	diagnostics    *scanapp.EngineExecutionDiagnostics
}

type smokeProgressKey struct {
	agentID      int
	sessionID    string
	sessionEpoch int64
	taskID       int
	requestID    string
}

type smokeProgressSnapshot struct {
	scanID    int
	sequence  int64
	level     string
	content   string
	emittedAt time.Time
}

type smokeArtifactObservation struct {
	attempts             uint64
	streams              uint64
	records              uint64
	bytes                uint64
	unavailableFailures  uint64
	integrityCorruptions uint64
}

type smokePlanRecord struct {
	scenario smokeScenario
	scanID   int
	taskID   int
	targetID int
	engineID string
	task     string

	// planBytes is the only assignment source. Every claim/replay decodes it
	// afresh so a mutable in-memory plan cannot silently diverge from the saved
	// plan contract exercised by the smoke.
	planBytes []byte

	wordlists map[string][]byte

	claimSessionID      string
	claimSessionEpoch   int64
	assignmentDelivered bool

	hostArtifact           smokeArtifactObservation
	webArtifact            smokeArtifactObservation
	configArtifact         smokeArtifactObservation
	platformArtifact       smokeArtifactObservation
	nucleiTemplateArtifact smokeArtifactObservation
	progressMessages       int
	lastProgressSeq        int64
	resultBatches          int
	resultItems            int
	resultMatched          bool
	malformedRecords       uint64
	invalidRecords         uint64
	cancelIssued           bool

	terminal                        *smokeTerminalSnapshot
	terminalAcked                   bool
	terminalRecoverySeen            bool
	terminalRecoveryFailureInjected bool
}

// smokePlanRecordSnapshot is the immutable evidence view. It deliberately
// excludes synchronization state and mutable byte/map references.
type smokePlanRecordSnapshot struct {
	scenario smokeScenario
	scanID   int
	taskID   int
	targetID int
	engineID string
	task     string

	assignmentDelivered    bool
	hostArtifact           smokeArtifactObservation
	webArtifact            smokeArtifactObservation
	configArtifact         smokeArtifactObservation
	platformArtifact       smokeArtifactObservation
	nucleiTemplateArtifact smokeArtifactObservation
	progressMessages       int
	lastProgressSeq        int64
	resultBatches          int
	resultItems            int
	resultMatched          bool
	malformedRecords       uint64
	invalidRecords         uint64
	cancelIssued           bool
	terminal               *smokeTerminalSnapshot
	terminalAcked          bool
	terminalRecoverySeen   bool
}

type smokeAuthority struct {
	mu       sync.RWMutex
	token    string
	agent    *agentdomain.Agent
	registry *agentcontrol.ActiveSessionRegistry
	sessions *smokeScanStore

	heartbeatCount      int
	lastZeroHeartbeatAt time.Time
	connectionCount     int
	firstSessionID      string
	firstSessionEpoch   int64
	sessionDrifted      bool
}

func (authority *smokeAuthority) UseSessionStore(store *smokeScanStore) {
	if authority == nil {
		return
	}
	authority.mu.Lock()
	authority.sessions = store
	authority.mu.Unlock()
}

func newSmokeAuthority(token string, registry *agentcontrol.ActiveSessionRegistry) *smokeAuthority {
	instanceID := strings.TrimPrefix(token, "cut-smoke-agent-token-")
	if instanceID == token || instanceID == "" {
		instanceID = "smoke-agent"
	} else {
		instanceID = "cut-smoke-agent-" + instanceID
	}
	return &smokeAuthority{
		token: token, registry: registry,
		agent: &agentdomain.Agent{
			ID: smokeAgentID, InstanceID: instanceID, AuthenticationToken: token,
			AgentVersion: "0.0.0", MaxTasks: 1,
			CPUThreshold: 100, MemThreshold: 100, DiskThreshold: 100,
			Status: "offline", HealthState: "healthy",
		},
	}
}

func (authority *smokeAuthority) FindByAuthenticationToken(_ context.Context, token string) (*agentdomain.Agent, error) {
	if authority == nil {
		return nil, agentdomain.ErrAgentNotFound
	}
	authority.mu.RLock()
	defer authority.mu.RUnlock()
	if token == "" || token != authority.token {
		return nil, agentdomain.ErrAgentNotFound
	}
	copy := *authority.agent
	copy.SupportedEngineAPIMajors = append([]uint32(nil), authority.agent.SupportedEngineAPIMajors...)
	return &copy, nil
}

func (authority *smokeAuthority) OnConnected(_ context.Context, agent *agentdomain.Agent, connectionIP string) error {
	if authority == nil || agent == nil || agent.ID != smokeAgentID {
		return errors.New("smoke Agent authority is invalid")
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	authority.agent.Status = "online"
	authority.agent.ConnectionIP = connectionIP
	if authority.agent.ObservedSourceIP != connectionIP {
		authority.agent.ObservedSourceIP = connectionIP
		authority.agent.ObservedIPGeneration++
	}
	authority.agent.ConnectedAt = ptrTime(time.Now().UTC())
	authority.connectionCount++
	return nil
}

func (authority *smokeAuthority) OnControlConnectionDetached(_ context.Context, agentID int) error {
	if authority == nil || agentID != smokeAgentID {
		return nil
	}
	authority.mu.Lock()
	authority.agent.ConnectionIP = ""
	authority.mu.Unlock()
	return nil
}

func (authority *smokeAuthority) OnDisconnected(_ context.Context, agentID int) error {
	if authority == nil || agentID != smokeAgentID {
		return nil
	}
	authority.mu.Lock()
	authority.agent.Status = "offline"
	authority.mu.Unlock()
	return nil
}

func (authority *smokeAuthority) RecordHeartbeat(_ context.Context, agentID int, event agentdomain.AgentHeartbeatEvent) error {
	if authority == nil || agentID != smokeAgentID {
		return errors.New("smoke Agent authority is invalid")
	}
	if event.SessionID == "" || event.SessionEpoch <= 0 || event.RunningTasks < 0 || event.TaskSlotsUsed < 0 || event.TaskSlotsUsed < event.RunningTasks {
		return agentdomain.ErrStaleAgentHeartbeat
	}
	authority.mu.RLock()
	sessions := authority.sessions
	authority.mu.RUnlock()
	if sessions != nil {
		if err := sessions.setAgentExecutionSession(agentID, event.SessionID, event.SessionEpoch); err != nil {
			if errors.Is(err, errSmokeStaleAgentExecutionSession) {
				return agentdomain.ErrStaleAgentHeartbeat
			}
			return err
		}
	}
	authority.mu.Lock()
	defer authority.mu.Unlock()
	authority.agent.SessionID = event.SessionID
	authority.agent.SessionEpoch = event.SessionEpoch
	authority.agent.AgentVersion = event.AgentVersion
	authority.agent.OperatingSystem = event.OperatingSystem
	authority.agent.Architecture = event.Architecture
	authority.agent.ContainerRuntimeReady = event.ContainerRuntimeReady
	authority.agent.SupportedEngineAPIMajors = append([]uint32(nil), event.SupportedEngineAPIMajors...)
	authority.agent.RunningTasks = event.RunningTasks
	authority.agent.TaskSlotsUsed = event.TaskSlotsUsed
	authority.agent.LastHeartbeat = ptrTime(time.Now().UTC())
	authority.agent.Status = "online"
	if authority.firstSessionID == "" {
		authority.firstSessionID = event.SessionID
		authority.firstSessionEpoch = event.SessionEpoch
	} else if authority.firstSessionID != event.SessionID || authority.firstSessionEpoch != event.SessionEpoch {
		authority.sessionDrifted = true
	}
	authority.heartbeatCount++
	if event.RunningTasks == 0 && event.TaskSlotsUsed == 0 {
		authority.lastZeroHeartbeatAt = time.Now().UTC()
	}
	return nil
}

func (authority *smokeAuthority) sameSessionReconnectEvidence() (int, bool) {
	if authority == nil {
		return 0, false
	}
	authority.mu.RLock()
	defer authority.mu.RUnlock()
	return authority.connectionCount, authority.connectionCount >= 2 && authority.firstSessionID != "" && authority.firstSessionEpoch > 0 && !authority.sessionDrifted
}

func (authority *smokeAuthority) heartbeatAfter(at time.Time) bool {
	if authority == nil {
		return false
	}
	authority.mu.RLock()
	defer authority.mu.RUnlock()
	return !authority.lastZeroHeartbeatAt.IsZero() && authority.lastZeroHeartbeatAt.After(at)
}

func ptrTime(value time.Time) *time.Time { return &value }

type smokeTaskBridge struct {
	mu              sync.Mutex
	authority       *smokeAuthority
	queue           []*smokePlanRecord
	records         map[string]*smokePlanRecord
	byTaskID        map[int]*smokePlanRecord
	claims          map[smokeClaimKey]*smokeCommittedClaim
	progress        map[smokeProgressKey]smokeProgressSnapshot
	nextIndex       int
	counters        smokeCounters
	terminal        chan struct{}
	allAckedAt      time.Time
	cancelPublisher *agentcontrol.AgentControlEventPublisher
	production      *scanapp.ScanTaskBridgeService
}

var _ agentcontrol.EngineDiagnosticScanTaskBridge = (*smokeTaskBridge)(nil)

func newSmokeTaskBridge(authority *smokeAuthority, records []*smokePlanRecord) *smokeTaskBridge {
	byTask := make(map[string]*smokePlanRecord, len(records))
	byID := make(map[int]*smokePlanRecord, len(records))
	for _, record := range records {
		if record == nil || record.task == "" || len(record.planBytes) == 0 {
			continue
		}
		byTask[record.task] = record
		byID[record.taskID] = record
	}
	return &smokeTaskBridge{
		authority: authority,
		queue:     append([]*smokePlanRecord(nil), records...),
		records:   byTask,
		byTaskID:  byID,
		claims:    make(map[smokeClaimKey]*smokeCommittedClaim),
		progress:  make(map[smokeProgressKey]smokeProgressSnapshot),
		terminal:  make(chan struct{}, len(records)+1),
	}
}

func (bridge *smokeTaskBridge) UseProductionBridge(production *scanapp.ScanTaskBridgeService) {
	if bridge == nil {
		return
	}
	bridge.mu.Lock()
	bridge.production = production
	bridge.mu.Unlock()
}

func (bridge *smokeTaskBridge) SetCancelPublisher(publisher *agentcontrol.AgentControlEventPublisher) {
	if bridge == nil {
		return
	}
	bridge.mu.Lock()
	bridge.cancelPublisher = publisher
	bridge.mu.Unlock()
}

func supportsEngineAPIMajor(majors []uint32, wanted uint32) bool {
	for _, major := range majors {
		if major == wanted {
			return true
		}
	}
	return false
}

func (bridge *smokeTaskBridge) validateClaimScope(agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) error {
	if bridge == nil || agentID != smokeAgentID || strings.TrimSpace(sessionID) == "" || sessionID != strings.TrimSpace(sessionID) || sessionEpoch <= 0 {
		return errors.New("invalid smoke claim scope")
	}
	parsed, err := uuid.Parse(requestID)
	if err != nil || parsed.String() != requestID {
		return errors.New("claim request ID is not canonical")
	}
	if !snapshot.ContainerRuntimeReady || snapshot.OperatingSystem != "linux" || (snapshot.Architecture != "amd64" && snapshot.Architecture != "arm64") || !supportsEngineAPIMajor(snapshot.SupportedEngineAPIMajors, 2) {
		return errors.New("Agent execution capability is incompatible")
	}
	bridge.authority.mu.RLock()
	authoritySession := bridge.authority.agent.SessionID
	authorityEpoch := bridge.authority.agent.SessionEpoch
	bridge.authority.mu.RUnlock()
	if authoritySession != sessionID || authorityEpoch != sessionEpoch {
		return errors.New("claim session does not match authenticated Agent session")
	}
	return nil
}

func decodeSmokePlan(record *smokePlanRecord) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if record == nil || len(record.planBytes) == 0 {
		return nil, errors.New("smoke plan bytes are missing")
	}
	return decodeSmokePlanBytes(record.planBytes)
}

func decodeSmokePlanBytes(planBytes []byte) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if len(planBytes) == 0 {
		return nil, errors.New("smoke plan bytes are missing")
	}
	plan, err := agentexecution.UnmarshalResolvedEngineExecutionPlan(append([]byte(nil), planBytes...))
	if err != nil {
		return nil, fmt.Errorf("decode saved smoke plan: %w", err)
	}
	if err := agentexecution.ValidateResolvedEngineExecutionPlan(plan); err != nil {
		return nil, fmt.Errorf("saved smoke plan is invalid: %w", err)
	}
	return plan, nil
}

func (bridge *smokeTaskBridge) ClaimNextExecutionPlan(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, requestID string, snapshot agentdomain.AgentExecutionCapabilitySnapshot) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	if ctx == nil {
		return nil, errors.New("claim context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := bridge.validateClaimScope(agentID, sessionID, sessionEpoch, requestID, snapshot); err != nil {
		return nil, err
	}
	bridge.mu.Lock()
	production := bridge.production
	bridge.mu.Unlock()
	if production != nil {
		return bridge.claimProductionExecutionPlan(ctx, production, agentID, sessionID, sessionEpoch, requestID, snapshot)
	}
	key := smokeClaimKey{agentID: agentID, sessionEpoch: sessionEpoch, requestID: requestID}
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	if committed := bridge.claims[key]; committed != nil {
		if committed.sessionID != sessionID {
			return nil, errors.New("claim replay session ID changed")
		}
		record := bridge.records[committed.task]
		if record == nil || !bytesEqual(record.planBytes, committed.planBytes) {
			return nil, errors.New("claim replay plan bytes changed")
		}
		if record.terminal != nil || record.terminalAcked {
			return nil, errors.New("claim replay is not allowed after terminal observation")
		}
		plan, err := decodeSmokePlan(record)
		if err != nil {
			return nil, err
		}
		if committed.failedAfterSave {
			committed.failedAfterSave = false
			committed.returned = true
			bridge.counters.claimRecoveryReplays++
			record.assignmentDelivered = true
			bridge.counters.planAssignments++
			return plan, nil
		}
		if !committed.returned || !record.assignmentDelivered {
			return nil, errors.New("claim replay state is incomplete")
		}
		return plan, nil
	}

	if bridge.nextIndex > 0 && !bridge.queue[bridge.nextIndex-1].terminalAcked {
		bridge.counters.noTaskAssignments++
		return nil, nil
	}
	if bridge.nextIndex >= len(bridge.queue) {
		bridge.counters.noTaskAssignments++
		return nil, nil
	}
	record := bridge.queue[bridge.nextIndex]
	if record == nil {
		return nil, errors.New("smoke claim queue contains nil record")
	}
	plan, err := decodeSmokePlan(record)
	if err != nil {
		return nil, err
	}
	if plan.GetEngineRelease().GetEngineApiMajor() != 2 {
		return nil, errors.New("smoke plan engine API major is unsupported")
	}
	bridge.claims[key] = &smokeCommittedClaim{
		sessionID: sessionID,
		task:      record.task,
		planBytes: append([]byte(nil), record.planBytes...),
	}
	bridge.nextIndex++
	record.claimSessionID = sessionID
	record.claimSessionEpoch = sessionEpoch
	if bridge.counters.claimRecoveryFailures == 0 {
		bridge.claims[key].failedAfterSave = true
		bridge.counters.claimRecoveryFailures++
		return nil, errors.New("smoke claim intentionally interrupted after plan commit")
	}
	bridge.claims[key].returned = true
	record.assignmentDelivered = true
	bridge.counters.planAssignments++
	return plan, nil
}

func (bridge *smokeTaskBridge) claimProductionExecutionPlan(
	ctx context.Context,
	production *scanapp.ScanTaskBridgeService,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	requestID string,
	snapshot agentdomain.AgentExecutionCapabilitySnapshot,
) (*agentexecutionv1.ResolvedEngineExecutionPlan, error) {
	plan, err := production.ClaimNextExecutionPlan(ctx, agentID, sessionID, sessionEpoch, requestID, snapshot)
	if err != nil {
		return nil, err
	}
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	bridge.counters.productionClaimCalls++
	if plan == nil {
		bridge.counters.noTaskAssignments++
		return nil, nil
	}
	if err := agentexecution.ValidateResolvedEngineExecutionPlan(plan); err != nil {
		return nil, fmt.Errorf("production claim returned an invalid saved plan: %w", err)
	}
	encoded, err := agentexecution.MarshalResolvedEngineExecutionPlan(plan)
	if err != nil {
		return nil, fmt.Errorf("encode production-claimed plan: %w", err)
	}
	record := bridge.records[plan.GetTask()]
	if record == nil || !bytesEqual(record.planBytes, encoded) {
		return nil, errors.New("production claim did not return the exact persisted smoke plan")
	}
	key := smokeClaimKey{agentID: agentID, sessionEpoch: sessionEpoch, requestID: requestID}
	if committed := bridge.claims[key]; committed != nil {
		if committed.sessionID != sessionID || committed.task != record.task || !bytesEqual(committed.planBytes, encoded) {
			return nil, errors.New("production claim replay changed the committed assignment")
		}
		if record.terminal != nil || record.terminalAcked {
			return nil, errors.New("claim replay is not allowed after terminal observation")
		}
		if committed.failedAfterSave {
			committed.failedAfterSave = false
			committed.returned = true
			record.assignmentDelivered = true
			bridge.counters.claimRecoveryReplays++
			bridge.counters.planAssignments++
		}
		return plan, nil
	}
	bridge.claims[key] = &smokeCommittedClaim{
		sessionID: sessionID,
		task:      record.task,
		planBytes: append([]byte(nil), encoded...),
		returned:  true,
	}
	record.claimSessionID = sessionID
	record.claimSessionEpoch = sessionEpoch
	if bridge.counters.claimRecoveryFailures == 0 {
		bridge.claims[key].failedAfterSave = true
		bridge.claims[key].returned = false
		bridge.counters.claimRecoveryFailures++
		return nil, errors.New("smoke claim intentionally interrupted after production repository commit")
	}
	record.assignmentDelivered = true
	bridge.counters.planAssignments++
	return plan, nil
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func terminalSnapshotEqual(left, right *smokeTerminalSnapshot) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.agentID == right.agentID && left.sessionID == right.sessionID && left.sessionEpoch == right.sessionEpoch && left.taskID == right.taskID && left.result == right.result && left.failurePresent == right.failurePresent && left.failureKind == right.failureKind && left.failureMessage == right.failureMessage &&
		scandomain.EqualEngineExecutionDiagnostics(left.diagnostics, right.diagnostics)
}

// The smoke accepts only Agent-owned stable failures so free-form diagnostics
// cannot satisfy the end-to-end terminal contract.
func isCanonicalSmokeFailure(failure *scanapp.FailureDetail) bool {
	if failure == nil {
		return false
	}
	switch failure.Kind {
	case smokeEngineExitFailureKind:
		return failure.Message == smokeEngineExitFailureMessage
	case smokeResultProtocolFailureKind:
		return failure.Message == smokeResultProtocolFailureMessage
	case smokeTaskTimeoutFailureKind:
		return failure.Message == smokeTaskTimeoutFailureMessage
	case smokeConfigHashFailureKind:
		return failure.Message == smokeConfigHashFailureMessage
	case smokeNoTemplatesFailureKind:
		return failure.Message == smokeNoTemplatesFailureMessage
	default:
		return false
	}
}

func (bridge *smokeTaskBridge) ReportTerminalTaskResult(ctx context.Context, agentID int, sessionID string, sessionEpoch int64, taskID int, result string, failure *scanapp.FailureDetail) error {
	return bridge.reportTerminalTaskResult(
		ctx,
		agentID,
		sessionID,
		sessionEpoch,
		taskID,
		result,
		failure,
		scandomain.UnavailableEngineExecutionDiagnostics(),
	)
}

// ReportTerminalTaskResultWithDiagnostics is the smoke implementation of the
// production Agent-control terminal contract. The stored snapshot is part of
// terminal replay identity, so a lost acknowledgement cannot replace accepted
// diagnostics with a different observation.
func (bridge *smokeTaskBridge) ReportTerminalTaskResultWithDiagnostics(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	taskID int,
	result string,
	failure *scanapp.FailureDetail,
	diagnostics *scanapp.EngineExecutionDiagnostics,
) error {
	return bridge.reportTerminalTaskResult(ctx, agentID, sessionID, sessionEpoch, taskID, result, failure, diagnostics)
}

func (bridge *smokeTaskBridge) reportTerminalTaskResult(
	ctx context.Context,
	agentID int,
	sessionID string,
	sessionEpoch int64,
	taskID int,
	result string,
	failure *scanapp.FailureDetail,
	diagnostics *scanapp.EngineExecutionDiagnostics,
) error {
	if ctx == nil {
		return errors.New("terminal context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if bridge == nil || agentID != smokeAgentID || sessionID == "" || sessionEpoch <= 0 {
		return errors.New("invalid smoke terminal scope")
	}
	if result != "succeeded" && result != "failed" && result != "cancelled" {
		return errors.New("invalid smoke terminal result")
	}
	if (result == "failed") != (failure != nil) {
		return errors.New("smoke terminal failure presence does not match result")
	}
	terminalStatus, ok := scandomain.ParseTaskStatus(result)
	if !ok || !scandomain.IsTerminalTaskStatus(terminalStatus) {
		return errors.New("smoke terminal result is invalid")
	}
	if err := scandomain.ValidateTerminalEngineExecutionDiagnostics(terminalStatus, diagnostics); err != nil {
		return fmt.Errorf("smoke terminal diagnostics are invalid: %w", err)
	}
	received := &smokeTerminalSnapshot{
		agentID:      agentID,
		sessionID:    sessionID,
		sessionEpoch: sessionEpoch,
		taskID:       taskID,
		result:       result,
		diagnostics:  scandomain.CloneEngineExecutionDiagnostics(diagnostics),
	}
	if failure != nil {
		received.failurePresent = true
		received.failureKind = failure.Kind
		received.failureMessage = failure.Message
		if !isCanonicalSmokeFailure(failure) {
			return errors.New("smoke terminal failure is not canonical")
		}
	}
	bridge.authority.mu.RLock()
	currentSession := bridge.authority.agent.SessionID
	currentEpoch := bridge.authority.agent.SessionEpoch
	bridge.authority.mu.RUnlock()
	if currentSession != sessionID || currentEpoch != sessionEpoch {
		return errors.New("terminal lease does not match authenticated Agent session")
	}
	bridge.mu.Lock()
	production := bridge.production
	bridge.mu.Unlock()
	if production != nil {
		return bridge.reportProductionTerminalTaskResult(ctx, production, received, failure, diagnostics)
	}
	bridge.mu.Lock()
	record := bridge.byTaskID[taskID]
	if record == nil || !record.assignmentDelivered || record.claimSessionID != sessionID || record.claimSessionEpoch != sessionEpoch {
		bridge.mu.Unlock()
		return errors.New("terminal scope does not match the claimed smoke task")
	}
	if record.terminal != nil {
		if !terminalSnapshotEqual(record.terminal, received) {
			bridge.mu.Unlock()
			return errors.New("terminal replay changed the immutable snapshot")
		}
		if record.terminalAcked {
			bridge.mu.Unlock()
			return nil
		}
		record.terminalRecoverySeen = true
		bridge.counters.recoveryReplays++
		record.terminalAcked = true
		bridge.counters.terminalAcks++
		bridge.markAllAckedLocked()
		bridge.mu.Unlock()
		bridge.signalTerminal()
		return nil
	}
	record.terminal = received
	if record.scenario == smokeScenarioWebsiteEmpty {
		if !record.terminalRecoveryFailureInjected {
			record.terminalRecoveryFailureInjected = true
			bridge.counters.terminalRecoveryFails++
		}
		bridge.mu.Unlock()
		return errors.New("smoke terminal acknowledgement intentionally interrupted")
	}
	record.terminalAcked = true
	bridge.counters.terminalAcks++
	bridge.markAllAckedLocked()
	bridge.mu.Unlock()
	bridge.signalTerminal()
	return nil
}

func (bridge *smokeTaskBridge) reportProductionTerminalTaskResult(
	ctx context.Context,
	production *scanapp.ScanTaskBridgeService,
	received *smokeTerminalSnapshot,
	failure *scanapp.FailureDetail,
	diagnostics *scanapp.EngineExecutionDiagnostics,
) error {
	if received == nil {
		return errors.New("production terminal snapshot is required")
	}
	bridge.mu.Lock()
	record := bridge.byTaskID[received.taskID]
	if record == nil || !record.assignmentDelivered || record.claimSessionID != received.sessionID || record.claimSessionEpoch != received.sessionEpoch {
		bridge.mu.Unlock()
		return errors.New("terminal scope does not match the claimed smoke task")
	}
	if record.terminal != nil && !terminalSnapshotEqual(record.terminal, received) {
		bridge.mu.Unlock()
		return errors.New("terminal replay changed the immutable snapshot")
	}
	if record.terminal != nil {
		// The production commit may have succeeded while its acknowledgement was
		// lost.  The persisted smoke snapshot is the local proof that this exact
		// result was already submitted, so replay only completes the acknowledgement
		// boundary and must not submit the terminal result a second time.
		if record.terminalAcked {
			bridge.mu.Unlock()
			return nil
		}
		record.terminalRecoverySeen = true
		bridge.counters.recoveryReplays++
		record.terminalAcked = true
		bridge.counters.terminalAcks++
		bridge.markAllAckedLocked()
		bridge.mu.Unlock()
		bridge.signalTerminal()
		return nil
	}
	bridge.mu.Unlock()

	if err := production.ReportTerminalTaskResultWithDiagnostics(
		ctx,
		received.agentID,
		received.sessionID,
		received.sessionEpoch,
		received.taskID,
		received.result,
		failure,
		diagnostics,
	); err != nil {
		return err
	}

	bridge.mu.Lock()
	bridge.counters.productionTerminalCalls++
	record = bridge.byTaskID[received.taskID]
	if record == nil || !record.assignmentDelivered || record.claimSessionID != received.sessionID || record.claimSessionEpoch != received.sessionEpoch {
		bridge.mu.Unlock()
		return errors.New("terminal scope changed after production commit")
	}
	if record.terminal != nil {
		if !terminalSnapshotEqual(record.terminal, received) {
			bridge.mu.Unlock()
			return errors.New("terminal replay changed after production commit")
		}
		if record.terminalAcked {
			bridge.mu.Unlock()
			return nil
		}
		record.terminalRecoverySeen = true
		bridge.counters.recoveryReplays++
		record.terminalAcked = true
		bridge.counters.terminalAcks++
		bridge.markAllAckedLocked()
		bridge.mu.Unlock()
		bridge.signalTerminal()
		return nil
	}
	record.terminal = received
	if record.scenario == smokeScenarioWebsiteEmpty {
		record.terminalRecoveryFailureInjected = true
		bridge.counters.terminalRecoveryFails++
		bridge.mu.Unlock()
		return errors.New("smoke terminal acknowledgement intentionally interrupted after production commit")
	}
	record.terminalAcked = true
	bridge.counters.terminalAcks++
	bridge.markAllAckedLocked()
	bridge.mu.Unlock()
	bridge.signalTerminal()
	return nil
}

func (bridge *smokeTaskBridge) markAllAckedLocked() {
	if bridge.allAckedAt.IsZero() {
		for _, record := range bridge.queue {
			if record == nil || !record.terminalAcked {
				return
			}
		}
		bridge.allAckedAt = time.Now().UTC()
	}
}

func (bridge *smokeTaskBridge) signalTerminal() {
	select {
	case bridge.terminal <- struct{}{}:
	default:
	}
}

func (bridge *smokeTaskBridge) FenceSupersededAgentSession(ctx context.Context, agentID int, currentSessionEpoch int64) error {
	if bridge == nil {
		return errors.New("smoke task bridge is required")
	}
	bridge.mu.Lock()
	production := bridge.production
	bridge.mu.Unlock()
	if production == nil {
		return nil
	}
	return production.FenceSupersededAgentSession(ctx, agentID, currentSessionEpoch)
}

func (bridge *smokeTaskBridge) countersSnapshot() smokeCounters {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	return bridge.counters
}

func (bridge *smokeTaskBridge) waitTerminal(ctx context.Context, expected int) bool {
	if expected <= 0 {
		return true
	}
	seen := 0
	for seen < expected {
		select {
		case <-bridge.terminal:
			seen++
		case <-ctx.Done():
			return false
		}
	}
	return true
}

func (bridge *smokeTaskBridge) allTerminalsAcked() bool {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	return !bridge.allAckedAt.IsZero()
}

func (bridge *smokeTaskBridge) cleanupObserved() bool {
	bridge.mu.Lock()
	allAcked := !bridge.allAckedAt.IsZero()
	at := bridge.allAckedAt
	bridge.mu.Unlock()
	return allAcked && bridge.authority.heartbeatAfter(at)
}

func cloneTerminalSnapshot(value *smokeTerminalSnapshot) *smokeTerminalSnapshot {
	if value == nil {
		return nil
	}
	copy := *value
	copy.diagnostics = scandomain.CloneEngineExecutionDiagnostics(value.diagnostics)
	return &copy
}

func smokeRecordSnapshot(record *smokePlanRecord) smokePlanRecordSnapshot {
	return smokePlanRecordSnapshot{
		scenario: record.scenario, scanID: record.scanID, taskID: record.taskID,
		targetID: record.targetID, engineID: record.engineID, task: record.task,
		assignmentDelivered: record.assignmentDelivered,
		hostArtifact:        record.hostArtifact, webArtifact: record.webArtifact,
		configArtifact: record.configArtifact, platformArtifact: record.platformArtifact,
		nucleiTemplateArtifact: record.nucleiTemplateArtifact,
		progressMessages:       record.progressMessages, lastProgressSeq: record.lastProgressSeq,
		resultBatches: record.resultBatches, resultItems: record.resultItems,
		resultMatched: record.resultMatched, malformedRecords: record.malformedRecords,
		invalidRecords: record.invalidRecords, cancelIssued: record.cancelIssued,
		terminal: cloneTerminalSnapshot(record.terminal), terminalAcked: record.terminalAcked,
		terminalRecoverySeen: record.terminalRecoverySeen,
	}
}

func (bridge *smokeTaskBridge) recordSnapshots() []smokePlanRecordSnapshot {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	result := make([]smokePlanRecordSnapshot, 0, len(bridge.queue))
	for _, record := range bridge.queue {
		if record == nil {
			continue
		}
		result = append(result, smokeRecordSnapshot(record))
	}
	return result
}

func (bridge *smokeTaskBridge) recordForTaskID(taskID int) (smokePlanRecordSnapshot, bool) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	record, ok := bridge.byTaskID[taskID]
	if !ok || record == nil {
		return smokePlanRecordSnapshot{}, false
	}
	return smokeRecordSnapshot(record), true
}

func (bridge *smokeTaskBridge) observeArtifact(taskID int, kind string, records, bytes uint64) {
	if bridge == nil || taskID <= 0 {
		return
	}
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	record := bridge.byTaskID[taskID]
	if record == nil {
		return
	}
	switch kind {
	case "host":
		record.hostArtifact.streams++
		record.hostArtifact.records += records
		record.hostArtifact.bytes += bytes
		bridge.counters.artifactHost++
	case "web":
		record.webArtifact.streams++
		record.webArtifact.records += records
		record.webArtifact.bytes += bytes
		bridge.counters.artifactWeb++
	case "config":
		record.configArtifact.streams++
		record.configArtifact.records += records
		record.configArtifact.bytes += bytes
		bridge.counters.artifactConfig++
	case "platform":
		record.platformArtifact.streams++
		record.platformArtifact.records += records
		record.platformArtifact.bytes += bytes
		bridge.counters.artifactPlatform++
	case "nuclei_template":
		record.nucleiTemplateArtifact.streams++
		record.nucleiTemplateArtifact.records += records
		record.nucleiTemplateArtifact.bytes += bytes
		bridge.counters.artifactNucleiTemplates++
	}
}

func (bridge *smokeTaskBridge) observeResult(taskID int, items int, matched bool) {
	bridge.mu.Lock()
	if record := bridge.byTaskID[taskID]; record != nil {
		record.resultBatches++
		record.resultItems += items
		record.resultMatched = record.resultMatched || matched
		bridge.counters.resultBatches++
	}
	bridge.mu.Unlock()
}

var (
	_ agentcontrol.AgentControlLifecycle      = (*smokeAuthority)(nil)
	_ agentcontrol.AgentFinder                = (*smokeAuthority)(nil)
	_ agentcontrol.ScanTaskBridge             = (*smokeTaskBridge)(nil)
	_ agentcontrol.EngineExecutionClaimBridge = (*smokeTaskBridge)(nil)
	_ agentcontrol.AgentSessionFenceBridge    = (*smokeTaskBridge)(nil)
)
