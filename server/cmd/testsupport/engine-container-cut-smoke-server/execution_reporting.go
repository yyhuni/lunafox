package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/yyhuni/lunafox/contracts/results"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	resultingestapp "github.com/yyhuni/lunafox/server/internal/modules/resultingest/application"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	snapshotapp "github.com/yyhuni/lunafox/server/internal/modules/snapshot/application"
)

type smokeResultScope struct{ bridge *smokeTaskBridge }

func (scope *smokeResultScope) GetResultTaskScope(_ context.Context, request agentdata.ResultTaskScopeRequest) (*agentdata.ResultTaskScope, error) {
	if scope == nil || scope.bridge == nil || request.AgentID != smokeAgentID || request.TaskID <= 0 || request.SessionID == "" || request.SessionEpoch <= 0 {
		return nil, errors.New("result task scope is unavailable")
	}
	scope.bridge.mu.Lock()
	record := scope.bridge.byTaskID[request.TaskID]
	valid := record != nil && record.assignmentDelivered && record.claimSessionID == request.SessionID && record.claimSessionEpoch == request.SessionEpoch
	var resultScope agentdata.ResultTaskScope
	if valid {
		resultScope = agentdata.ResultTaskScope{TaskID: record.taskID, ScanID: record.scanID, TargetID: record.targetID}
	}
	scope.bridge.mu.Unlock()
	if !valid {
		return nil, errors.New("result task scope is unavailable")
	}
	return &resultScope, nil
}

type smokeProgressSink struct{ bridge *smokeTaskBridge }

func parseSmokeProgressCounter(content, field string) (uint64, bool) {
	index := strings.Index(content, field)
	if index < 0 {
		return 0, false
	}
	value := content[index+len(field):]
	if end := strings.IndexAny(value, " \t\r\n"); end >= 0 {
		value = value[:end]
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	return parsed, err == nil
}

func smokeProgressSnapshotEqual(left, right smokeProgressSnapshot) bool {
	return left.scanID == right.scanID &&
		left.sequence == right.sequence &&
		left.level == right.level &&
		left.content == right.content &&
		left.emittedAt.Equal(right.emittedAt)
}

func (sink *smokeProgressSink) WriteTaskProgressLogs(ctx context.Context, batch agentdata.TaskProgressLogBatch) (int, int, error) {
	if ctx == nil {
		return 0, 0, errors.New("progress context is required")
	}
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	if sink == nil || sink.bridge == nil || batch.TaskID <= 0 || len(batch.Entries) != 1 || batch.AgentID != smokeAgentID || strings.TrimSpace(batch.SessionID) != batch.SessionID || batch.SessionID == "" || batch.SessionEpoch <= 0 {
		return 0, 0, errors.New("progress batch is invalid")
	}
	requestID, err := uuid.Parse(strings.TrimSpace(batch.RequestID))
	if err != nil || requestID.String() != batch.RequestID || requestID.Version() != uuid.Version(4) {
		return 0, 0, errors.New("progress request ID is invalid")
	}
	entry := batch.Entries[0]
	if entry.Sequence != 1 || entry.Level != "info" || strings.TrimSpace(entry.Content) == "" || entry.EmittedAt.IsZero() {
		return 0, 0, errors.New("progress entry is not canonical")
	}
	snapshot := smokeProgressSnapshot{
		scanID: batch.ScanID, sequence: entry.Sequence, level: entry.Level,
		content: entry.Content, emittedAt: entry.EmittedAt,
	}
	key := smokeProgressKey{
		agentID: batch.AgentID, sessionID: batch.SessionID, sessionEpoch: batch.SessionEpoch,
		taskID: batch.TaskID, requestID: batch.RequestID,
	}
	sink.bridge.authority.mu.RLock()
	currentSession := sink.bridge.authority.agent.SessionID
	currentEpoch := sink.bridge.authority.agent.SessionEpoch
	sink.bridge.authority.mu.RUnlock()
	if currentSession != batch.SessionID || currentEpoch != batch.SessionEpoch {
		return 0, 0, errors.New("progress lease is stale")
	}
	sink.bridge.mu.Lock()
	record := sink.bridge.byTaskID[batch.TaskID]
	if record == nil || record.scanID != batch.ScanID || !record.assignmentDelivered || record.claimSessionID != batch.SessionID || record.claimSessionEpoch != batch.SessionEpoch {
		sink.bridge.mu.Unlock()
		return 0, 0, errors.New("progress batch does not match an active smoke claim")
	}
	if saved, ok := sink.bridge.progress[key]; ok {
		if !smokeProgressSnapshotEqual(saved, snapshot) {
			sink.bridge.mu.Unlock()
			return 0, 0, errors.New("progress replay changed the immutable request")
		}
		sink.bridge.mu.Unlock()
		return 0, 1, nil
	}
	if record.terminal != nil || record.terminalAcked {
		sink.bridge.mu.Unlock()
		return 0, 0, errors.New("new progress is not allowed after terminal observation")
	}
	shouldCancel := (record.scenario == smokeScenarioPortCancel || record.scenario == smokeScenarioNucleiCancel) && !record.cancelIssued
	if shouldCancel && sink.bridge.cancelPublisher == nil {
		sink.bridge.mu.Unlock()
		return 0, 0, errors.New("cancel publisher is unavailable")
	}
	sink.bridge.progress[key] = snapshot
	record.progressMessages++
	record.lastProgressSeq = entry.Sequence
	// The Engine emits aggregate parser counters only in its bounded completion
	// message. Keep those counters as smoke evidence without retaining raw
	// result lines or widening the production progress contract.
	if strings.Contains(entry.Content, "malformedRecords=") || strings.Contains(entry.Content, "invalidRecords=") {
		for _, field := range []struct {
			name   string
			target *uint64
		}{
			{name: "malformedRecords=", target: &record.malformedRecords},
			{name: "invalidRecords=", target: &record.invalidRecords},
		} {
			if value, ok := parseSmokeProgressCounter(entry.Content, field.name); ok {
				*field.target = value
			}
		}
	}
	sink.bridge.counters.progressMessages++
	if shouldCancel {
		record.cancelIssued = true
	}
	publisher := sink.bridge.cancelPublisher
	scanID := record.scanID
	taskID := record.taskID
	sink.bridge.mu.Unlock()
	if shouldCancel {
		// Terminal cancelled is the delivery proof; SendTaskCancel itself is a
		// best-effort control downlink and intentionally has no success result.
		publisher.SendTaskCancel(smokeAgentID, scanID, taskID)
	}
	return 1, 0, nil
}

type smokeSubdomainMaterializer struct{}

func (*smokeSubdomainMaterializer) SaveAndSyncContext(_ context.Context, _ int, _ int, items []snapshotapp.SubdomainSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: int64(len(items)), AssetCount: int64(len(items))}, nil
}

type smokeResultEvidenceStore struct {
	mu              sync.RWMutex
	fixtureIP       string
	fixturePort     int
	hostPorts       map[int][]scanapp.HostPortEvidence
	websites        map[int][]snapshotapp.WebsiteSnapshotItem
	vulnerabilities map[int][]snapshotapp.VulnerabilitySnapshotItem
}

func newSmokeResultEvidenceStore(cfg smokeConfig) *smokeResultEvidenceStore {
	return &smokeResultEvidenceStore{
		fixtureIP: cfg.fixtureIP, fixturePort: cfg.fixturePort,
		hostPorts:       make(map[int][]scanapp.HostPortEvidence),
		websites:        make(map[int][]snapshotapp.WebsiteSnapshotItem),
		vulnerabilities: make(map[int][]snapshotapp.VulnerabilitySnapshotItem),
	}
}

func (store *smokeResultEvidenceStore) seedWebsiteURL(scanID int, url string) error {
	if store == nil || scanID <= 0 || url == "" {
		return errors.New("smoke Website seed is invalid")
	}
	store.mu.Lock()
	store.websites[scanID] = append(store.websites[scanID], snapshotapp.WebsiteSnapshotItem{URL: url, Host: store.fixtureIP})
	store.mu.Unlock()
	return nil
}

func (store *smokeResultEvidenceStore) vulnerabilityCount(scanID int) int {
	if store == nil || scanID <= 0 {
		return 0
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	return len(store.vulnerabilities[scanID])
}

func (store *smokeResultEvidenceStore) ForEachHostPortByScanID(ctx context.Context, scanID int, visit func(scanapp.HostPortEvidence) error) error {
	if store == nil || ctx == nil || scanID <= 0 || visit == nil {
		return errors.New("smoke HostPort cursor is invalid")
	}
	store.mu.RLock()
	values := append([]scanapp.HostPortEvidence(nil), store.hostPorts[scanID]...)
	store.mu.RUnlock()
	sort.Slice(values, func(left, right int) bool {
		leftHost := values[left].Host
		if leftHost == "" {
			leftHost = values[left].IP
		}
		rightHost := values[right].Host
		if rightHost == "" {
			rightHost = values[right].IP
		}
		if leftHost == rightHost {
			if values[left].IP != values[right].IP {
				return values[left].IP < values[right].IP
			}
			return values[left].Port < values[right].Port
		}
		return leftHost < rightHost
	})
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return err
		}
		host := value.Host
		if host == "" {
			host = value.IP
		}
		if err := visit(value); err != nil {
			return err
		}
	}
	return nil
}

// ForEachWebsiteURLByScanID exposes the finalized Website URL projection to
// the production artifact resolver. The smoke store keeps the same stable URL
// order as the result snapshot and deliberately performs no baseline or URL
// repair here; the resolver must consume the recorded facts as-is.
func (store *smokeResultEvidenceStore) ForEachWebsiteURLByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	if store == nil || ctx == nil || scanID <= 0 || visit == nil {
		return errors.New("smoke Website URL cursor is invalid")
	}
	store.mu.RLock()
	values := append([]snapshotapp.WebsiteSnapshotItem(nil), store.websites[scanID]...)
	store.mu.RUnlock()
	sort.SliceStable(values, func(left, right int) bool { return values[left].URL < values[right].URL })
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return err
		}
		if value.URL == "" {
			return errors.New("smoke Website URL snapshot contains an empty URL")
		}
		if err := visit(value.URL); err != nil {
			return err
		}
	}
	return nil
}

// ForEachEndpointURLByScanID reuses the smoke URL evidence as a narrow
// Endpoint projection. Production storage uses the dedicated Endpoint
// snapshot repository; this test store only needs a deterministic URL stream.
func (store *smokeResultEvidenceStore) ForEachEndpointURLByScanID(ctx context.Context, scanID int, visit func(string) error) error {
	return store.ForEachWebsiteURLByScanID(ctx, scanID, visit)
}

type smokeHostPortMaterializer struct {
	store *smokeResultEvidenceStore
}

func (materializer *smokeHostPortMaterializer) SaveAndSyncContext(_ context.Context, scanID int, _ int, items []snapshotapp.HostPortSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	if materializer == nil || materializer.store == nil || len(items) == 0 {
		return snapshotapp.MaterializationSummary{}, errors.New("smoke HostPort result is empty")
	}
	matched := false
	values := make([]scanapp.HostPortEvidence, 0, len(items))
	for _, item := range items {
		if item.Port == materializer.store.fixturePort && (item.IP == materializer.store.fixtureIP || item.Host == materializer.store.fixtureIP) {
			matched = true
		}
		values = append(values, scanapp.HostPortEvidence{Host: item.Host, IP: item.IP, Port: item.Port})
	}
	if !matched {
		return snapshotapp.MaterializationSummary{}, fmt.Errorf("smoke HostPort result did not contain %s:%d", materializer.store.fixtureIP, materializer.store.fixturePort)
	}
	materializer.store.mu.Lock()
	materializer.store.hostPorts[scanID] = append(materializer.store.hostPorts[scanID], values...)
	materializer.store.mu.Unlock()
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: int64(len(items)), AssetCount: int64(len(items))}, nil
}

type smokeWebsiteMaterializer struct{ store *smokeResultEvidenceStore }

func (materializer *smokeWebsiteMaterializer) SaveAndSyncContext(_ context.Context, scanID int, _ int, items []snapshotapp.WebsiteSnapshotItem) (snapshotapp.MaterializationSummary, error) {
	if materializer == nil || materializer.store == nil || len(items) == 0 {
		return snapshotapp.MaterializationSummary{}, errors.New("smoke Website result is empty")
	}
	expectedURL := fmt.Sprintf("http://%s:%d", materializer.store.fixtureIP, materializer.store.fixturePort)
	matched := false
	for _, item := range items {
		if item.URL == expectedURL && item.Host == materializer.store.fixtureIP && item.StatusCode != nil && *item.StatusCode == 200 {
			matched = true
		}
	}
	if !matched {
		return snapshotapp.MaterializationSummary{}, fmt.Errorf("smoke Website result did not contain exact fixture response %s host=%s status=200", expectedURL, materializer.store.fixtureIP)
	}
	materializer.store.mu.Lock()
	materializer.store.websites[scanID] = append(materializer.store.websites[scanID], items...)
	materializer.store.mu.Unlock()
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: int64(len(items)), AssetCount: int64(len(items))}, nil
}

type smokeVulnerabilityMaterializer struct{ store *smokeResultEvidenceStore }

func (materializer *smokeVulnerabilityMaterializer) SaveResultBatchContext(_ context.Context, scanID int, _ int, items []snapshotapp.VulnerabilitySnapshotItem) (snapshotapp.MaterializationSummary, error) {
	if materializer == nil || materializer.store == nil || scanID <= 0 || len(items) == 0 {
		return snapshotapp.MaterializationSummary{}, errors.New("smoke Nuclei vulnerability result is empty")
	}
	expectedURL := fmt.Sprintf("http://%s:%d", materializer.store.fixtureIP, materializer.store.fixturePort)
	for _, item := range items {
		if item.URL != expectedURL || item.Source != "nuclei" || item.VulnType == "" ||
			results.NormalizeVulnerabilitySeverity(item.Severity) != item.Severity || item.CVSSScore == nil ||
			item.CVSSScore.LessThan(decimal.Zero) || item.CVSSScore.GreaterThan(decimal.NewFromInt(10)) || item.RawOutput == nil {
			return snapshotapp.MaterializationSummary{}, fmt.Errorf("smoke Nuclei vulnerability is outside the fixture contract")
		}
	}
	materializer.store.mu.Lock()
	materializer.store.vulnerabilities[scanID] = append(materializer.store.vulnerabilities[scanID], items...)
	materializer.store.mu.Unlock()
	return snapshotapp.MaterializationSummary{ReceivedItems: len(items), SnapshotCount: int64(len(items)), AssetCount: int64(len(items))}, nil
}

type smokeSummaryUpdater struct{}

func (*smokeSummaryUpdater) RefreshScanResultSummary(context.Context, int, int) error { return nil }

// The cut-smoke Server keeps result evidence in memory. It still supplies the
// same Facade transaction seam as the production Server, while database-backed
// tests exercise the durable execution fence itself.
type smokeResultMaterializationCoordinator struct{}

func (smokeResultMaterializationCoordinator) Materialize(ctx context.Context, _ resultingestapp.ResultMaterializationScope, persist func(context.Context) error) error {
	return persist(ctx)
}

type smokeResultIngest struct {
	bridge *smokeTaskBridge
	inner  *resultingestapp.ResultIngestFacade
}

func (ingest *smokeResultIngest) Ingest(ctx context.Context, command resultingestapp.ResultIngestCommand) (resultingestapp.ResultIngestOutcome, error) {
	if ingest == nil || ingest.bridge == nil || ingest.inner == nil {
		return resultingestapp.ResultIngestOutcome{}, errors.New("result ingest is unavailable")
	}
	ingest.bridge.mu.Lock()
	record := ingest.bridge.byTaskID[command.TaskID]
	validScope := record != nil && record.scanID == command.ScanID && record.targetID == command.TargetID && record.assignmentDelivered && !record.terminalAcked
	validResult := record != nil && ((record.scenario == smokeScenarioPortSuccess && record.engineID == enginePortID && command.ResultType == results.ResultKindAssetHostPort) ||
		(record.scenario == smokeScenarioWebsiteSuccess && record.engineID == engineWebsiteID && command.ResultType == results.ResultKindAssetWebsite) ||
		((record.scenario == smokeScenarioNucleiHit || record.scenario == smokeScenarioNucleiAckFailure || record.scenario == smokeScenarioNucleiMixed) && record.engineID == engineNucleiID && command.ResultType == results.ResultKindAssetVulnerability))
	ingest.bridge.mu.Unlock()
	if !validScope || !validResult || len(command.Items) == 0 {
		return resultingestapp.ResultIngestOutcome{}, errors.New("result batch is outside the smoke success scope")
	}
	for _, item := range command.Items {
		if len(item) == 0 {
			return resultingestapp.ResultIngestOutcome{}, errors.New("result batch contains an empty item")
		}
	}
	outcome, err := ingest.inner.Ingest(ctx, command)
	if err != nil {
		return outcome, err
	}
	matched := outcome.ReceivedItems > 0 && outcome.SnapshotCount > 0
	ingest.bridge.observeResult(command.TaskID, len(command.Items), matched)
	return outcome, nil
}

func newSmokeResultFacade(store *smokeResultEvidenceStore) *resultingestapp.ResultIngestFacade {
	return resultingestapp.NewResultIngestFacade(resultingestapp.ResultIngestFacadeDependencies{
		Subdomains:      &smokeSubdomainMaterializer{},
		HostPorts:       &smokeHostPortMaterializer{store: store},
		Websites:        &smokeWebsiteMaterializer{store: store},
		Vulnerabilities: &smokeVulnerabilityMaterializer{store: store},
		ScanSummary:     &smokeSummaryUpdater{},
		Materialization: smokeResultMaterializationCoordinator{},
	})
}
