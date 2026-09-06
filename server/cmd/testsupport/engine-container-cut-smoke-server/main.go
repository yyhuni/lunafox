// Command engine-container-cut-smoke-server is the Server side of the
// implementation-cut smoke harness.  The harness deliberately lives outside
// production bootstrap wiring: it owns run-scoped fixtures and a control-plane
// bridge while exercising the real package cache, ScanCreateService, and scan
// repositories.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/yyhuni/lunafox/contracts/ociartifact"
	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scanwiring "github.com/yyhuni/lunafox/server/internal/bootstrap/wiring/scan"
	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentdata"
	"github.com/yyhuni/lunafox/server/internal/grpc/agentserver"
	nucleipocdomain "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type smokeNucleiTemplateSource struct {
	bridge *smokeTaskBridge
}

func smokeNucleiTemplateContent(record smokePlanRecordSnapshot) string {
	matchStatus := 200
	if record.scenario == smokeScenarioNucleiZero {
		matchStatus = 418
	}
	paths := "      - \"{{BaseURL}}\"\n"
	if record.scenario == smokeScenarioNucleiAckFailure {
		var pathList strings.Builder
		pathList.WriteString("      - \"{{BaseURL}}\"\n")
		// Keep the baseline-first contract observable without making every
		// unreachable HTTPS baseline consume the whole task budget before the
		// finalized Website URL is scheduled. The delayed paths still keep the
		// process alive after the first acknowledged finding.
		for index := 1; index < 8; index++ {
			fmt.Fprintf(&pathList, "      - \"{{BaseURL}}/smoke-%d\"\n", index)
		}
		paths = pathList.String()
	}
	templateID := "smoke-nuclei-" + string(record.scenario)
	matcher := fmt.Sprintf("      - type: status\n        status:\n          - %d\n", matchStatus)
	return fmt.Sprintf("id: %s\ninfo:\n  name: LunaFox smoke Nuclei finding\n  author: lunafox\n  severity: medium\n  classification:\n    cvss-score: 5.0\nhttp:\n  - method: GET\n    path:\n%s    matchers:\n%s", templateID, paths, matcher)
}

func (source smokeNucleiTemplateSource) ListEnabledForExecution(ctx context.Context) ([]nucleipocdomain.POC, error) {
	if ctx == nil {
		return nil, errors.New("Nuclei template context is required")
	}
	task, ok := agentdata.NucleiTemplateTaskFromContext(ctx)
	if !ok {
		return nil, errors.New("Nuclei template task scope is required")
	}
	_, taskID, err := resourcenames.ParseTask(task)
	if err != nil {
		return nil, fmt.Errorf("parse Nuclei template task: %w", err)
	}
	if source.bridge == nil {
		return nil, errors.New("Nuclei template smoke bridge is required")
	}
	record, ok := source.bridge.recordForTaskID(taskID)
	if !ok {
		return nil, fmt.Errorf("Nuclei template task %q is not registered", task)
	}
	if record.scenario == smokeScenarioNucleiNoTemplates {
		return []nucleipocdomain.POC{}, nil
	}
	if record.engineID != engineNucleiID {
		return nil, fmt.Errorf("Nuclei template task %q has engine %q", task, record.engineID)
	}
	templateID := "smoke-nuclei-" + string(record.scenario)
	content := smokeNucleiTemplateContent(record)
	digest := sha256.Sum256([]byte(content))
	return []nucleipocdomain.POC{{
		TemplateID: templateID, RelativePath: "smoke/" + templateID + ".yaml",
		Content: content, ContentSHA256: hex.EncodeToString(digest[:]), IsEnabled: true,
	}}, nil
}

const smokeEvidenceSchema = "lunafox.engine-container-cut-smoke.server-evidence.v11"

const (
	smokeEngineExitFailureKind        = "engine_exit_failed"
	smokeEngineExitFailureMessage     = "The Engine exited unsuccessfully."
	smokeResultProtocolFailureKind    = "result_protocol_failed"
	smokeResultProtocolFailureMessage = "The Engine returned invalid result output."
	smokeTaskTimeoutFailureKind       = "task_timeout"
	smokeTaskTimeoutFailureMessage    = "The task exceeded its execution time limit."
	smokeConfigHashFailureKind        = "config_resource_hash_failed"
	smokeConfigHashFailureMessage     = "The config resource failed integrity validation."
	smokeNoTemplatesFailureKind       = "no_enabled_templates"
	smokeNoTemplatesFailureMessage    = "The runtime artifact required by this task is unavailable."
)

type smokeScenarioEvidence struct {
	Passed           bool                    `json:"passed"`
	Terminal         string                  `json:"terminal,omitempty"`
	FailureClass     string                  `json:"failureClass,omitempty"`
	Diagnostics      smokeDiagnosticEvidence `json:"diagnostics"`
	Artifact         smokeArtifactEvidence   `json:"artifact"`
	ResultBatches    int                     `json:"resultBatches"`
	ResultItems      int                     `json:"resultItems"`
	ResultMatched    bool                    `json:"resultMatched"`
	FindingCount     int                     `json:"findingCount"`
	MalformedRecords uint64                  `json:"malformedRecords"`
	InvalidRecords   uint64                  `json:"invalidRecords"`
}

type smokeArtifactEvidence struct {
	Attempts             uint64 `json:"attempts"`
	Streams              uint64 `json:"streams"`
	Records              uint64 `json:"records"`
	Bytes                uint64 `json:"bytes"`
	UnavailableFailures  uint64 `json:"unavailableFailures"`
	IntegrityCorruptions uint64 `json:"integrityCorruptions"`
}

// smokeDiagnosticEvidence is the bounded end-to-end assertion view. It keeps
// only the persisted terminal snapshot fields needed by the smoke verifier;
// it must never become a second result or log store.
type smokeDiagnosticEvidence struct {
	Present               bool   `json:"present"`
	Valid                 bool   `json:"valid"`
	CompatibilityRevision string `json:"compatibilityRevision,omitempty"`
	Availability          string `json:"availability,omitempty"`
	ResultState           string `json:"resultState,omitempty"`
	FailedStage           string `json:"failedStage,omitempty"`
	ErrorType             string `json:"errorType,omitempty"`
	ResultTypeCount       int    `json:"resultTypeCount"`
	ReceivedItems         uint64 `json:"receivedItems"`
	EncodedItems          uint64 `json:"encodedItems"`
	SubmittedItems        uint64 `json:"submittedItems"`
	AcknowledgedItems     uint64 `json:"acknowledgedItems"`
	SubmittedBatches      uint64 `json:"submittedBatches"`
	AcknowledgedBatches   uint64 `json:"acknowledgedBatches"`
	UnresolvedBatches     uint64 `json:"unresolvedBatches"`
}

// smokeEvidence is intentionally a concrete schema.  The CI verifier reads
// these fields directly, so adding a broad map here would hide missing smoke
// observations behind a successful process exit.
type smokeEvidence struct {
	SchemaVersion string `json:"schemaVersion"`
	Passed        bool   `json:"passed"`
	NucleiFixture struct {
		Enabled      bool   `json:"enabled"`
		SourceRef    string `json:"sourceRef,omitempty"`
		ImageRef     string `json:"imageRef,omitempty"`
		SourceDigest string `json:"sourceDigest,omitempty"`
		ImageDigest  string `json:"imageDigest,omitempty"`
	} `json:"nucleiFixture"`
	PackageCacheColdStart struct {
		ObservedEmpty      bool     `json:"observedEmpty"`
		InstalledPackages  int      `json:"installedPackages"`
		InstalledEngineIDs []string `json:"installedEngineIds"`
		ReceiptMode        string   `json:"receiptMode"`
	} `json:"packageCacheColdStart"`
	ScenarioEngineIDs []string `json:"scenarioEngineIds"`
	Protocol          struct {
		ScanCreate      smokeScanCreateEvidence     `json:"scanCreate"`
		ServerInputs    smokeInputSemanticsEvidence `json:"serverInputs"`
		Workflow        smokeWorkflowEvidence       `json:"workflow"`
		TaskAssignOneof struct {
			PlanCount             int  `json:"planCount"`
			NoTaskCount           int  `json:"noTaskCount"`
			ClaimRecoveryFailures int  `json:"claimRecoveryFailures"`
			ClaimRecoveryReplays  int  `json:"claimRecoveryReplays"`
			ProductionCalls       int  `json:"productionCalls"`
			RepositoryBacked      bool `json:"repositoryBacked"`
		} `json:"taskAssignOneof"`
		SessionReconnect struct {
			Connections int  `json:"connections"`
			Passed      bool `json:"passed"`
		} `json:"sessionReconnect"`
		SessionFence  smokeSessionFenceEvidence `json:"sessionFence"`
		RuntimeChains struct {
			IPPortToWebsite   bool `json:"ipPortToWebsite"`
			CIDRPortToWebsite bool `json:"cidrPortToWebsite"`
		} `json:"runtimeChains"`
		ArtifactStreams struct {
			Subdomains       int `json:"subdomains"`
			HostPorts        int `json:"hostPorts"`
			ConfigResource   int `json:"configResource"`
			PlatformResource int `json:"platformResource"`
			NucleiTemplates  int `json:"nucleiTemplates"`
		} `json:"artifactStreams"`
		ArtifactTransfers struct {
			ProductionResolver bool `json:"productionResolver"`
			SubdomainsRetry    struct {
				Attempts            uint64 `json:"attempts"`
				UnavailableFailures uint64 `json:"unavailableFailures"`
			} `json:"subdomainsRetry"`
			CIDRMultiChunk struct {
				Records uint64 `json:"records"`
				Bytes   uint64 `json:"bytes"`
			} `json:"cidrMultiChunk"`
			IntegrityFailure struct {
				Attempts    uint64 `json:"attempts"`
				Corruptions uint64 `json:"corruptions"`
			} `json:"integrityFailure"`
			WordlistCacheHit struct {
				ConfigStreams   uint64 `json:"configStreams"`
				ProviderStreams uint64 `json:"providerStreams"`
			} `json:"wordlistCacheHit"`
		} `json:"artifactTransfers"`
		EngineHandlers struct {
			SubdomainDiscovery  int `json:"subdomainDiscovery"`
			PortScan            int `json:"portScan"`
			WebsiteDiscovery    int `json:"websiteDiscovery"`
			NucleiVulnerability int `json:"nucleiVulnerability"`
		} `json:"engineHandlers"`
		ProgressMessages int `json:"progressMessages"`
		ResultBatches    int `json:"resultBatches"`
		TerminalAcks     struct {
			Count            int  `json:"count"`
			RecoveryFailures int  `json:"recoveryFailures"`
			RecoveryReplays  int  `json:"recoveryReplays"`
			ProductionCalls  int  `json:"productionCalls"`
			RepositoryBacked bool `json:"repositoryBacked"`
		} `json:"terminalAcks"`
	} `json:"protocol"`
	Scenarios struct {
		Success             smokeScenarioEvidence `json:"success"`
		WebsiteSuccess      smokeScenarioEvidence `json:"websiteSuccess"`
		PortEmpty           smokeScenarioEvidence `json:"portEmpty"`
		EmptyInput          smokeScenarioEvidence `json:"emptyInput"`
		ArtifactIntegrity   smokeScenarioEvidence `json:"artifactIntegrity"`
		Failure             smokeScenarioEvidence `json:"failure"`
		WordlistCacheHit    smokeScenarioEvidence `json:"wordlistCacheHit"`
		Cancel              smokeScenarioEvidence `json:"cancel"`
		Timeout             smokeScenarioEvidence `json:"timeout"`
		NucleiHit           smokeScenarioEvidence `json:"nucleiHit"`
		NucleiZeroOutput    smokeScenarioEvidence `json:"nucleiZeroOutput"`
		NucleiNoTemplates   smokeScenarioEvidence `json:"nucleiNoEnabledTemplates"`
		NucleiCancel        smokeScenarioEvidence `json:"nucleiCancel"`
		NucleiTimeout       smokeScenarioEvidence `json:"nucleiTimeout"`
		NucleiAckFailure    smokeScenarioEvidence `json:"nucleiAckThenFailed"`
		NucleiMalformed     smokeScenarioEvidence `json:"nucleiMalformedJsonl"`
		NucleiMixed         smokeScenarioEvidence `json:"nucleiMixedOutput"`
		NucleiNonzero       smokeScenarioEvidence `json:"nucleiNonzeroExit"`
		TerminalAckRecovery smokeScenarioEvidence `json:"terminalAckRecovery"`
		Cleanup             smokeScenarioEvidence `json:"cleanup"`
	} `json:"scenarios"`
}

func newSmokeEvidence(installedEngineIDs []string, receiptMode string, runtime *smokeRuntime) smokeEvidence {
	var evidence smokeEvidence
	evidence.SchemaVersion = smokeEvidenceSchema
	if runtime != nil {
		evidence.NucleiFixture.Enabled = runtime.cfg.nucleiFixtureImageRef != ""
		evidence.NucleiFixture.SourceRef = runtime.cfg.nucleiFixtureSourceRef
		evidence.NucleiFixture.ImageRef = runtime.cfg.nucleiFixtureImageRef
		if source, err := ociartifact.ParseDigestReference(runtime.cfg.nucleiFixtureSourceRef); err == nil {
			evidence.NucleiFixture.SourceDigest = source.Digest
		}
		if image, err := ociartifact.ParseDigestReference(runtime.cfg.nucleiFixtureImageRef); err == nil {
			evidence.NucleiFixture.ImageDigest = image.Digest
		}
	}
	evidence.PackageCacheColdStart.ObservedEmpty = true
	// Sorting makes the receipt-derived set stable for the outer smoke verifier.
	evidence.PackageCacheColdStart.InstalledEngineIDs = append([]string(nil), installedEngineIDs...)
	sort.Strings(evidence.PackageCacheColdStart.InstalledEngineIDs)
	evidence.PackageCacheColdStart.InstalledPackages = len(evidence.PackageCacheColdStart.InstalledEngineIDs)
	evidence.PackageCacheColdStart.ReceiptMode = receiptMode
	if runtime == nil || runtime.bridge == nil {
		return evidence
	}
	evidence.Protocol.ScanCreate = runtime.scanCreateEvidence
	evidence.Protocol.ServerInputs = runtime.inputEvidence
	evidence.Protocol.Workflow = runtime.workflowEvidence
	counters := runtime.bridge.countersSnapshot()
	records := runtime.bridge.recordSnapshots()
	evidence.ScenarioEngineIDs = sortedScenarioEngineIDs(records)
	byScenario := make(map[smokeScenario]smokePlanRecordSnapshot, len(records))
	for _, record := range records {
		byScenario[record.scenario] = record
	}
	terminalFor := func(record smokePlanRecordSnapshot) string {
		if record.terminal == nil {
			return ""
		}
		return record.terminal.result
	}
	failureFor := func(record smokePlanRecordSnapshot) string {
		if record.terminal == nil {
			return ""
		}
		return record.terminal.failureKind
	}
	passedTerminal := func(record smokePlanRecordSnapshot, state string) bool {
		return record.assignmentDelivered && record.terminalAcked && terminalFor(record) == state
	}
	diagnosticsFor := func(record smokePlanRecordSnapshot) smokeDiagnosticEvidence {
		return smokeDiagnosticEvidenceFor(record)
	}
	evidence.Protocol.TaskAssignOneof.PlanCount = counters.planAssignments
	evidence.Protocol.TaskAssignOneof.NoTaskCount = counters.noTaskAssignments
	evidence.Protocol.TaskAssignOneof.ClaimRecoveryFailures = counters.claimRecoveryFailures
	evidence.Protocol.TaskAssignOneof.ClaimRecoveryReplays = counters.claimRecoveryReplays
	evidence.Protocol.TaskAssignOneof.ProductionCalls = counters.productionClaimCalls
	evidence.Protocol.TaskAssignOneof.RepositoryBacked = runtime.bridge.production != nil && counters.productionClaimCalls > 0
	evidence.Protocol.SessionReconnect.Connections, evidence.Protocol.SessionReconnect.Passed = runtime.authority.sameSessionReconnectEvidence()
	evidence.Protocol.SessionFence = runtime.sessionFenceEvidence
	evidence.Protocol.ArtifactStreams.Subdomains = counters.artifactHost
	evidence.Protocol.ArtifactStreams.HostPorts = counters.artifactWeb
	evidence.Protocol.ArtifactStreams.ConfigResource = counters.artifactConfig
	evidence.Protocol.ArtifactStreams.PlatformResource = counters.artifactPlatform
	evidence.Protocol.ArtifactStreams.NucleiTemplates = counters.artifactNucleiTemplates
	evidence.Protocol.ProgressMessages = counters.progressMessages
	evidence.Protocol.ResultBatches = counters.resultBatches
	evidence.Protocol.TerminalAcks.Count = counters.terminalAcks
	evidence.Protocol.TerminalAcks.RecoveryFailures = counters.terminalRecoveryFails
	evidence.Protocol.TerminalAcks.RecoveryReplays = counters.recoveryReplays
	evidence.Protocol.TerminalAcks.ProductionCalls = counters.productionTerminalCalls
	evidence.Protocol.TerminalAcks.RepositoryBacked = runtime.bridge.production != nil && counters.productionTerminalCalls > 0
	for _, record := range records {
		if !record.assignmentDelivered || record.terminal == nil || record.scenario == smokeScenarioArtifactIntegrity || record.scenario == smokeScenarioNucleiNoTemplates {
			continue
		}
		switch record.engineID {
		case engineSubdomainID:
			evidence.Protocol.EngineHandlers.SubdomainDiscovery++
		case enginePortID:
			evidence.Protocol.EngineHandlers.PortScan++
		case engineWebsiteID:
			evidence.Protocol.EngineHandlers.WebsiteDiscovery++
		case engineNucleiID:
			evidence.Protocol.EngineHandlers.NucleiVulnerability++
		}
	}
	success := byScenario[smokeScenarioPortSuccess]
	websiteSuccess := byScenario[smokeScenarioWebsiteSuccess]
	portEmpty := byScenario[smokeScenarioPortEmpty]
	emptyInput := byScenario[smokeScenarioWebsiteEmpty]
	evidence.Protocol.RuntimeChains.IPPortToWebsite = success.scanID > 0 && success.scanID == websiteSuccess.scanID
	evidence.Protocol.RuntimeChains.CIDRPortToWebsite = portEmpty.scanID > 0 && portEmpty.scanID == emptyInput.scanID
	artifactIntegrity := byScenario[smokeScenarioArtifactIntegrity]
	failure := byScenario[smokeScenarioSubdomainFail]
	wordlistCacheHit := byScenario[smokeScenarioWordlistCacheHit]
	cancel := byScenario[smokeScenarioPortCancel]
	timeout := byScenario[smokeScenarioPortTimeout]
	nucleiHit := byScenario[smokeScenarioNucleiHit]
	nucleiZero := byScenario[smokeScenarioNucleiZero]
	nucleiNoTemplates := byScenario[smokeScenarioNucleiNoTemplates]
	nucleiCancel := byScenario[smokeScenarioNucleiCancel]
	nucleiTimeout := byScenario[smokeScenarioNucleiTimeout]
	nucleiAckFailure := byScenario[smokeScenarioNucleiAckFailure]
	nucleiMalformed := byScenario[smokeScenarioNucleiMalformed]
	nucleiMixed := byScenario[smokeScenarioNucleiMixed]
	nucleiNonzero := byScenario[smokeScenarioNucleiNonzero]
	successDiagnostics := diagnosticsFor(success)
	websiteSuccessDiagnostics := diagnosticsFor(websiteSuccess)
	portEmptyDiagnostics := diagnosticsFor(portEmpty)
	emptyInputDiagnostics := diagnosticsFor(emptyInput)
	artifactIntegrityDiagnostics := diagnosticsFor(artifactIntegrity)
	failureDiagnostics := diagnosticsFor(failure)
	wordlistCacheHitDiagnostics := diagnosticsFor(wordlistCacheHit)
	cancelDiagnostics := diagnosticsFor(cancel)
	timeoutDiagnostics := diagnosticsFor(timeout)
	nucleiHitDiagnostics := diagnosticsFor(nucleiHit)
	nucleiZeroDiagnostics := diagnosticsFor(nucleiZero)
	nucleiNoTemplatesDiagnostics := diagnosticsFor(nucleiNoTemplates)
	nucleiCancelDiagnostics := diagnosticsFor(nucleiCancel)
	nucleiTimeoutDiagnostics := diagnosticsFor(nucleiTimeout)
	nucleiAckFailureDiagnostics := diagnosticsFor(nucleiAckFailure)
	nucleiMalformedDiagnostics := diagnosticsFor(nucleiMalformed)
	nucleiMixedDiagnostics := diagnosticsFor(nucleiMixed)
	nucleiNonzeroDiagnostics := diagnosticsFor(nucleiNonzero)
	nucleiArtifact := func(record smokePlanRecordSnapshot) smokeArtifactEvidence {
		return smokeArtifactEvidence{
			Attempts: record.nucleiTemplateArtifact.attempts, Streams: record.nucleiTemplateArtifact.streams,
			Records: record.nucleiTemplateArtifact.records, Bytes: record.nucleiTemplateArtifact.bytes,
			UnavailableFailures:  record.nucleiTemplateArtifact.unavailableFailures,
			IntegrityCorruptions: record.nucleiTemplateArtifact.integrityCorruptions,
		}
	}
	nucleiFindingCount := func(record smokePlanRecordSnapshot) int {
		if runtime.resultEvidence == nil {
			return 0
		}
		return runtime.resultEvidence.vulnerabilityCount(record.scanID)
	}
	nucleiInvalidCounts := func(record smokePlanRecordSnapshot) (uint64, uint64) {
		return record.malformedRecords, record.invalidRecords
	}
	evidence.Scenarios.Success = smokeScenarioEvidence{
		Passed:      evidence.Protocol.RuntimeChains.IPPortToWebsite && passedTerminal(success, "succeeded") && success.hostArtifact.attempts == 2 && success.hostArtifact.unavailableFailures == 1 && success.hostArtifact.streams == 1 && success.hostArtifact.records == 1 && success.hostArtifact.bytes > 0 && success.progressMessages > 0 && success.resultMatched && success.resultBatches > 0 && smokeConfirmedResultDiagnostics(successDiagnostics),
		Terminal:    terminalFor(success),
		Diagnostics: successDiagnostics,
	}
	evidence.Scenarios.WebsiteSuccess = smokeScenarioEvidence{
		Passed:      passedTerminal(websiteSuccess, "succeeded") && websiteSuccess.webArtifact.streams == 1 && websiteSuccess.webArtifact.records > 0 && websiteSuccess.webArtifact.bytes > 0 && websiteSuccess.progressMessages > 0 && websiteSuccess.resultMatched && websiteSuccess.resultBatches > 0 && smokeConfirmedResultDiagnostics(websiteSuccessDiagnostics),
		Terminal:    terminalFor(websiteSuccess),
		Diagnostics: websiteSuccessDiagnostics,
	}
	evidence.Scenarios.PortEmpty = smokeScenarioEvidence{
		Passed:      evidence.Protocol.RuntimeChains.CIDRPortToWebsite && passedTerminal(portEmpty, "succeeded") && portEmpty.hostArtifact.streams == 1 && portEmpty.hostArtifact.records == 0 && portEmpty.hostArtifact.bytes == 0 && portEmpty.resultBatches == 0 && smokeZeroResultDiagnostics(portEmptyDiagnostics),
		Terminal:    terminalFor(portEmpty),
		Diagnostics: portEmptyDiagnostics,
	}
	evidence.Scenarios.EmptyInput = smokeScenarioEvidence{
		Passed:      passedTerminal(emptyInput, "succeeded") && emptyInput.webArtifact.streams == 1 && emptyInput.webArtifact.records == 0 && emptyInput.webArtifact.bytes == 0 && emptyInput.progressMessages > 0 && emptyInput.resultBatches == 0 && smokeZeroResultDiagnostics(emptyInputDiagnostics),
		Terminal:    terminalFor(emptyInput),
		Diagnostics: emptyInputDiagnostics,
	}
	evidence.Scenarios.Failure = smokeScenarioEvidence{
		Passed:       passedTerminal(failure, "failed") && failure.configArtifact.streams == 2 && failure.configArtifact.records > 0 && failure.configArtifact.bytes > 0 && failure.platformArtifact.streams == 1 && failure.platformArtifact.bytes > 0 && failureFor(failure) == smokeEngineExitFailureKind && failure.terminal.failureMessage == smokeEngineExitFailureMessage && smokeNoConfirmedFailureDiagnostics(failureDiagnostics),
		Terminal:     terminalFor(failure),
		FailureClass: failureFor(failure),
		Diagnostics:  failureDiagnostics,
	}
	evidence.Scenarios.ArtifactIntegrity = smokeScenarioEvidence{
		Passed: passedTerminal(artifactIntegrity, "failed") && artifactIntegrity.configArtifact.attempts == 1 &&
			artifactIntegrity.configArtifact.integrityCorruptions == 1 && failureFor(artifactIntegrity) == smokeConfigHashFailureKind &&
			artifactIntegrity.terminal.failureMessage == smokeConfigHashFailureMessage && smokeUnavailableDiagnostics(artifactIntegrityDiagnostics),
		Terminal:     terminalFor(artifactIntegrity),
		FailureClass: failureFor(artifactIntegrity),
		Diagnostics:  artifactIntegrityDiagnostics,
	}
	evidence.Scenarios.WordlistCacheHit = smokeScenarioEvidence{
		Passed: passedTerminal(wordlistCacheHit, "failed") && wordlistCacheHit.configArtifact.streams == 0 &&
			wordlistCacheHit.platformArtifact.streams == 1 && failureFor(wordlistCacheHit) == smokeEngineExitFailureKind && smokeNoConfirmedFailureDiagnostics(wordlistCacheHitDiagnostics),
		Terminal:     terminalFor(wordlistCacheHit),
		FailureClass: failureFor(wordlistCacheHit),
		Diagnostics:  wordlistCacheHitDiagnostics,
	}
	// Cancellation and budget expiration revoke the task-private UDS before
	// Docker is stopped, so these terminal facts deliberately preserve missing
	// Engine evidence as unavailable rather than fabricating zero-result data.
	evidence.Scenarios.Cancel = smokeScenarioEvidence{
		Passed:      passedTerminal(cancel, "cancelled") && cancel.cancelIssued && cancel.progressMessages > 0 && cancel.hostArtifact.streams == 1 && smokeUnavailableDiagnostics(cancelDiagnostics),
		Terminal:    terminalFor(cancel),
		Diagnostics: cancelDiagnostics,
	}
	evidence.Scenarios.Timeout = smokeScenarioEvidence{
		Passed:       passedTerminal(timeout, "failed") && timeout.progressMessages > 0 && timeout.hostArtifact.streams == 1 && failureFor(timeout) == smokeTaskTimeoutFailureKind && timeout.terminal.failureMessage == smokeTaskTimeoutFailureMessage && smokeUnavailableDiagnostics(timeoutDiagnostics),
		Terminal:     terminalFor(timeout),
		FailureClass: failureFor(timeout),
		Diagnostics:  timeoutDiagnostics,
	}
	evidence.Scenarios.NucleiHit = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiHit, "succeeded") && nucleiHit.nucleiTemplateArtifact.streams == 1 && nucleiHit.nucleiTemplateArtifact.records == 1 && nucleiHit.nucleiTemplateArtifact.bytes > 0 && nucleiHit.resultBatches > 0 && nucleiHit.resultItems > 0 && nucleiHit.resultMatched && nucleiFindingCount(nucleiHit) > 0 && smokeConfirmedResultDiagnostics(nucleiHitDiagnostics),
		Terminal: terminalFor(nucleiHit), Diagnostics: nucleiHitDiagnostics, Artifact: nucleiArtifact(nucleiHit),
		ResultBatches: nucleiHit.resultBatches, ResultItems: nucleiHit.resultItems, ResultMatched: nucleiHit.resultMatched, FindingCount: nucleiFindingCount(nucleiHit),
	}
	evidence.Scenarios.NucleiZeroOutput = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiZero, "succeeded") && nucleiZero.nucleiTemplateArtifact.streams == 1 && nucleiZero.nucleiTemplateArtifact.records == 1 && nucleiZero.nucleiTemplateArtifact.bytes > 0 && nucleiZero.resultBatches == 0 && smokeZeroResultDiagnostics(nucleiZeroDiagnostics),
		Terminal: terminalFor(nucleiZero), Diagnostics: nucleiZeroDiagnostics, Artifact: nucleiArtifact(nucleiZero),
		ResultBatches: nucleiZero.resultBatches, ResultItems: nucleiZero.resultItems, ResultMatched: nucleiZero.resultMatched,
	}
	evidence.Scenarios.NucleiNoTemplates = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiNoTemplates, "failed") && nucleiNoTemplates.nucleiTemplateArtifact.attempts == 0 && nucleiNoTemplates.nucleiTemplateArtifact.streams == 0 && nucleiNoTemplates.resultBatches == 0 && failureFor(nucleiNoTemplates) == smokeNoTemplatesFailureKind,
		Terminal: terminalFor(nucleiNoTemplates), FailureClass: failureFor(nucleiNoTemplates), Diagnostics: nucleiNoTemplatesDiagnostics, Artifact: nucleiArtifact(nucleiNoTemplates),
	}
	evidence.Scenarios.NucleiCancel = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiCancel, "cancelled") && nucleiCancel.nucleiTemplateArtifact.streams == 1 && nucleiCancel.progressMessages > 0 && smokeUnavailableDiagnostics(nucleiCancelDiagnostics),
		Terminal: terminalFor(nucleiCancel), Diagnostics: nucleiCancelDiagnostics, Artifact: nucleiArtifact(nucleiCancel),
	}
	evidence.Scenarios.NucleiTimeout = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiTimeout, "failed") && nucleiTimeout.nucleiTemplateArtifact.streams == 1 && nucleiTimeout.progressMessages > 0 && failureFor(nucleiTimeout) == smokeTaskTimeoutFailureKind && smokeUnavailableDiagnostics(nucleiTimeoutDiagnostics),
		Terminal: terminalFor(nucleiTimeout), FailureClass: failureFor(nucleiTimeout), Diagnostics: nucleiTimeoutDiagnostics, Artifact: nucleiArtifact(nucleiTimeout),
	}
	evidence.Scenarios.NucleiAckFailure = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiAckFailure, "failed") && nucleiAckFailure.nucleiTemplateArtifact.streams == 1 && nucleiAckFailure.resultBatches > 0 && nucleiFindingCount(nucleiAckFailure) > 0 && failureFor(nucleiAckFailure) == smokeTaskTimeoutFailureKind && smokeUnavailableDiagnostics(nucleiAckFailureDiagnostics),
		Terminal: terminalFor(nucleiAckFailure), FailureClass: failureFor(nucleiAckFailure), Diagnostics: nucleiAckFailureDiagnostics, Artifact: nucleiArtifact(nucleiAckFailure),
		ResultBatches: nucleiAckFailure.resultBatches, ResultItems: nucleiAckFailure.resultItems, ResultMatched: nucleiAckFailure.resultMatched, FindingCount: nucleiFindingCount(nucleiAckFailure),
	}
	malformedRecords, malformedInvalid := nucleiInvalidCounts(nucleiMalformed)
	evidence.Scenarios.NucleiMalformed = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiMalformed, "failed") && nucleiMalformed.nucleiTemplateArtifact.streams == 1 && nucleiMalformed.resultBatches == 0 && failureFor(nucleiMalformed) == smokeResultProtocolFailureKind && nucleiMalformed.terminal != nil && nucleiMalformed.terminal.failureMessage == smokeResultProtocolFailureMessage && malformedRecords > 0 && smokeNoConfirmedFailureDiagnostics(nucleiMalformedDiagnostics),
		Terminal: terminalFor(nucleiMalformed), FailureClass: failureFor(nucleiMalformed), Diagnostics: nucleiMalformedDiagnostics, Artifact: nucleiArtifact(nucleiMalformed),
		MalformedRecords: malformedRecords, InvalidRecords: malformedInvalid,
	}
	mixedRecords, mixedInvalid := nucleiInvalidCounts(nucleiMixed)
	evidence.Scenarios.NucleiMixed = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiMixed, "succeeded") && nucleiMixed.nucleiTemplateArtifact.streams == 1 && nucleiMixed.resultBatches > 0 && nucleiMixed.resultItems > 0 && nucleiFindingCount(nucleiMixed) > 0 && mixedRecords > 0 && smokeConfirmedResultDiagnostics(nucleiMixedDiagnostics),
		Terminal: terminalFor(nucleiMixed), Diagnostics: nucleiMixedDiagnostics, Artifact: nucleiArtifact(nucleiMixed),
		ResultBatches: nucleiMixed.resultBatches, ResultItems: nucleiMixed.resultItems, ResultMatched: nucleiMixed.resultMatched, FindingCount: nucleiFindingCount(nucleiMixed),
		MalformedRecords: mixedRecords, InvalidRecords: mixedInvalid,
	}
	nonzeroRecords, nonzeroInvalid := nucleiInvalidCounts(nucleiNonzero)
	evidence.Scenarios.NucleiNonzero = smokeScenarioEvidence{
		Passed:   passedTerminal(nucleiNonzero, "failed") && nucleiNonzero.nucleiTemplateArtifact.streams == 1 && nucleiNonzero.resultBatches == 0 && failureFor(nucleiNonzero) == smokeEngineExitFailureKind && nucleiNonzero.terminal != nil && nucleiNonzero.terminal.failureMessage == smokeEngineExitFailureMessage && nonzeroRecords == 0 && nonzeroInvalid == 0 && smokeNoConfirmedFailureDiagnostics(nucleiNonzeroDiagnostics),
		Terminal: terminalFor(nucleiNonzero), FailureClass: failureFor(nucleiNonzero), Diagnostics: nucleiNonzeroDiagnostics, Artifact: nucleiArtifact(nucleiNonzero),
		MalformedRecords: nonzeroRecords, InvalidRecords: nonzeroInvalid,
	}
	evidence.Scenarios.TerminalAckRecovery = smokeScenarioEvidence{
		Passed:   emptyInput.terminalRecoverySeen && counters.terminalRecoveryFails == 1 && counters.recoveryReplays > 0,
		Terminal: terminalFor(emptyInput),
	}
	evidence.Scenarios.Cleanup = smokeScenarioEvidence{Passed: runtime.bridge.cleanupObserved()}
	evidence.Protocol.ArtifactTransfers.ProductionResolver = true
	evidence.Protocol.ArtifactTransfers.SubdomainsRetry.Attempts = success.hostArtifact.attempts
	evidence.Protocol.ArtifactTransfers.SubdomainsRetry.UnavailableFailures = success.hostArtifact.unavailableFailures
	evidence.Protocol.ArtifactTransfers.CIDRMultiChunk.Records = portEmpty.hostArtifact.records
	evidence.Protocol.ArtifactTransfers.CIDRMultiChunk.Bytes = portEmpty.hostArtifact.bytes
	evidence.Protocol.ArtifactTransfers.IntegrityFailure.Attempts = artifactIntegrity.configArtifact.attempts
	evidence.Protocol.ArtifactTransfers.IntegrityFailure.Corruptions = artifactIntegrity.configArtifact.integrityCorruptions
	evidence.Protocol.ArtifactTransfers.WordlistCacheHit.ConfigStreams = wordlistCacheHit.configArtifact.streams
	evidence.Protocol.ArtifactTransfers.WordlistCacheHit.ProviderStreams = wordlistCacheHit.platformArtifact.streams
	evidence.Passed = len(evidence.ScenarioEngineIDs) > 0 && evidence.Protocol.ScanCreate.Passed && evidence.Protocol.ServerInputs.Passed && evidence.Protocol.Workflow.Passed &&
		evidence.Protocol.TaskAssignOneof.PlanCount == len(scenarioOrder(runtime.cfg)) &&
		evidence.Protocol.TaskAssignOneof.NoTaskCount > 0 &&
		evidence.Protocol.TaskAssignOneof.ClaimRecoveryFailures == 1 && evidence.Protocol.TaskAssignOneof.ClaimRecoveryReplays == 1 &&
		evidence.Protocol.TaskAssignOneof.RepositoryBacked && evidence.Protocol.TaskAssignOneof.ProductionCalls >= len(scenarioOrder(runtime.cfg)) &&
		evidence.Protocol.SessionReconnect.Passed && evidence.Protocol.SessionFence.Passed && evidence.Protocol.RuntimeChains.IPPortToWebsite && evidence.Protocol.RuntimeChains.CIDRPortToWebsite &&
		evidence.Protocol.ArtifactStreams.Subdomains > 0 && evidence.Protocol.ArtifactStreams.HostPorts > 0 &&
		evidence.Protocol.ArtifactStreams.ConfigResource > 0 && evidence.Protocol.ArtifactStreams.PlatformResource > 0 &&
		evidence.Protocol.ArtifactTransfers.ProductionResolver && evidence.Protocol.ArtifactTransfers.SubdomainsRetry.Attempts == 2 &&
		evidence.Protocol.ArtifactTransfers.SubdomainsRetry.UnavailableFailures == 1 && evidence.Protocol.ArtifactTransfers.CIDRMultiChunk.Records == 0 &&
		evidence.Protocol.ArtifactTransfers.CIDRMultiChunk.Bytes == 0 && evidence.Protocol.ArtifactTransfers.IntegrityFailure.Attempts == 1 &&
		evidence.Protocol.ArtifactTransfers.IntegrityFailure.Corruptions == 1 && evidence.Protocol.ArtifactTransfers.WordlistCacheHit.ConfigStreams == 0 &&
		evidence.Protocol.ArtifactTransfers.WordlistCacheHit.ProviderStreams == 1 &&
		evidence.Protocol.EngineHandlers.SubdomainDiscovery == 2 && evidence.Protocol.EngineHandlers.PortScan == 4 &&
		evidence.Protocol.EngineHandlers.WebsiteDiscovery == 2 && evidence.Protocol.EngineHandlers.NucleiVulnerability == nucleiHandlerCount(runtime.cfg) && evidence.Protocol.ArtifactStreams.NucleiTemplates == nucleiTemplateStreamCount(runtime.cfg) && evidence.Protocol.ProgressMessages > 0 &&
		evidence.Protocol.ResultBatches > 0 && evidence.Protocol.TerminalAcks.Count == len(scenarioOrder(runtime.cfg)) &&
		evidence.Protocol.TerminalAcks.RepositoryBacked && evidence.Protocol.TerminalAcks.ProductionCalls >= len(scenarioOrder(runtime.cfg)) &&
		evidence.Protocol.TerminalAcks.RecoveryFailures == 1 && evidence.Protocol.TerminalAcks.RecoveryReplays == 1 && evidence.Scenarios.Success.Passed &&
		evidence.Scenarios.WebsiteSuccess.Passed && evidence.Scenarios.PortEmpty.Passed && evidence.Scenarios.EmptyInput.Passed &&
		evidence.Scenarios.ArtifactIntegrity.Passed && evidence.Scenarios.Failure.Passed && evidence.Scenarios.WordlistCacheHit.Passed &&
		evidence.Scenarios.Cancel.Passed && evidence.Scenarios.Timeout.Passed &&
		evidence.Scenarios.NucleiHit.Passed && evidence.Scenarios.NucleiZeroOutput.Passed && evidence.Scenarios.NucleiNoTemplates.Passed && evidence.Scenarios.NucleiCancel.Passed && evidence.Scenarios.NucleiTimeout.Passed && evidence.Scenarios.NucleiAckFailure.Passed &&
		(!evidence.NucleiFixture.Enabled || (evidence.Scenarios.NucleiMalformed.Passed && evidence.Scenarios.NucleiMixed.Passed && evidence.Scenarios.NucleiNonzero.Passed && evidence.NucleiFixture.SourceDigest != "" && evidence.NucleiFixture.ImageDigest != "" && evidence.NucleiFixture.SourceDigest != evidence.NucleiFixture.ImageDigest)) &&
		evidence.Scenarios.TerminalAckRecovery.Passed && evidence.Scenarios.Cleanup.Passed
	return evidence
}

func smokeDiagnosticEvidenceFor(record smokePlanRecordSnapshot) smokeDiagnosticEvidence {
	if record.terminal == nil || record.terminal.diagnostics == nil {
		return smokeDiagnosticEvidence{}
	}
	diagnostics := record.terminal.diagnostics
	evidence := smokeDiagnosticEvidence{
		Present:               true,
		CompatibilityRevision: diagnostics.CompatibilityRevision,
		Availability:          diagnostics.Availability,
		ResultState:           diagnostics.ResultState,
		FailedStage:           diagnostics.FailedStage,
		ErrorType:             diagnostics.ErrorType,
		ResultTypeCount:       len(diagnostics.ResultTypeWatermarks),
	}
	for _, watermark := range diagnostics.ResultTypeWatermarks {
		evidence.ReceivedItems += watermark.ReceivedItems
		evidence.EncodedItems += watermark.EncodedItems
		evidence.SubmittedItems += watermark.SubmittedItems
		evidence.AcknowledgedItems += watermark.AcknowledgedItems
		evidence.SubmittedBatches += watermark.SubmittedBatches
		evidence.AcknowledgedBatches += watermark.AcknowledgedBatches
		if watermark.SubmittedBatches >= watermark.AcknowledgedBatches {
			evidence.UnresolvedBatches += watermark.SubmittedBatches - watermark.AcknowledgedBatches
		}
	}
	evidence.Valid = smokeDiagnosticsMatchTerminal(record.terminal.result, evidence)
	return evidence
}

func smokeDiagnosticsMatchTerminal(terminal string, diagnostics smokeDiagnosticEvidence) bool {
	if !diagnostics.Present || diagnostics.CompatibilityRevision == "" {
		return false
	}
	if diagnostics.Availability == "unavailable" {
		return diagnostics.ResultState == "unknown" && diagnostics.FailedStage == "" && diagnostics.ErrorType == "" && diagnostics.ResultTypeCount == 0 &&
			diagnostics.ReceivedItems == 0 && diagnostics.EncodedItems == 0 && diagnostics.SubmittedItems == 0 && diagnostics.AcknowledgedItems == 0 &&
			diagnostics.SubmittedBatches == 0 && diagnostics.AcknowledgedBatches == 0 && diagnostics.UnresolvedBatches == 0
	}
	if diagnostics.Availability != "available" {
		return false
	}
	if terminal == "succeeded" {
		if diagnostics.FailedStage != "" || diagnostics.ErrorType != "" {
			return false
		}
		if diagnostics.UnresolvedBatches > 0 {
			return diagnostics.ResultState == "unknown"
		}
		return diagnostics.ResultState == "complete"
	}
	if terminal != "failed" && terminal != "cancelled" {
		return false
	}
	if diagnostics.FailedStage == "" || diagnostics.ErrorType == "" {
		return false
	}
	if diagnostics.AcknowledgedItems > 0 || diagnostics.AcknowledgedBatches > 0 {
		return diagnostics.ResultState == "partial"
	}
	if diagnostics.UnresolvedBatches > 0 {
		return diagnostics.ResultState == "unknown"
	}
	return diagnostics.ResultState == "none"
}

func smokeConfirmedResultDiagnostics(diagnostics smokeDiagnosticEvidence) bool {
	return diagnostics.Valid && diagnostics.Availability == "available" && diagnostics.ResultState == "complete" && diagnostics.ResultTypeCount > 0 &&
		diagnostics.AcknowledgedItems > 0 && diagnostics.AcknowledgedBatches > 0 && diagnostics.UnresolvedBatches == 0
}

func smokeZeroResultDiagnostics(diagnostics smokeDiagnosticEvidence) bool {
	// Result ports register lazily on the first submitted item, so a valid
	// zero-hit execution may have no result-type watermark at all.
	return diagnostics.Valid && diagnostics.Availability == "available" && diagnostics.ResultState == "complete" && diagnostics.ResultTypeCount >= 0 &&
		diagnostics.ReceivedItems == 0 && diagnostics.EncodedItems == 0 && diagnostics.SubmittedItems == 0 && diagnostics.AcknowledgedItems == 0 &&
		diagnostics.SubmittedBatches == 0 && diagnostics.AcknowledgedBatches == 0 && diagnostics.UnresolvedBatches == 0
}

func smokeNoConfirmedFailureDiagnostics(diagnostics smokeDiagnosticEvidence) bool {
	return diagnostics.Valid && diagnostics.Availability == "available" && diagnostics.ResultState == "none" && diagnostics.UnresolvedBatches == 0 &&
		diagnostics.AcknowledgedItems == 0 && diagnostics.AcknowledgedBatches == 0
}

func smokeUnavailableDiagnostics(diagnostics smokeDiagnosticEvidence) bool {
	return diagnostics.Valid && diagnostics.Availability == "unavailable" && diagnostics.ResultState == "unknown"
}

func sortedScenarioEngineIDs(records []smokePlanRecordSnapshot) []string {
	unique := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.engineID != "" {
			unique[record.engineID] = struct{}{}
		}
	}
	engineIDs := make([]string, 0, len(unique))
	for engineID := range unique {
		engineIDs = append(engineIDs, engineID)
	}
	sort.Strings(engineIDs)
	return engineIDs
}

func publishFile(path string, value []byte) error {
	if path == "" {
		return errors.New("output path is required")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lunafox-engine-cut-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(value); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return nil
}

func runSmoke(ctx context.Context, cfg smokeConfig) error {
	if ctx == nil {
		return errors.New("smoke context is required")
	}
	packages, installed, err := loadPackageReceipt(ctx, cfg.packageBuildResults, cfg.packageCacheRoot)
	if err != nil {
		return err
	}
	runtime, err := buildSmokeRuntime(ctx, cfg, packages, installed)
	if err != nil {
		return err
	}
	defer runtime.cleanup()

	streams := agentcontrol.NewAgentStreamRegistry()
	control := agentcontrol.NewControlPlaneService(runtime.authority, runtime.authority, runtime.bridge, streams).WithActiveSessionRegistry(runtime.registry)
	runtime.bridge.SetCancelPublisher(agentcontrol.NewAgentControlEventPublisher(streams))
	ingest := &smokeResultIngest{bridge: runtime.bridge, inner: newSmokeResultFacade(runtime.resultEvidence)}
	data := agentdata.NewDataPlaneService(
		runtime.authority,
		agentdata.ResultIngestDataPlanes{
			TaskScopes: agentdata.NewResultTaskScopeDataPlane(runtime.scanStore.tasks, runtime.scanStore.scans),
			Ingest:     ingest,
		}).
		WithTaskProgressLogDataPlane(&smokeProgressSink{bridge: runtime.bridge}).WithAgentSessionReader(runtime.registry)
	artifactResolver, err := newProductionArtifactObserver(agentdata.ServerExecutionArtifactResolverDependencies{
		Tasks: runtime.scanStore.tasks, Sessions: runtime.registry,
		DNSNames: smokeRuntimeDNSFacts(runtime), HostPorts: runtime.resultEvidence, WebsiteURLs: runtime.resultEvidence, EndpointURLs: runtime.resultEvidence,
		BlacklistSnapshots: newSmokeExecutionInputBlacklistSnapshotSource(scanwiring.NewScanBlacklistSnapshotStoreAdapter(runtime.scanStore.scans)),
		Wordlists:          runtime.wordlists, ProviderConfig: runtime.providerSource,
		NucleiTemplates: smokeNucleiTemplateSource{bridge: runtime.bridge},
	}, runtime.bridge)
	if err != nil {
		return err
	}
	artifacts := agentdata.NewExecutionArtifactService(runtime.authority, artifactResolver)
	grpcServer, err := agentserver.New(
		cfg.listen,
		control,
		data,
		artifacts,
		grpc.MaxRecvMsgSize(agentdata.MaxDataPlaneRecvMessageBytes),
		// The smoke Agent uses the production transport keepalive policy. The
		// standalone harness must accept that cadence as well, otherwise its
		// default gRPC enforcement closes an otherwise healthy control stream.
		grpc.KeepaliveParams(keepalive.ServerParameters{Time: 10 * time.Second, Timeout: 20 * time.Second}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: 5 * time.Second, PermitWithoutStream: true}),
	)
	if err != nil {
		return err
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcServer.Serve() }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = grpcServer.Shutdown(shutdownCtx)
	}()

	if err := publishFile(cfg.addrFile, []byte(grpcServer.Addr()+"\n")); err != nil {
		return fmt.Errorf("publish Server readiness: %w", err)
	}

	scenarioCtx, cancelScenario := context.WithTimeout(ctx, 8*time.Minute)
	defer cancelScenario()
	terminalDone := make(chan bool, 1)
	go func() { terminalDone <- runtime.bridge.waitTerminal(scenarioCtx, len(scenarioOrder(runtime.cfg))) }()
	select {
	case err := <-serveErr:
		cancelScenario()
		if err != nil {
			return fmt.Errorf("serve Agent plane: %w", err)
		}
		return errors.New("Agent plane stopped before smoke completion")
	case done := <-terminalDone:
		if !done {
			return errors.New("not all smoke terminal acknowledgements arrived")
		}
	case <-scenarioCtx.Done():
		return fmt.Errorf("implementation-cut smoke timed out waiting for terminal acknowledgements: %w", scenarioCtx.Err())
	}
	ackDeadline := time.Now().Add(10 * time.Second)
	for !runtime.bridge.allTerminalsAcked() && time.Now().Before(ackDeadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if !runtime.bridge.allTerminalsAcked() {
		return errors.New("smoke terminal acknowledgements did not converge")
	}
	cleanupDeadline := time.Now().Add(15 * time.Second)
	for !runtime.bridge.cleanupObserved() && time.Now().Before(cleanupDeadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if !runtime.bridge.cleanupObserved() {
		return errors.New("smoke did not observe post-terminal zero heartbeat cleanup")
	}
	// Let the Agent make one post-terminal pull so the real TaskAssign no_task
	// oneof is observed without extending the task execution budget.
	noTaskDeadline := time.Now().Add(5 * time.Second)
	for runtime.bridge.countersSnapshot().noTaskAssignments == 0 && time.Now().Before(noTaskDeadline) {
		time.Sleep(25 * time.Millisecond)
	}

	evidenceBytes, err := json.MarshalIndent(newSmokeEvidence(packages.packageReceiptIDs(), packages.receiptMode, runtime), "", "  ")
	if err != nil {
		return fmt.Errorf("encode smoke evidence: %w", err)
	}
	if err := publishFile(cfg.evidenceFile, append(evidenceBytes, '\n')); err != nil {
		return fmt.Errorf("publish smoke evidence: %w", err)
	}
	if err := publishFile(cfg.doneFile, []byte("done\n")); err != nil {
		return fmt.Errorf("publish smoke completion: %w", err)
	}

	return nil
}

func main() {
	cfg, err := parseSmokeConfig(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		log.Fatalf("load smoke Server flags: %v", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runSmoke(ctx, cfg); err != nil {
		log.Fatalf("engine-container-cut-smoke-server: %v", err)
	}
}
