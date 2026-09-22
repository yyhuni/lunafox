package upgrader

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/yyhuni/lunafox/contracts/releasemanifest"
)

const (
	defaultManifestName = "release.manifest.yaml"
	requestTimeout      = 30 * time.Second
	// macOS exposes a shorter sockaddr_un path than Linux. Keep a conservative
	// bound so a deployment root that works on one supported host also works on
	// the other, instead of failing after the lock has been acquired.
	maxUnixSocketPathBytes = 100
)

var (
	ErrReplayDigestMismatch = errors.New("operation replay has a different manifest digest")
	ErrOperationInProgress  = errors.New("another upgrade operation is already in progress")
	ErrResumeNotFound       = errors.New("upgrade operation cannot be resumed because its journal is missing")
	ErrManifestMismatch     = errors.New("deployment manifest digest does not match the requested operation")
	ErrRepairRequired       = errors.New("terminal upgrade operation requires an explicit repair action")
)

// Executor is intentionally narrower than a shell command runner. A future
// Compose executor receives the already-validated request and the private
// journal store; it cannot receive command strings, paths, or image refs from
// the Server boundary.
type Executor interface {
	Execute(context.Context, Request, *JournalStore) error
}

// ExecutorFunc adapts a function to Executor.
type ExecutorFunc func(context.Context, Request, *JournalStore) error

func (function ExecutorFunc) Execute(ctx context.Context, request Request, store *JournalStore) error {
	return function(ctx, request, store)
}

// ManifestVerifier re-reads the fixed deployment manifest after a request is
// accepted. The digest supplied by the Server is only an identity check; the
// host never trusts client-provided image or path values.
type ManifestVerifier func(string, string) error

func VerifyDeploymentManifest(manifestPath, expectedDigest string) error {
	info, err := os.Lstat(manifestPath)
	if err != nil {
		return fmt.Errorf("load deployment manifest: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("deployment manifest must be a regular file")
	}
	manifest, err := loadManifestWithLegacyCompatibility(manifestPath)
	if err != nil {
		return fmt.Errorf("validate deployment manifest: %w", err)
	}
	if manifest.Digest() != expectedDigest {
		return ErrManifestMismatch
	}
	return nil
}

// Daemon is the independent host process boundary. Its lock is held for the
// full listener lifetime, so two processes cannot race Compose/root actions.
type Daemon struct {
	store          *JournalStore
	executor       Executor
	verifyManifest ManifestVerifier
	scopePlanner   ScopePlanner
	listener       net.Listener
	lock           *fileLock

	mu              sync.Mutex
	closing         bool
	running         bool
	runCtx          context.Context
	runCancel       context.CancelFunc
	executing       map[string]context.CancelFunc
	cancelRequested map[string]struct{}
	// deploymentLock is the host-visible lock shared with the public Bash
	// lifecycle scripts. It is held for the whole duration of an Upgrade
	// Operation and kept as a recovery fence while the journal needs recovery.
	deploymentLock *DeploymentLock
	wg             sync.WaitGroup
}

func NewDaemon(store *JournalStore, executor Executor) *Daemon {
	var observer RuntimeObserver
	var candidate CandidateDeploymentSource
	// Digest-addressed composition evidence exists only in the public release
	// layout. Legacy/development stores still advertise the protocol envelope,
	// but their planner must deliberately produce a full plan from the fixed
	// manifest rather than failing a v2 negotiation on a missing cache.
	candidate = NewManifestCandidateDeploymentSource(store)
	if store != nil && store.manifestCache {
		candidate = NewCompositionCandidateDeploymentSource(store)
	}
	if composeExecutor, ok := executor.(*ComposeExecutor); ok {
		observer = NewDockerRuntimeObserver(composeExecutor)
		if dockerObserver, ok := observer.(*DockerRuntimeObserver); ok && store != nil {
			dockerObserver.SetDeploymentRoot(store.DeploymentRoot())
		}
	}
	return &Daemon{
		store:           store,
		executor:        executor,
		verifyManifest:  VerifyDeploymentManifest,
		scopePlanner:    NewHostScopePlanner(store, candidate, observer),
		executing:       make(map[string]context.CancelFunc),
		cancelRequested: make(map[string]struct{}),
	}
}

// SetScopePlanner replaces host-side scope planning for production wiring and
// hermetic tests. A nil planner deliberately disables v2 planning rather than
// letting the daemon manufacture a scope from a request or release manifest.
func (daemon *Daemon) SetScopePlanner(planner ScopePlanner) {
	if daemon == nil {
		return
	}
	daemon.mu.Lock()
	daemon.scopePlanner = planner
	daemon.mu.Unlock()
}

// SetManifestVerifier exists for hermetic tests and for distributions that
// stage the manifest under a platform-specific fixed filename. Production
// callers should keep VerifyDeploymentManifest.
func (daemon *Daemon) SetManifestVerifier(verifier ManifestVerifier) {
	if verifier == nil {
		verifier = VerifyDeploymentManifest
	}
	daemon.verifyManifest = verifier
}

func (daemon *Daemon) verifyRequestManifest(digest string) error {
	if daemon == nil || daemon.store == nil || daemon.verifyManifest == nil {
		return fmt.Errorf("manifest verifier is not configured")
	}
	manifestPath, err := daemon.store.ManifestPath(digest)
	if err != nil {
		return err
	}
	return daemon.verifyManifest(manifestPath, digest)
}

// SetEventSink forwards sanitized journal/receipt observations to the control
// plane. Delivery is best effort and never participates in the privileged
// execution decision.
func (daemon *Daemon) SetEventSink(sink EventSink) {
	if daemon == nil || daemon.store == nil {
		return
	}
	daemon.store.SetEventSink(sink)
}

func (daemon *Daemon) Store() *JournalStore {
	if daemon == nil {
		return nil
	}
	return daemon.store
}

// Listen binds the private Unix socket and acquires the process lock. It is
// exposed separately so tests can inspect the socket before serving requests.
func (daemon *Daemon) Listen() error {
	if daemon == nil || daemon.store == nil {
		return fmt.Errorf("upgrader daemon is not configured")
	}
	if err := validateUnixSocketPath(daemon.store.SocketPath()); err != nil {
		return err
	}
	daemon.mu.Lock()
	defer daemon.mu.Unlock()
	if daemon.listener != nil {
		return nil
	}
	lock, err := acquireFileLock(daemon.store.LockPath())
	if err != nil {
		return err
	}
	if err := removeStaleSocket(daemon.store.SocketPath()); err != nil {
		_ = lock.Close()
		return err
	}
	listener, err := net.Listen("unix", daemon.store.SocketPath())
	if err != nil {
		_ = lock.Close()
		return fmt.Errorf("listen on upgrader socket: %w", err)
	}
	if err := os.Chmod(daemon.store.SocketPath(), 0o600); err != nil {
		_ = listener.Close()
		_ = os.Remove(daemon.store.SocketPath())
		_ = lock.Close()
		return fmt.Errorf("secure upgrader socket: %w", err)
	}
	daemon.listener = listener
	daemon.lock = lock
	daemon.closing = false
	daemon.running = true
	return nil
}

func validateUnixSocketPath(path string) error {
	if path == "" {
		return fmt.Errorf("upgrader socket path is required")
	}
	if len([]byte(path)) > maxUnixSocketPathBytes {
		return fmt.Errorf("upgrader socket path exceeds %d bytes", maxUnixSocketPathBytes)
	}
	return nil
}

// Serve runs until ctx is cancelled. It resumes an unfinished journal before
// accepting new requests, which is what makes Server/browser restarts safe.
func (daemon *Daemon) Serve(ctx context.Context) (serveErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := daemon.Listen(); err != nil {
		return err
	}
	runCtx, runCancel := context.WithCancel(ctx)
	daemon.mu.Lock()
	daemon.runCtx = runCtx
	daemon.runCancel = runCancel
	daemon.mu.Unlock()
	defer func() {
		serveErr = errors.Join(serveErr, daemon.Close())
	}()
	defer runCancel()
	// A pending v2 confirmation keeps the deployment lock as a fence. Establish
	// that fence before resuming the journal or accepting a socket request; an
	// asynchronous acquisition would leave a window in which confirm could
	// promote a staged override while a lifecycle command mutates the same
	// deployment.
	if err := daemon.ensureRecoveryFence(runCtx); err != nil && !errors.Is(err, ErrJournalNotFound) {
		return err
	}
	if err := daemon.resumeCurrent(runCtx); err != nil && !errors.Is(err, ErrJournalNotFound) {
		return err
	}
	// Capture the listener while holding the daemon mutex. Close clears the
	// field during shutdown; the accept loop must keep using this stable local
	// handle until the listener is closed, otherwise shutdown races with Accept.
	daemon.mu.Lock()
	listener := daemon.listener
	daemon.mu.Unlock()
	if listener == nil {
		return fmt.Errorf("upgrader listener is not configured")
	}

	acceptErr := make(chan error, 1)
	// Keep the accept loop in the wait group so Close cannot return while it is
	// still able to add a connection worker. The counter remains non-zero until
	// the loop exits after listener.Close unblocks Accept.
	daemon.wg.Add(1)
	go func() {
		defer daemon.wg.Done()
		for {
			connection, err := listener.Accept()
			if err != nil {
				select {
				case acceptErr <- err:
				default:
				}
				return
			}
			daemon.wg.Add(1)
			go func() {
				defer daemon.wg.Done()
				daemon.serveConnection(connection)
			}()
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-acceptErr:
		if daemon.isClosing() {
			return nil
		}
		return err
	}
}

func (daemon *Daemon) Close() error {
	if daemon == nil {
		return nil
	}
	daemon.mu.Lock()
	if daemon.closing {
		daemon.mu.Unlock()
		return nil
	}
	daemon.closing = true
	listener := daemon.listener
	lock := daemon.lock
	runCancel := daemon.runCancel
	daemon.listener = nil
	daemon.lock = nil
	daemon.runCtx = nil
	daemon.runCancel = nil
	daemon.running = false
	daemon.mu.Unlock()

	var firstErr error
	if runCancel != nil {
		runCancel()
	}
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			firstErr = err
		}
	}
	daemon.wg.Wait()
	if err := os.Remove(daemon.store.SocketPath()); err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
		firstErr = err
	}
	// The deployment lock is intentionally not released here when the journal
	// needs recovery: it is the fence that stops lifecycle scripts from mutating
	// an unresolved deployment.
	daemon.mu.Lock()
	deploymentLock := daemon.deploymentLock
	daemon.mu.Unlock()
	if deploymentLock != nil {
		if journal, loadErr := daemon.store.LoadCurrent(); loadErr == nil && !requiresDeploymentFence(journal) {
			if err := deploymentLock.Release(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	if err := lock.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (daemon *Daemon) isClosing() bool {
	daemon.mu.Lock()
	defer daemon.mu.Unlock()
	return daemon.closing
}

func (daemon *Daemon) resumeCurrent(ctx context.Context) error {
	journal, err := daemon.store.LoadCurrent()
	if err != nil {
		return err
	}
	if IsTerminal(journal.Stage) || daemon.executor == nil {
		return nil
	}
	// A frontend-only receipt proves the host handoff already completed. Its
	// baseline intentionally still names the previous frontend digest until
	// confirm promotes the staged override, so replaying/revalidating here would
	// turn a completed handoff into a false stale-plan failure after a restart.
	pending, err := daemon.store.frontendOnlyConfirmationPending(journal)
	if err != nil {
		return err
	}
	if pending {
		return nil
	}
	request := Request{
		SchemaVersion:  RequestSchema,
		OperationID:    journal.OperationID,
		Action:         ActionResume,
		ManifestDigest: journal.ManifestDigest,
	}
	if journal.SchemaVersion == ScopedJournalSchema {
		request.SchemaVersion = ScopedRequestSchema
		request.ExecutionMode = journal.ExecutionMode
		request.PlanDigest = journal.PlanDigest
		request.BaselineStateDigest = journal.BaselineStateDigest
		request.TouchedServices = append([]string(nil), journal.TouchedServices...)
		request.ConfirmedDeploymentVersion = journal.ConfirmedDeploymentVersion
		if _, err := daemon.validatePlanBoundRequest(request); err != nil {
			return daemon.failJournal(request, err)
		}
	}
	if daemon.verifyManifest != nil {
		if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
			return daemon.failJournal(request, err)
		}
	}
	daemon.mu.Lock()
	daemon.launchExecutionLocked(ctx, request)
	daemon.mu.Unlock()
	return nil
}

func (daemon *Daemon) serveConnection(connection net.Conn) {
	defer func() {
		_ = connection.Close()
	}()
	_ = connection.SetReadDeadline(time.Now().Add(requestTimeout))
	reader := bufio.NewReader(io.LimitReader(connection, MaxRequestBytes+1))
	data, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		daemon.writeResponse(connection, Response{SchemaVersion: RequestSchema, Error: "invalid upgrader request"})
		return
	}
	if len(data) > MaxRequestBytes {
		daemon.writeResponse(connection, Response{SchemaVersion: RequestSchema, Error: "upgrader request is too large"})
		return
	}
	var request Request
	if err := decodeStrict(data, &request); err != nil {
		daemon.writeResponse(connection, Response{SchemaVersion: RequestSchema, Error: "invalid upgrader request"})
		return
	}
	response := daemon.accept(request)
	daemon.writeResponse(connection, response)
}

func (daemon *Daemon) accept(request Request) Response {
	if err := request.Validate(); err != nil {
		return rejectedResponseFor(responseSchemaFor(request), err)
	}
	switch request.Action {
	case ActionCapabilities:
		return daemon.acceptCapabilities(request)
	case ActionPlan:
		return daemon.acceptScopePlan(request)
	case ActionConfirm:
		return daemon.acceptConfirmation(request)
	case ActionDeploymentState:
		return daemon.acceptDeploymentState(request)
	case ActionCandidateAvailability:
		return daemon.acceptCandidateAvailability(request)
	}
	daemon.mu.Lock()
	if request.SchemaVersion == ScopedRequestSchema {
		if _, err := daemon.validatePlanBoundRequest(request); err != nil {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, err)
		}
	}

	current, currentErr := daemon.store.LoadCurrent()
	if currentErr != nil && !errors.Is(currentErr, ErrJournalNotFound) {
		// A damaged current checkpoint is not equivalent to an empty deployment.
		// Refuse new work until the operator repairs the journal, otherwise a
		// fresh request could overwrite the only durable recovery evidence.
		daemon.mu.Unlock()
		return rejectedResponseFor(request.SchemaVersion, currentErr)
	}
	history, historyErr := daemon.store.LoadOperation(request.OperationID)
	if historyErr != nil && !errors.Is(historyErr, ErrJournalNotFound) {
		daemon.mu.Unlock()
		return rejectedResponseFor(request.SchemaVersion, historyErr)
	}
	if request.Action == ActionStop {
		response := daemon.acceptStopLocked(request, current, currentErr, history, historyErr)
		daemon.mu.Unlock()
		return response
	}
	if historyErr == nil {
		if history.ManifestDigest != request.ManifestDigest {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, ErrReplayDigestMismatch)
		}
		if err := validateJournalRequestScope(history, request); err != nil {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, err)
		}
		// Save writes current before the per-operation copy. If a process was
		// interrupted between those writes, current is the newer complete
		// checkpoint and must win over a stale history snapshot.
		if currentErr == nil && current.OperationID == request.OperationID && current.UpdatedAt.After(history.UpdatedAt) {
			history = current
		}
		// A history entry may outlive the current deployment checkpoint. A
		// non-terminal old operation must never be resumed after a newer
		// operation has reached any stage (including a terminal one), because
		// an executor failure could otherwise overwrite current.json with the
		// stale operation's identity.
		if currentErr == nil && current.OperationID != request.OperationID && !IsTerminal(history.Stage) {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, ErrOperationInProgress)
		}
		pendingConfirmation, pendingErr := daemon.store.frontendOnlyConfirmationPending(history)
		if pendingErr != nil {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, pendingErr)
		}
		if IsTerminal(history.Stage) {
			if request.Action == ActionRepair {
				if _, exists := daemon.executing[request.OperationID]; exists {
					daemon.mu.Unlock()
					return rejectedResponseFor(request.SchemaVersion, ErrRepairNotAllowed)
				}
				if daemon.verifyManifest != nil {
					if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
						daemon.mu.Unlock()
						return rejectedResponseFor(request.SchemaVersion, ErrManifestMismatch)
					}
				}
				repaired, err := daemon.store.ResetForRepair(request.OperationID, request.ManifestDigest)
				if err != nil {
					daemon.mu.Unlock()
					return rejectedResponseFor(request.SchemaVersion, err)
				}
				daemon.launchExecutionLocked(context.Background(), request)
				daemon.mu.Unlock()
				return executionResponse(request, repaired, false, true)
			}
			if request.Action == ActionResume {
				daemon.mu.Unlock()
				return rejectedResponseFor(request.SchemaVersion, ErrRepairRequired)
			}
			daemon.mu.Unlock()
			return executionResponse(request, history, true, false)
		}
		if pendingConfirmation {
			// Host verification and receipt persistence already completed. Keep a
			// duplicate start/resume idempotent while the Server performs confirm;
			// launching here would revalidate against the pre-promotion baseline.
			daemon.mu.Unlock()
			return executionResponse(request, history, true, false)
		}
		if request.Action == ActionRepair {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, ErrRepairNotAllowed)
		}
		if request.Action != ActionResume && request.Action != ActionStart {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, ErrResumeNotFound)
		}
		if _, exists := daemon.executing[request.OperationID]; exists {
			daemon.mu.Unlock()
			return executionResponse(request, history, true, false)
		}
		if daemon.verifyManifest != nil {
			if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
				daemon.mu.Unlock()
				return rejectedResponseFor(request.SchemaVersion, ErrManifestMismatch)
			}
		}
		daemon.launchExecutionLocked(context.Background(), request)
		daemon.mu.Unlock()
		return executionResponse(request, history, true, false)
	}
	if currentErr == nil && !IsTerminal(current.Stage) {
		daemon.mu.Unlock()
		return rejectedResponseFor(request.SchemaVersion, ErrOperationInProgress)
	}
	if request.Action == ActionResume {
		daemon.mu.Unlock()
		return rejectedResponseFor(request.SchemaVersion, ErrResumeNotFound)
	}
	if request.Action == ActionRepair {
		daemon.mu.Unlock()
		return rejectedResponseFor(request.SchemaVersion, ErrRepairNotAllowed)
	}
	if daemon.verifyManifest != nil {
		if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
			daemon.mu.Unlock()
			return rejectedResponseFor(request.SchemaVersion, ErrManifestMismatch)
		}
	}
	journal := journalForRequest(request, time.Now().UTC())
	if err := daemon.store.Save(journal); err != nil {
		daemon.mu.Unlock()
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	daemon.launchExecutionLocked(context.Background(), request)
	daemon.mu.Unlock()
	return executionResponse(request, journal, false, false)
}

func (daemon *Daemon) acceptCapabilities(request Request) Response {
	daemon.mu.Lock()
	planner := daemon.scopePlanner
	daemon.mu.Unlock()
	capabilities := HostCapabilities{SchemaVersions: []int{RequestSchema}}
	if planner != nil {
		capabilities = planner.Capabilities()
	}
	// ScopePlanner implementations own the slice returned from Capabilities.
	// Clone before filtering v3 so negotiation cannot mutate custom planner state.
	capabilities = cloneHostCapabilities(capabilities)
	if _, supported := planner.(CandidateAvailabilityPlanner); !supported {
		// A custom planner may implement only the v2 planning contract. Never
		// advertise a v3 action that this daemon cannot actually dispatch.
		capabilities.CandidateInventoryAvailability = false
		filteredSchemas := capabilities.SchemaVersions[:0]
		for _, schema := range capabilities.SchemaVersions {
			if schema != RequestSchemaV3 {
				filteredSchemas = append(filteredSchemas, schema)
			}
		}
		capabilities.SchemaVersions = filteredSchemas
	}
	capabilities = projectHostCapabilitiesForSchema(capabilities, request.SchemaVersion)
	if err := capabilities.Validate(); err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	return Response{SchemaVersion: request.SchemaVersion, Accepted: true, Capabilities: &capabilities}
}

// acceptDeploymentState is a read-only v2 projection. It intentionally has no
// operation binding: availability checks need the last confirmed inventory even
// when the running Server binary predates the deployment that established it.
func (daemon *Daemon) acceptDeploymentState(request Request) Response {
	state, err := daemon.store.LoadConfirmedDeploymentState()
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	return Response{
		SchemaVersion:            request.SchemaVersion,
		Accepted:                 true,
		ConfirmedDeploymentState: &state,
	}
}

// acceptCandidateAvailability is a schema-v3 read-only comparison. It must not
// allocate an operation, persist a scope plan, or acquire the deployment lock:
// the result only controls whether the Server presents a candidate at all.
func (daemon *Daemon) acceptCandidateAvailability(request Request) Response {
	daemon.mu.Lock()
	planner := daemon.scopePlanner
	daemon.mu.Unlock()
	availabilityPlanner, ok := planner.(CandidateAvailabilityPlanner)
	if !ok || availabilityPlanner == nil {
		return rejectedResponseFor(request.SchemaVersion, fmt.Errorf("host candidate availability is not supported"))
	}
	availability, err := availabilityPlanner.CandidateAvailability(context.Background(), request.ManifestDigest)
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	return Response{
		SchemaVersion:         request.SchemaVersion,
		Accepted:              true,
		CandidateAvailability: &availability,
	}
}

func (daemon *Daemon) acceptScopePlan(request Request) Response {
	daemon.mu.Lock()
	planner := daemon.scopePlanner
	daemon.mu.Unlock()
	if planner == nil {
		return rejectedResponseFor(request.SchemaVersion, fmt.Errorf("host scope planning is not supported"))
	}
	plan, err := planner.Plan(context.Background(), request.OperationID, request.ManifestDigest, request.RequireFull)
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	if err := daemon.store.SaveScopePlan(plan); err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	// Return the persisted plan rather than a freshly generated equivalent plan.
	// GeneratedAt is not part of the digest, while the on-disk copy is the exact
	// evidence the subsequent plan-bound start must match.
	persisted, err := daemon.store.LoadScopePlan(request.OperationID)
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	return Response{SchemaVersion: request.SchemaVersion, Accepted: true, ScopePlan: &persisted}
}

func (daemon *Daemon) acceptConfirmation(request Request) Response {
	daemon.mu.Lock()
	releaseLockAfterConfirmation := false
	defer func() {
		daemon.mu.Unlock()
		if releaseLockAfterConfirmation {
			daemon.releaseDeploymentLock()
		}
	}()
	if _, running := daemon.executing[request.OperationID]; running {
		return rejectedResponseFor(request.SchemaVersion, ErrOperationInProgress)
	}
	plan, err := daemon.validatePlanBoundRequest(request)
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	journal, receipt, err := daemon.store.ReconcileReceipt(request.OperationID, request.ManifestDigest)
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	if err := validateJournalRequestScope(journal, request); err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	if journal.Stage != StageVerifying && journal.Stage != StageSucceeded {
		return rejectedResponseFor(request.SchemaVersion, fmt.Errorf("host execution has not reached verification"))
	}
	if err := validateReceiptAgainstScopePlan(receipt, plan); err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}

	state, stateErr := daemon.store.LoadConfirmedDeploymentState()
	// A confirmation that changes host state must be serialized with lifecycle
	// scripts. A terminal replay with the exact already-confirmed identity is
	// read/idempotent and does not need to reacquire a lock that was released
	// after the original confirmation.
	if journal.Stage != StageSucceeded || stateErr != nil || !sameConfirmationIdentity(state, plan) {
		if err := daemon.requireDeploymentLockLocked(request.OperationID); err != nil {
			return rejectedResponseFor(request.SchemaVersion, err)
		}
	}
	if stateErr == nil && sameConfirmationIdentity(state, plan) {
		if journal.Stage != StageSucceeded {
			journal, err = daemon.store.Checkpoint(request.OperationID, request.ManifestDigest, StageSucceeded, "", nil)
			if err != nil {
				return rejectedResponseFor(request.SchemaVersion, err)
			}
		}
		releaseLockAfterConfirmation = true
		return Response{SchemaVersion: request.SchemaVersion, Accepted: true, Journal: journal, ConfirmedDeploymentState: &state}
	}
	if stateErr != nil && !errors.Is(stateErr, ErrConfirmedStateNotFound) {
		return rejectedResponseFor(request.SchemaVersion, stateErr)
	}
	var liveObservation *RuntimeObservation
	if plan.ExecutionMode == ExecutionModeFull {
		verifier, ok := daemon.scopePlanner.(FullDeploymentConfirmation)
		if !ok {
			return rejectedResponseFor(request.SchemaVersion, fmt.Errorf("full confirmation requires live deployment observation"))
		}
		observation, observeErr := verifier.ObserveFullDeployment(context.Background(), plan)
		if observeErr != nil {
			return rejectedResponseFor(request.SchemaVersion, observeErr)
		}
		liveObservation = &observation
	}
	if plan.ExecutionMode == ExecutionModeFrontendOnly {
		promoter, ok := daemon.executor.(interface {
			PromoteFrontendOnlyComposeOverride(*JournalStore, string, *releasemanifest.Manifest) error
		})
		if !ok {
			return rejectedResponseFor(request.SchemaVersion, fmt.Errorf("frontend-only confirmation requires a scoped Compose executor"))
		}
		manifestPath, pathErr := daemon.store.ManifestPath(request.ManifestDigest)
		if pathErr != nil {
			return rejectedResponseFor(request.SchemaVersion, pathErr)
		}
		manifest, loadErr := loadManifestWithLegacyCompatibility(manifestPath)
		if loadErr != nil || manifest.Digest() != request.ManifestDigest {
			return rejectedResponseFor(request.SchemaVersion, ErrManifestMismatch)
		}
		if promoteErr := promoter.PromoteFrontendOnlyComposeOverride(daemon.store, request.OperationID, manifest); promoteErr != nil {
			return rejectedResponseFor(request.SchemaVersion, promoteErr)
		}
	}
	if liveObservation != nil {
		state, err = confirmedDeploymentStateFromPlan(plan, time.Now().UTC(), *liveObservation)
	} else {
		state, err = confirmedDeploymentStateFromPlan(plan, time.Now().UTC())
	}
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	if err := daemon.store.SaveConfirmedDeploymentState(state); err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	if journal.Stage != StageSucceeded {
		journal, err = daemon.store.Checkpoint(request.OperationID, request.ManifestDigest, StageSucceeded, "", nil)
		if err != nil {
			return rejectedResponseFor(request.SchemaVersion, err)
		}
	}
	releaseLockAfterConfirmation = true
	return Response{SchemaVersion: request.SchemaVersion, Accepted: true, Journal: journal, ConfirmedDeploymentState: &state}
}

func (daemon *Daemon) validatePlanBoundRequest(request Request) (ScopePlan, error) {
	if daemon == nil || daemon.store == nil {
		return ScopePlan{}, fmt.Errorf("upgrader daemon is not configured")
	}
	if request.SchemaVersion != ScopedRequestSchema {
		return ScopePlan{}, nil
	}
	plan, err := daemon.store.LoadScopePlan(request.OperationID)
	if err != nil {
		return ScopePlan{}, err
	}
	if plan.ManifestDigest != request.ManifestDigest || !sameScopeEvidence(plan.ExecutionMode, plan.PlanDigest, plan.BaselineStateDigest, plan.TouchedServices, plan.ConfirmedDeploymentVersion, request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion) {
		return ScopePlan{}, ErrScopePlanStale
	}
	return plan, nil
}

func journalForRequest(request Request, now time.Time) Journal {
	journal := Journal{
		SchemaVersion:  JournalSchema,
		OperationID:    request.OperationID,
		ManifestDigest: request.ManifestDigest,
		Stage:          StageQueued,
		StartedAt:      now,
		UpdatedAt:      now,
		StageUpdatedAt: now,
	}
	if request.SchemaVersion == ScopedRequestSchema {
		journal.SchemaVersion = ScopedJournalSchema
		journal.ExecutionMode = request.ExecutionMode
		journal.PlanDigest = request.PlanDigest
		journal.BaselineStateDigest = request.BaselineStateDigest
		journal.TouchedServices = append([]string(nil), request.TouchedServices...)
		journal.ConfirmedDeploymentVersion = request.ConfirmedDeploymentVersion
	}
	return journal
}

func validateJournalRequestScope(journal Journal, request Request) error {
	if request.SchemaVersion == RequestSchema {
		if journal.SchemaVersion != JournalSchema {
			return ErrScopePlanStale
		}
		return nil
	}
	if journal.SchemaVersion != ScopedJournalSchema || !sameScopeEvidence(journal.ExecutionMode, journal.PlanDigest, journal.BaselineStateDigest, journal.TouchedServices, journal.ConfirmedDeploymentVersion, request.ExecutionMode, request.PlanDigest, request.BaselineStateDigest, request.TouchedServices, request.ConfirmedDeploymentVersion) {
		return ErrScopePlanStale
	}
	return nil
}

func validateReceiptAgainstScopePlan(receipt Receipt, plan ScopePlan) error {
	if receipt.SchemaVersion != ScopedJournalSchema || !sameScopeEvidence(receipt.ExecutionMode, receipt.PlanDigest, receipt.BaselineStateDigest, receipt.TouchedServices, receipt.ConfirmedDeploymentVersion, plan.ExecutionMode, plan.PlanDigest, plan.BaselineStateDigest, plan.TouchedServices, plan.ConfirmedDeploymentVersion) {
		return ErrReceiptMismatch
	}
	if plan.ExecutionMode == ExecutionModeFrontendOnly && !sameStringSlice(receipt.Services, []string{FrontendOnlyService}) {
		return ErrReceiptMismatch
	}
	for service, digest := range receipt.ObservedImages {
		if digest != plan.Candidate.componentDigest("runtime."+service) {
			return ErrReceiptMismatch
		}
	}
	return nil
}

func sameConfirmationIdentity(state ConfirmedDeploymentState, plan ScopePlan) bool {
	if state.OperationID != plan.OperationID || state.ManifestDigest != plan.ManifestDigest || state.CompositionDigest != plan.Candidate.CompositionDigest || state.ReleaseVersion != plan.Candidate.ReleaseVersion || state.Capabilities != plan.Candidate.Capabilities {
		return false
	}
	if plan.ObservedNginxConfigDigest != "" && state.NginxConfigDigest != plan.ObservedNginxConfigDigest {
		return false
	}
	return sameDeploymentComponents(state.Components, plan.Candidate.Components)
}

func sameDeploymentComponents(left, right []DeploymentComponent) bool {
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

func responseSchemaFor(request Request) int {
	if validRequestSchema(request.SchemaVersion) {
		return request.SchemaVersion
	}
	return RequestSchema
}

func executionResponse(request Request, journal Journal, replayed, repaired bool) Response {
	return Response{SchemaVersion: request.SchemaVersion, Accepted: true, Replayed: replayed, Repaired: repaired, Journal: journal}
}

// acceptStopLocked handles the one mutating socket action that does not start
// execution. The caller holds daemon.mu; cancellation is signalled immediately
// while the executor remains responsible for writing the durable terminal
// checkpoint after its current command unwinds.
func (daemon *Daemon) acceptStopLocked(request Request, current Journal, currentErr error, history Journal, historyErr error) Response {
	if historyErr == nil && history.ManifestDigest != request.ManifestDigest {
		return rejectedResponseFor(request.SchemaVersion, ErrReplayDigestMismatch)
	}
	journal := history
	if historyErr != nil {
		if currentErr != nil || current.OperationID != request.OperationID || current.ManifestDigest != request.ManifestDigest {
			return rejectedResponseFor(request.SchemaVersion, ErrResumeNotFound)
		}
		journal = current
	} else if currentErr == nil && current.OperationID == request.OperationID && current.UpdatedAt.After(history.UpdatedAt) {
		journal = current
	}
	if err := validateJournalRequestScope(journal, request); err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	if currentErr == nil && current.OperationID != request.OperationID && !IsTerminal(journal.Stage) {
		return rejectedResponseFor(request.SchemaVersion, ErrOperationInProgress)
	}
	if IsTerminal(journal.Stage) {
		return executionResponse(request, journal, true, false)
	}
	daemon.cancelRequested[request.OperationID] = struct{}{}
	if cancel, exists := daemon.executing[request.OperationID]; exists && cancel != nil {
		cancel()
		return executionResponse(request, journal, false, false)
	}
	updated, err := daemon.stopJournal(request, journal)
	if err != nil {
		return rejectedResponseFor(request.SchemaVersion, err)
	}
	delete(daemon.cancelRequested, request.OperationID)
	return executionResponse(request, updated, false, false)
}

func (daemon *Daemon) stopJournal(request Request, current Journal) (Journal, error) {
	if IsTerminal(current.Stage) {
		// A stop can race with the executor's final checkpoint. Never downgrade a
		// durable success or an already-classified recovery outcome.
		return current, nil
	}
	stage := StageNeedsAttention
	if migrationEvidenceForJournal(current) {
		stage = StageNeedsRecovery
	}
	diagnostic := "upgrade stopped by operator"
	if stage == StageNeedsRecovery {
		diagnostic = "upgrade stopped after the migration boundary; manual recovery is required"
	}
	return daemon.store.Checkpoint(request.OperationID, request.ManifestDigest, stage, diagnostic, nil)
}

// launchExecutionLocked marks the operation before launching a goroutine. The
// caller must hold daemon.mu; all disk checkpoint transitions belong to the
// executor so a future Compose implementation can make each phase explicit.
func (daemon *Daemon) launchExecutionLocked(ctx context.Context, request Request) {
	if daemon.executor == nil {
		return
	}
	if ctx == nil {
		ctx = daemon.runCtx
		if ctx == nil {
			ctx = context.Background()
		}
	}
	if _, exists := daemon.executing[request.OperationID]; exists {
		return
	}
	executionCtx, cancel := context.WithCancel(ctx)
	daemon.executing[request.OperationID] = cancel
	daemon.wg.Add(1)
	go func() {
		defer daemon.wg.Done()
		// The deployment lock is taken before any Compose mutation and released
		// when the operation reaches a stage that needs no recovery. Waiting here
		// rather than in accept() keeps a lifecycle command and an Upgrade
		// Operation mutually exclusive without failing either of them.
		if err := daemon.holdDeploymentLock(executionCtx, request.OperationID); err != nil {
			if daemon.consumeCancellation(request.OperationID) {
				_, _ = daemon.stopJournal(request, currentJournalOrEmpty(daemon.store, request))
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// A daemon/server shutdown cancels the execution context but is not
				// an operator stop. Leave the journal active for resume/recovery.
			} else {
				_ = daemon.failJournal(request, err)
			}
		} else if err := daemon.revalidateScopeAfterLock(executionCtx, request); err != nil {
			// A no-interruption plan is not permission to broaden scope after
			// the Server skipped cancellation. Record a bounded recheck result
			// and stop before any Compose command is constructed or executed.
			_ = daemon.failScopePlanForRecheck(request)
		} else if err := daemon.executor.Execute(executionCtx, request, daemon.store); err != nil {
			if daemon.consumeCancellation(request.OperationID) {
				_, _ = daemon.stopJournal(request, currentJournalOrEmpty(daemon.store, request))
			} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// Preserve the last durable checkpoint when the host process is
				// stopping; the recovery supervisor will decide the final outcome.
			} else {
				_ = daemon.failJournal(request, err)
			}
		} else if daemon.consumeCancellation(request.OperationID) {
			_, _ = daemon.stopJournal(request, currentJournalOrEmpty(daemon.store, request))
		}
		cancel()
		daemon.finishExecution(request.OperationID)
		daemon.mu.Lock()
		delete(daemon.executing, request.OperationID)
		delete(daemon.cancelRequested, request.OperationID)
		daemon.mu.Unlock()
	}()
}

func (daemon *Daemon) revalidateScopeAfterLock(ctx context.Context, request Request) error {
	if request.SchemaVersion != ScopedRequestSchema || request.ExecutionMode != ExecutionModeFrontendOnly {
		return nil
	}
	plan, err := daemon.validatePlanBoundRequest(request)
	if err != nil {
		return fmt.Errorf("%w: persisted scope plan changed", ErrScopePlanStale)
	}
	daemon.mu.Lock()
	planner := daemon.scopePlanner
	daemon.mu.Unlock()
	if planner == nil {
		return fmt.Errorf("%w: host scope planner is unavailable", ErrScopePlanStale)
	}
	if err := planner.Revalidate(ctx, plan); err != nil {
		if errors.Is(err, ErrScopePlanStale) {
			return err
		}
		return fmt.Errorf("%w: host scope revalidation failed", ErrScopePlanStale)
	}
	return nil
}

func (daemon *Daemon) failScopePlanForRecheck(request Request) error {
	if daemon == nil || daemon.store == nil {
		return fmt.Errorf("upgrader daemon is not configured")
	}
	current, err := daemon.store.LoadCurrent()
	if err != nil {
		return err
	}
	if current.OperationID != request.OperationID || current.ManifestDigest != request.ManifestDigest {
		return ErrReplayDigestMismatch
	}
	if IsTerminal(current.Stage) {
		return nil
	}
	_, err = daemon.store.Checkpoint(request.OperationID, request.ManifestDigest, StageFailed, "frontend-only scope plan requires recheck", nil)
	return err
}

func (daemon *Daemon) consumeCancellation(operationID string) bool {
	daemon.mu.Lock()
	defer daemon.mu.Unlock()
	_, requested := daemon.cancelRequested[operationID]
	return requested
}

func currentJournalOrEmpty(store *JournalStore, request Request) Journal {
	if store != nil {
		if journal, err := store.LoadCurrent(); err == nil && journal.OperationID == request.OperationID && journal.ManifestDigest == request.ManifestDigest {
			return journal
		}
	}
	return Journal{OperationID: request.OperationID, ManifestDigest: request.ManifestDigest, Stage: StageQueued}
}

// holdDeploymentLock acquires the shared deployment lock for one Operation.
func (daemon *Daemon) holdDeploymentLock(ctx context.Context, operationID string) error {
	if daemon == nil || daemon.store == nil {
		return fmt.Errorf("upgrader daemon is not configured")
	}
	daemon.mu.Lock()
	held := daemon.deploymentLock
	daemon.mu.Unlock()
	if held != nil && held.OperationID() == operationID {
		return nil
	}
	lock, err := AcquireDeploymentLockContext(ctx, daemon.store.DeploymentRoot(), operationID, 0)
	if err != nil {
		return err
	}
	daemon.mu.Lock()
	daemon.deploymentLock = lock
	daemon.mu.Unlock()
	return nil
}

// releaseDeploymentLock drops the shared lock without touching it when another
// owner has taken it over in the meantime.
func (daemon *Daemon) releaseDeploymentLock() {
	daemon.mu.Lock()
	lock := daemon.deploymentLock
	daemon.deploymentLock = nil
	daemon.mu.Unlock()
	if lock != nil {
		_ = lock.Release()
	}
}

// finishExecution decides whether the shared lock can be released. A journal
// that needs recovery keeps the lock as a fence so a lifecycle command cannot
// mutate a deployment whose database outcome is still unknown.
func (daemon *Daemon) finishExecution(operationID string) {
	daemon.mu.Lock()
	lock := daemon.deploymentLock
	daemon.mu.Unlock()
	if lock == nil || lock.OperationID() != operationID {
		return
	}
	journal, err := daemon.store.LoadCurrent()
	if err != nil {
		// The journal cannot be read, so the outcome is unknown: keep the fence.
		_ = lock.MarkRecoveryFence()
		return
	}
	if requiresDeploymentFence(journal) {
		_ = lock.MarkRecoveryFence()
		return
	}
	daemon.releaseDeploymentLock()
}

// requiresDeploymentFence keeps the shared lifecycle lock while a v2 host
// receipt is waiting for Server confirmation. A staged frontend override is
// not safe to promote after another lifecycle command has changed the
// deployment, so StageVerifying remains fenced until ActionConfirm succeeds.
func requiresDeploymentFence(journal Journal) bool {
	if requiresRecoveryFence(journal.Stage) {
		return true
	}
	return journal.SchemaVersion == ScopedJournalSchema && journal.Stage == StageVerifying
}

// ensureRecoveryFence synchronously takes the shared lock when the persisted
// journal still needs recovery after a restart. Serve must finish this step
// before it accepts requests, otherwise confirmation could race a lifecycle
// command that currently owns the deployment lock.
func (daemon *Daemon) ensureRecoveryFence(ctx context.Context) error {
	if daemon == nil || daemon.store == nil {
		return nil
	}
	journal, err := daemon.store.LoadCurrent()
	if err != nil {
		return err
	}
	if !requiresDeploymentFence(journal) {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Reuse a lock already acquired by a resumed executor for this operation.
	// The lock is represented by a directory rather than a kernel handle, so
	// releasing a second same-operation wrapper would accidentally remove the
	// shared fence from underneath the first wrapper.
	daemon.mu.Lock()
	held := daemon.deploymentLock
	daemon.mu.Unlock()
	if held != nil {
		if held.OperationID() != journal.OperationID {
			return ErrDeploymentLockHeld
		}
		if err := held.MarkRecoveryFence(); err != nil {
			return err
		}
		return nil
	}

	lock, err := AcquireDeploymentLockContext(ctx, daemon.store.DeploymentRoot(), journal.OperationID, 100*time.Millisecond)
	if err != nil {
		return err
	}
	if err := lock.MarkRecoveryFence(); err != nil {
		_ = lock.Release()
		return err
	}
	daemon.mu.Lock()
	if daemon.deploymentLock == nil {
		daemon.deploymentLock = lock
		daemon.mu.Unlock()
		return nil
	}
	if daemon.deploymentLock.OperationID() == journal.OperationID {
		// Keep the existing wrapper and leave the shared directory in place.
		daemon.mu.Unlock()
		return nil
	}
	daemon.mu.Unlock()
	// Do not remove a lock that may now belong to another operation.
	return ErrDeploymentLockHeld
}

// requireDeploymentLockLocked proves both in-memory and host-visible
// ownership before confirmation mutates the persistent deployment baseline.
// The caller must hold daemon.mu.
func (daemon *Daemon) requireDeploymentLockLocked(operationID string) error {
	if daemon == nil || daemon.store == nil {
		return ErrDeploymentLockNotOwned
	}
	lock := daemon.deploymentLock
	if lock == nil || lock.OperationID() != operationID {
		return ErrDeploymentLockNotOwned
	}
	metadata, exists, err := ReadDeploymentLock(daemon.store.DeploymentRoot())
	if err != nil {
		return err
	}
	if !exists || metadata.Owner != deploymentLockOwnerUpgrade || metadata.OperationID != operationID {
		return ErrDeploymentLockNotOwned
	}
	return nil
}

func (daemon *Daemon) failJournal(request Request, cause error) error {
	_ = cause
	// Executor errors may contain command output, host paths, or credentials.
	// Persist only a fixed diagnostic; the durable journal is later projected to
	// API clients and must remain safe even when an executor is misbehaving.
	diagnostic := "upgrade execution failed"
	now := time.Now().UTC()
	stage := StageFailed
	repairStage := StageQueued
	startedAt := now
	var currentJournal Journal
	haveCurrent := false
	if current, err := daemon.store.LoadCurrent(); err == nil {
		// An executor must never replace a newer operation's checkpoint (or a
		// checkpoint with a different digest) while reporting its own failure.
		// This can happen after an external restart/tamper and would otherwise
		// destroy the only durable recovery evidence for the current deployment.
		if current.OperationID != request.OperationID || current.ManifestDigest != request.ManifestDigest {
			return ErrReplayDigestMismatch
		}
		currentJournal = current
		haveCurrent = true
		startedAt = current.StartedAt
		repairStage = repairStageFor(current.Stage)
		// Once migration or post-migration verification has begun, a generic
		// executor error cannot prove that the database is unchanged. Preserve
		// the stronger recovery-required classification rather than downgrading
		// it to a retryable pre-migration failure.
		switch current.Stage {
		case StageMigrating:
			// The explicit migrating checkpoint is an irreversible database
			// boundary, even when migration identity has not reached disk yet.
			stage = StageNeedsRecovery
		case StageRestarting, StageAgentVerifying, StageVerifying:
			// These checkpoints are also used by no-migration releases. Only
			// classify them as recovery-required when the journal carries
			// migration evidence; otherwise the operation remains retryable.
			if migrationEvidenceForJournal(current) {
				stage = StageNeedsRecovery
			}
		case StageSucceeded, StageFailed, StageNeedsRecovery, StageNeedsAttention:
			return nil
		}
		if migrationEvidenceForJournal(current) {
			// A migration identity/status is durable evidence even if the stage
			// checkpoint was written just before the executor returned an error.
			// Do not downgrade that operation to an ordinary retryable failure.
			stage = StageNeedsRecovery
		}
	} else if !errors.Is(err, ErrJournalNotFound) {
		// A corrupt current checkpoint is not an empty deployment. Preserve it
		// for operator recovery instead of replacing it with a synthesized error.
		return err
	}
	journal := journalForRequest(request, startedAt)
	journal.Stage = stage
	journal.UpdatedAt = now
	journal.StageUpdatedAt = now
	journal.CompletedAt = &now
	journal.RepairStage = repairStage
	journal.Diagnostic = diagnostic
	if haveCurrent {
		journal.SchemaVersion = currentJournal.SchemaVersion
		journal.ExecutionMode = currentJournal.ExecutionMode
		journal.PlanDigest = currentJournal.PlanDigest
		journal.BaselineStateDigest = currentJournal.BaselineStateDigest
		journal.TouchedServices = append([]string(nil), currentJournal.TouchedServices...)
		journal.ConfirmedDeploymentVersion = currentJournal.ConfirmedDeploymentVersion
		// Preserve migration evidence when an I/O or receipt error escapes the
		// executor. Replacing the checkpoint must not erase the identity/status
		// needed to distinguish a recoverable pre-migration failure from an
		// unknown database outcome.
		journal.MigrationID = currentJournal.MigrationID
		journal.MigrationChecksum = currentJournal.MigrationChecksum
		journal.MigrationStatus = currentJournal.MigrationStatus
	}
	return daemon.store.Save(journal)
}

func (daemon *Daemon) writeResponse(connection net.Conn, response Response) {
	_ = connection.SetWriteDeadline(time.Now().Add(requestTimeout))
	encoder := json.NewEncoder(connection)
	_ = encoder.Encode(response)
}

func rejectedResponse(err error) Response {
	return rejectedResponseFor(RequestSchema, err)
}

func rejectedResponseFor(schemaVersion int, err error) Response {
	if !validRequestSchema(schemaVersion) {
		schemaVersion = RequestSchema
	}
	message := "upgrader request rejected"
	if err != nil {
		// Errors exposed over the socket are stable, bounded diagnostics. Avoid
		// returning filesystem paths or subprocess output to the Server.
		switch {
		case errors.Is(err, ErrReplayDigestMismatch):
			message = ErrReplayDigestMismatch.Error()
		case errors.Is(err, ErrOperationInProgress):
			message = ErrOperationInProgress.Error()
		case errors.Is(err, ErrResumeNotFound):
			message = ErrResumeNotFound.Error()
		case errors.Is(err, ErrManifestMismatch):
			message = ErrManifestMismatch.Error()
		case errors.Is(err, ErrRepairRequired):
			message = ErrRepairRequired.Error()
		case errors.Is(err, ErrRepairNotAllowed):
			message = ErrRepairNotAllowed.Error()
		case errors.Is(err, ErrDeploymentLockNotOwned):
			message = ErrDeploymentLockNotOwned.Error()
		default:
			message = err.Error()
		}
	}
	return Response{SchemaVersion: schemaVersion, Error: message}
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect upgrader socket: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to replace non-socket upgrader path")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale upgrader socket: %w", err)
	}
	return nil
}

// Client sends one fixed-schema request over the private deployment socket.
// The socket path is derived from deploymentRoot and cannot be supplied by a
// Server request.
type Client struct {
	store *JournalStore
}

func NewClient(deploymentRoot string) (*Client, error) {
	store, err := NewJournalStore(deploymentRoot)
	if err != nil {
		return nil, err
	}
	return &Client{store: store}, nil
}

func (client *Client) Send(ctx context.Context, request Request) (Response, error) {
	if client == nil || client.store == nil {
		return Response{}, fmt.Errorf("upgrader client is not configured")
	}
	if err := request.Validate(); err != nil {
		return Response{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "unix", client.store.SocketPath())
	if err != nil {
		return Response{}, fmt.Errorf("connect to upgrader socket: %w", err)
	}
	defer func() {
		_ = connection.Close()
	}()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	} else {
		_ = connection.SetDeadline(time.Now().Add(requestTimeout))
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return Response{}, err
	}
	encoded = append(encoded, '\n')
	if _, err := connection.Write(encoded); err != nil {
		return Response{}, fmt.Errorf("send upgrader request: %w", err)
	}
	if unixConnection, ok := connection.(*net.UnixConn); ok {
		_ = unixConnection.CloseWrite()
	}
	var response Response
	if err := decodeStrictFromReader(bufio.NewReader(io.LimitReader(connection, MaxRequestBytes+1)), &response); err != nil {
		return Response{}, fmt.Errorf("decode upgrader response: %w", err)
	}
	if err := response.ValidateFor(request); err != nil {
		return Response{}, fmt.Errorf("validate upgrader response: %w", err)
	}
	if response.Accepted && !isZeroJournal(response.Journal) && (response.Journal.OperationID != request.OperationID || response.Journal.ManifestDigest != request.ManifestDigest) {
		return Response{}, ErrReplayDigestMismatch
	}
	if response.Error != "" {
		return response, errors.New(response.Error)
	}
	return response, nil
}

func decodeStrictFromReader(reader *bufio.Reader, target any) error {
	data, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(data) > MaxRequestBytes {
		return fmt.Errorf("response is too large")
	}
	return decodeStrict(data, target)
}
