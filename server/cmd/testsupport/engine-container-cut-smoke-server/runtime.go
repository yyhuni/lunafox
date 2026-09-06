package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	agentcontrol "github.com/yyhuni/lunafox/server/internal/grpc/agentcontrol"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
)

type smokeScenario string

const (
	smokeScenarioPortSuccess       smokeScenario = "success"
	smokeScenarioWebsiteSuccess    smokeScenario = "website_success"
	smokeScenarioPortEmpty         smokeScenario = "port_empty"
	smokeScenarioWebsiteEmpty      smokeScenario = "empty_input"
	smokeScenarioArtifactIntegrity smokeScenario = "artifact_integrity"
	smokeScenarioSubdomainFail     smokeScenario = "failure"
	smokeScenarioWordlistCacheHit  smokeScenario = "wordlist_cache_hit"
	smokeScenarioPortCancel        smokeScenario = "cancel"
	smokeScenarioPortTimeout       smokeScenario = "timeout"
	smokeScenarioNucleiHit         smokeScenario = "nuclei_hit"
	smokeScenarioNucleiZero        smokeScenario = "nuclei_zero_output"
	smokeScenarioNucleiNoTemplates smokeScenario = "nuclei_no_enabled_templates"
	smokeScenarioNucleiCancel      smokeScenario = "nuclei_cancel"
	smokeScenarioNucleiTimeout     smokeScenario = "nuclei_timeout"
	smokeScenarioNucleiAckFailure  smokeScenario = "nuclei_ack_then_failed"
	smokeScenarioNucleiMalformed   smokeScenario = "nuclei_malformed_jsonl"
	smokeScenarioNucleiMixed       smokeScenario = "nuclei_mixed_output"
	smokeScenarioNucleiNonzero     smokeScenario = "nuclei_nonzero_exit"
)

const (
	engineSubdomainID = "engine.lunafox.subdomain_discovery"
	enginePortID      = "engine.lunafox.port_scan"
	engineWebsiteID   = "engine.lunafox.website_discovery"
	engineNucleiID    = "engine.lunafox.nuclei_vulnerability"
	// Keep the empty-facts chain on a target that is not answered by the
	// transparent network used by the container smoke. The Engine still must
	// materialize and consume its Target baseline before HTTPX starts.
	smokeEmptyCIDR = "0.0.0.0/32"
)

func scenarioOrder(config ...smokeConfig) []smokeScenario {
	order := []smokeScenario{
		smokeScenarioPortSuccess,
		smokeScenarioWebsiteSuccess,
		smokeScenarioPortEmpty,
		smokeScenarioWebsiteEmpty,
		smokeScenarioArtifactIntegrity,
		smokeScenarioSubdomainFail,
		smokeScenarioWordlistCacheHit,
		smokeScenarioPortCancel,
		smokeScenarioPortTimeout,
		smokeScenarioNucleiHit,
		smokeScenarioNucleiZero,
		smokeScenarioNucleiNoTemplates,
		smokeScenarioNucleiCancel,
		smokeScenarioNucleiTimeout,
		smokeScenarioNucleiAckFailure,
	}
	if len(config) > 0 && config[0].nucleiFixtureImageRef != "" {
		order = append(order,
			smokeScenarioNucleiMalformed,
			smokeScenarioNucleiMixed,
			smokeScenarioNucleiNonzero,
		)
	}
	return order
}

func nucleiHandlerCount(cfg smokeConfig) int {
	if cfg.nucleiFixtureImageRef != "" {
		return 8
	}
	return 5
}

func nucleiTemplateStreamCount(cfg smokeConfig) int {
	return nucleiHandlerCount(cfg)
}

type smokeScenarioPlan struct {
	manifest             scanapp.ScanCreateWorkflowManifest
	targetType           string
	targetValue          string
	maxExecutionDuration time.Duration
	engineID             string
}

func scenarioPlan(cfg smokeConfig, scenario smokeScenario) (smokeScenarioPlan, error) {
	engineID := enginePortID
	stageID := "ports"
	stepID := "port_scan"
	targetType := scanapp.ExecutionTargetTypeIP
	targetValue := cfg.fixtureIP
	var taskConfig map[string]any
	maxExecutionDuration := 5 * time.Minute
	switch scenario {
	case smokeScenarioPortSuccess:
		taskConfig = portTaskConfig(strconv.Itoa(cfg.fixturePort), 100)
	case smokeScenarioPortCancel:
		taskConfig = portTaskConfig("1-65535", 10_000)
	case smokeScenarioPortTimeout:
		taskConfig = portTaskConfig("1-65535", 1)
		maxExecutionDuration = 2 * time.Second
	case smokeScenarioNucleiHit, smokeScenarioNucleiZero, smokeScenarioNucleiNoTemplates:
		engineID = engineNucleiID
		stageID = "nuclei"
		stepID = "nuclei_vulnerability"
		taskConfig = nucleiTaskConfig(25, 150, 1, 25, 1)
	case smokeScenarioNucleiCancel:
		engineID = engineNucleiID
		stageID = "nuclei"
		stepID = "nuclei_vulnerability"
		targetType = scanapp.ExecutionTargetTypeCIDR
		targetValue = "0.0.0.0/16"
		taskConfig = nucleiTaskConfig(1, 1, 1, 1, 0)
	case smokeScenarioNucleiTimeout:
		engineID = engineNucleiID
		stageID = "nuclei"
		stepID = "nuclei_vulnerability"
		targetType = scanapp.ExecutionTargetTypeCIDR
		targetValue = "0.0.0.0/16"
		taskConfig = nucleiTaskConfig(1, 1, 1, 1, 0)
		maxExecutionDuration = 2 * time.Second
	case smokeScenarioNucleiAckFailure:
		engineID = engineNucleiID
		stageID = "nuclei"
		stepID = "nuclei_vulnerability"
		// Keep the Target baseline first, but reach the finalized fixture URL
		// quickly; later fixture paths deliberately delay to prove an already
		// acknowledged finding survives task timeout.
		taskConfig = nucleiTaskConfig(1, 100, 1, 1, 0)
		// The root URL emits one in-scope finding and each later fixture path
		// waits for two seconds. The valid persisted plan below narrows its
		// result batch to one item, allowing that finding to be acknowledged
		// before this budget expires during the delayed requests.
		maxExecutionDuration = 10 * time.Second
	case smokeScenarioNucleiMalformed, smokeScenarioNucleiMixed, smokeScenarioNucleiNonzero:
		engineID = engineNucleiID
		stageID = "nuclei"
		stepID = "nuclei_vulnerability"
		targetType = scanapp.ExecutionTargetTypeIP
		targetValue = cfg.fixtureIP
		taskConfig = nucleiTaskConfig(1, 25, 1, 1, 0)
	case smokeScenarioWebsiteEmpty:
		engineID = engineWebsiteID
		stageID = "websites"
		stepID = "website_discovery"
		targetType = scanapp.ExecutionTargetTypeCIDR
		targetValue = cfg.fixtureIP + "/32"
		taskConfig = websiteTaskConfig()
	case smokeScenarioArtifactIntegrity:
		engineID = engineSubdomainID
		stageID = "discovery"
		stepID = "subdomain_discovery"
		targetType = scanapp.ExecutionTargetTypeDomain
		targetValue = "example.com"
		taskConfig = subdomainResourceTaskConfig()
	case smokeScenarioSubdomainFail:
		engineID = engineSubdomainID
		stageID = "discovery"
		stepID = "subdomain_discovery"
		targetType = scanapp.ExecutionTargetTypeDomain
		targetValue = "example.com"
		// The persisted plan is deliberately narrowed after it has passed the
		// Server-side required-section gate so the Engine's fail-closed Resolve
		// check remains covered without treating an invalid user config as valid.
		taskConfig = subdomainFailureTaskConfig()
	case smokeScenarioWordlistCacheHit:
		engineID = engineSubdomainID
		stageID = "discovery"
		stepID = "subdomain_discovery"
		targetType = scanapp.ExecutionTargetTypeDomain
		targetValue = "example.com"
		taskConfig = subdomainResourceTaskConfig()
	default:
		return smokeScenarioPlan{}, fmt.Errorf("unsupported smoke scenario %q", scenario)
	}
	return smokeScenarioPlan{
		manifest: smokeOneStepManifest(
			"smoke_runtime_"+string(scenario), stageID, stepID, engineID, taskConfig,
		),
		targetType: targetType, targetValue: targetValue,
		maxExecutionDuration: maxExecutionDuration, engineID: engineID,
	}, nil
}

func createSmokeRecord(
	ctx context.Context,
	store *smokeScanStore,
	compiler *scanapp.PlanTaskCompiler,
	packages *packageMapReader,
	wordlists *smokeWordlists,
	cfg smokeConfig,
	scenario smokeScenario,
) (*smokePlanRecord, error) {
	if ctx == nil || store == nil || compiler == nil || packages == nil || wordlists == nil {
		return nil, errors.New("smoke scan-create dependencies are required")
	}
	spec, err := scenarioPlan(cfg, scenario)
	if err != nil {
		return nil, err
	}
	created, err := store.createScan(
		compiler, packages, spec.manifest, spec.targetType, spec.targetValue, spec.maxExecutionDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("scan-create %s scenario: %w", scenario, err)
	}
	task, ok := created.Tasks[spec.engineID]
	if !ok {
		return nil, fmt.Errorf("scan-create %s scenario did not persist engine %q", scenario, spec.engineID)
	}
	if scenario == smokeScenarioSubdomainFail || scenario == smokeScenarioWordlistCacheHit {
		if err := store.forceSubdomainResolveDisabledPlan(&task); err != nil {
			return nil, fmt.Errorf("prepare %s Engine defense-in-depth scenario: %w", scenario, err)
		}
		created.Tasks[spec.engineID] = task
	}
	if scenario == smokeScenarioNucleiAckFailure {
		if err := store.forceNucleiAcknowledgedBatchPlan(&task); err != nil {
			return nil, fmt.Errorf("prepare %s acknowledged-result scenario: %w", scenario, err)
		}
		created.Tasks[spec.engineID] = task
	}
	return smokeRecordFromPersistedTask(ctx, store, wordlists, created, task, scenario)
}

func smokeRecordFromPersistedTask(
	ctx context.Context,
	store *smokeScanStore,
	wordlists *smokeWordlists,
	created *smokeCreatedScan,
	task smokePersistedTask,
	scenario smokeScenario,
) (*smokePlanRecord, error) {
	plan, encoded, err := store.savedPlan(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("read persisted %s plan: %w", scenario, err)
	}
	wordlistContents := make(map[string][]byte, len(wordlists.contents))
	for name, content := range wordlists.contents {
		wordlistContents[name] = append([]byte(nil), content...)
	}
	return &smokePlanRecord{
		scenario: scenario, scanID: created.ID, taskID: task.ID, targetID: created.TargetID,
		engineID: task.EngineID, task: plan.GetTask(), planBytes: append([]byte(nil), encoded...),
		wordlists: wordlistContents,
	}, nil
}

func createLinkedSmokeRecords(
	ctx context.Context,
	store *smokeScanStore,
	compiler *scanapp.PlanTaskCompiler,
	packages *packageMapReader,
	wordlists *smokeWordlists,
	manifest scanapp.ScanCreateWorkflowManifest,
	targetType string,
	targetValue string,
	maxExecutionDuration time.Duration,
	portScenario smokeScenario,
	websiteScenario smokeScenario,
) ([]*smokePlanRecord, error) {
	created, err := store.createScan(compiler, packages, manifest, targetType, targetValue, maxExecutionDuration)
	if err != nil {
		return nil, err
	}
	portTask, portOK := created.Tasks[enginePortID]
	websiteTask, websiteOK := created.Tasks[engineWebsiteID]
	if !portOK || !websiteOK {
		return nil, fmt.Errorf("linked smoke scan did not persist port and website tasks")
	}
	portRecord, err := smokeRecordFromPersistedTask(ctx, store, wordlists, created, portTask, portScenario)
	if err != nil {
		return nil, err
	}
	websiteRecord, err := smokeRecordFromPersistedTask(ctx, store, wordlists, created, websiteTask, websiteScenario)
	if err != nil {
		return nil, err
	}
	return []*smokePlanRecord{portRecord, websiteRecord}, nil
}

func createSmokeRuntimeRecords(
	ctx context.Context,
	store *smokeScanStore,
	compiler *scanapp.PlanTaskCompiler,
	packages *packageMapReader,
	wordlists *smokeWordlists,
	cfg smokeConfig,
) ([]*smokePlanRecord, error) {
	records, err := createLinkedSmokeRecords(
		ctx, store, compiler, packages, wordlists,
		smokePortWebsiteManifest("smoke_runtime_success_chain", portTaskConfig(strconv.Itoa(cfg.fixturePort), 100)),
		scanapp.ExecutionTargetTypeIP, cfg.fixtureIP, 2*time.Minute,
		smokeScenarioPortSuccess, smokeScenarioWebsiteSuccess,
	)
	if err != nil {
		return nil, fmt.Errorf("create success chain: %w", err)
	}
	emptyRecords, err := createLinkedSmokeRecords(
		ctx, store, compiler, packages, wordlists,
		smokePortWebsiteManifest("smoke_runtime_empty_chain", portZeroResultTaskConfig()),
		scanapp.ExecutionTargetTypeCIDR, smokeEmptyCIDR, 2*time.Minute,
		smokeScenarioPortEmpty, smokeScenarioWebsiteEmpty,
	)
	if err != nil {
		return nil, fmt.Errorf("create empty CIDR chain: %w", err)
	}
	records = append(records, emptyRecords...)
	for _, scenario := range []smokeScenario{
		smokeScenarioArtifactIntegrity,
		smokeScenarioSubdomainFail,
		smokeScenarioWordlistCacheHit,
		smokeScenarioPortCancel,
		smokeScenarioPortTimeout,
		smokeScenarioNucleiHit,
		smokeScenarioNucleiZero,
		smokeScenarioNucleiNoTemplates,
		smokeScenarioNucleiCancel,
		smokeScenarioNucleiTimeout,
		smokeScenarioNucleiAckFailure,
	} {
		record, createErr := createSmokeRecord(ctx, store, compiler, packages, wordlists, cfg, scenario)
		if createErr != nil {
			return nil, createErr
		}
		records = append(records, record)
	}
	if cfg.nucleiFixtureImageRef != "" {
		for _, scenario := range []smokeScenario{
			smokeScenarioNucleiMalformed,
			smokeScenarioNucleiMixed,
			smokeScenarioNucleiNonzero,
		} {
			record, createErr := createSmokeRecord(ctx, store, compiler, packages, wordlists, cfg, scenario)
			if createErr != nil {
				return nil, createErr
			}
			records = append(records, record)
		}
	}
	return records, nil
}

func portTaskConfig(ports string, rate int) map[string]any {
	return portTaskConfigWithTimeout(ports, rate, 60)
}

func portTaskConfigWithTimeout(ports string, rate, timeout int) map[string]any {
	return map[string]any{
		"naabu_active": map[string]any{
			"enabled": true, "port-mode": "custom", "ports": ports,
			"threads": 1, "rate": rate, "timeout": timeout,
		},
		"naabu_passive": map[string]any{"enabled": false},
	}
}

func portZeroResultTaskConfig() map[string]any {
	// Keep one section enabled: the canonical Engine contract rejects an
	// enabled Step whose config sections are all disabled. Passive mode still
	// consumes the empty CIDR Subdomains snapshot and produces zero results.
	return map[string]any{
		"naabu_active":  map[string]any{"enabled": false},
		"naabu_passive": map[string]any{"enabled": true},
	}
}

func nucleiTaskConfig(concurrency, rateLimit, requestTimeout, bulkSize, retries int) map[string]any {
	return map[string]any{"nuclei": map[string]any{
		"enabled": true, "timeout": 60, "concurrency": concurrency,
		"rate-limit": rateLimit, "request-timeout": requestTimeout,
		"bulk-size": bulkSize, "retries": retries,
		"severity": []any{"medium", "high", "critical"},
		"tags":     []any{}, "exclude-tags": []any{},
	}}
}

// smokeRuntimeDNSFacts models one finalized subdomain fact for the linked
// success scan. All other runtime scans intentionally expose a legal empty
// product; Target baselines are generated inside the consuming Engine.
func smokeRuntimeDNSFacts(runtime *smokeRuntime) smokeDNSCursor {
	cursor := smokeDNSCursor{byScan: make(map[int][]string)}
	if runtime == nil || runtime.bridge == nil {
		return cursor
	}
	for _, record := range runtime.bridge.recordSnapshots() {
		if record.scenario == smokeScenarioPortSuccess && record.scanID > 0 {
			cursor.byScan[record.scanID] = []string{"api.example.com"}
		}
	}
	return cursor
}

func buildSmokeRuntime(ctx context.Context, cfg smokeConfig, packages *packageMapReader, installed int) (*smokeRuntime, error) {
	wordlists, err := newSmokeWordlists()
	if err != nil {
		return nil, err
	}
	if cfg.nucleiFixtureImageRef != "" {
		if cfg.nucleiFixtureSourceRef == "" {
			wordlists.cleanup()
			return nil, errors.New("Nuclei fixture source ref is required")
		}
		if err := packages.OverrideRuntimeImageForSmoke(engineNucleiID, cfg.nucleiFixtureSourceRef, cfg.nucleiFixtureImageRef); err != nil {
			wordlists.cleanup()
			return nil, fmt.Errorf("apply Nuclei smoke fixture image: %w", err)
		}
	}
	compiler, err := scanapp.NewPlanTaskCompiler(packages, wordlists)
	if err != nil {
		wordlists.cleanup()
		return nil, err
	}
	store, err := newSmokeScanStore()
	if err != nil {
		wordlists.cleanup()
		return nil, err
	}
	cleanupOnError := func() {
		store.cleanup()
		wordlists.cleanup()
	}
	workflowEvidence, err := runSmokeWorkflowEvidence(ctx, store, compiler, packages)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("collect workflow evidence: %w", err)
	}
	inputEvidence, err := runSmokeInputSemanticsEvidence(ctx, store, compiler, packages, cfg.fixtureIP)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("collect input semantics evidence: %w", err)
	}
	sessionFenceEvidence, err := runSmokeSessionFenceEvidence(ctx, compiler, packages)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("collect session-fence evidence: %w", err)
	}
	records, err := createSmokeRuntimeRecords(ctx, store, compiler, packages, wordlists, cfg)
	if err != nil {
		cleanupOnError()
		return nil, err
	}
	scanCreateEvidence, err := store.scanCreateEvidence(ctx)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("collect scan-create evidence: %w", err)
	}
	providerStore := &smokeProviderSettingsStore{}
	providerSource := catalogExecutionProviderSource(providerStore)
	providerContent, err := providerSource.GetExecutionSubfinderProviderConfig(ctx)
	if err != nil {
		cleanupOnError()
		return nil, fmt.Errorf("produce smoke provider config: %w", err)
	}
	registry := agentcontrol.NewActiveSessionRegistry()
	authority := newSmokeAuthority(cfg.authenticationToken, registry)
	authority.UseSessionStore(store)
	bridge := newSmokeTaskBridge(authority, records)
	bridge.UseProductionBridge(
		scanapp.NewScanTaskBridgeService(store.tasks, store.scans).
			WithEngineExecutionClaimStore(store.tasks),
	)
	if installed != len(packages.packages) || installed != len(packages.packageReceiptIDs()) {
		cleanupOnError()
		return nil, fmt.Errorf("package installation count does not match the receipt identity set")
	}
	resultEvidence := newSmokeResultEvidenceStore(cfg)
	fixtureURL := fmt.Sprintf("http://%s:%d", cfg.fixtureIP, cfg.fixturePort)
	for _, record := range records {
		if record == nil || record.engineID != engineNucleiID || record.scenario == smokeScenarioNucleiNoTemplates ||
			record.scenario == smokeScenarioNucleiCancel || record.scenario == smokeScenarioNucleiTimeout {
			continue
		}
		if err := resultEvidence.seedWebsiteURL(record.scanID, fixtureURL); err != nil {
			cleanupOnError()
			return nil, fmt.Errorf("seed Nuclei Website fixture for %s: %w", record.scenario, err)
		}
	}
	return &smokeRuntime{
		cfg: cfg, packages: packages, compiler: compiler, wordlists: wordlists,
		scanStore: store, scanCreateEvidence: scanCreateEvidence,
		inputEvidence: inputEvidence, workflowEvidence: workflowEvidence,
		sessionFenceEvidence: sessionFenceEvidence,
		authority:            authority, registry: registry, bridge: bridge,
		providerSource: providerSource, providerContent: []byte(providerContent),
		resultEvidence: resultEvidence,
	}, nil
}

// The import aliases below are kept in this small factory section so the
// provider source remains visibly Server-owned in the smoke composition.
type smokeRuntime struct {
	cfg                  smokeConfig
	packages             *packageMapReader
	compiler             *scanapp.PlanTaskCompiler
	wordlists            *smokeWordlists
	scanStore            *smokeScanStore
	scanCreateEvidence   smokeScanCreateEvidence
	inputEvidence        smokeInputSemanticsEvidence
	workflowEvidence     smokeWorkflowEvidence
	sessionFenceEvidence smokeSessionFenceEvidence
	authority            *smokeAuthority
	registry             *agentcontrol.ActiveSessionRegistry
	bridge               *smokeTaskBridge
	providerSource       providerConfigSource
	providerContent      []byte
	resultEvidence       *smokeResultEvidenceStore
}

func (runtime *smokeRuntime) cleanup() {
	if runtime == nil {
		return
	}
	if runtime.wordlists != nil {
		runtime.wordlists.cleanup()
	}
	if runtime.scanStore != nil {
		runtime.scanStore.cleanup()
	}
}

func (runtime *smokeRuntime) record(scenario smokeScenario) (smokePlanRecordSnapshot, bool) {
	if runtime == nil || runtime.bridge == nil {
		return smokePlanRecordSnapshot{}, false
	}
	runtime.bridge.mu.Lock()
	defer runtime.bridge.mu.Unlock()
	for _, record := range runtime.bridge.queue {
		if record != nil && record.scenario == scenario {
			return smokeRecordSnapshot(record), true
		}
	}
	return smokePlanRecordSnapshot{}, false
}

var (
	_ agentcontrol.AgentControlLifecycle = (*smokeAuthority)(nil)
)
