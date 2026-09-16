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
	manifest, err := releasemanifest.Load(manifestPath)
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
	listener       net.Listener
	lock           *fileLock

	mu        sync.Mutex
	closing   bool
	running   bool
	runCtx    context.Context
	runCancel context.CancelFunc
	executing map[string]struct{}
	wg        sync.WaitGroup
}

func NewDaemon(store *JournalStore, executor Executor) *Daemon {
	return &Daemon{
		store:          store,
		executor:       executor,
		verifyManifest: VerifyDeploymentManifest,
		executing:      make(map[string]struct{}),
	}
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
	request := Request{
		SchemaVersion:  RequestSchema,
		OperationID:    journal.OperationID,
		Action:         ActionResume,
		ManifestDigest: journal.ManifestDigest,
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
		return rejectedResponse(err)
	}
	daemon.mu.Lock()

	current, currentErr := daemon.store.LoadCurrent()
	if currentErr != nil && !errors.Is(currentErr, ErrJournalNotFound) {
		// A damaged current checkpoint is not equivalent to an empty deployment.
		// Refuse new work until the operator repairs the journal, otherwise a
		// fresh request could overwrite the only durable recovery evidence.
		daemon.mu.Unlock()
		return rejectedResponse(currentErr)
	}
	history, historyErr := daemon.store.LoadOperation(request.OperationID)
	if historyErr != nil && !errors.Is(historyErr, ErrJournalNotFound) {
		daemon.mu.Unlock()
		return rejectedResponse(historyErr)
	}
	if historyErr == nil {
		if history.ManifestDigest != request.ManifestDigest {
			daemon.mu.Unlock()
			return rejectedResponse(ErrReplayDigestMismatch)
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
			return rejectedResponse(ErrOperationInProgress)
		}
		if IsTerminal(history.Stage) {
			if request.Action == ActionRepair {
				if _, exists := daemon.executing[request.OperationID]; exists {
					daemon.mu.Unlock()
					return rejectedResponse(ErrRepairNotAllowed)
				}
				if daemon.verifyManifest != nil {
					if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
						daemon.mu.Unlock()
						return rejectedResponse(ErrManifestMismatch)
					}
				}
				repaired, err := daemon.store.ResetForRepair(request.OperationID, request.ManifestDigest)
				if err != nil {
					daemon.mu.Unlock()
					return rejectedResponse(err)
				}
				daemon.launchExecutionLocked(context.Background(), request)
				daemon.mu.Unlock()
				return Response{SchemaVersion: RequestSchema, Accepted: true, Repaired: true, Journal: repaired}
			}
			if request.Action == ActionResume {
				daemon.mu.Unlock()
				return rejectedResponse(ErrRepairRequired)
			}
			daemon.mu.Unlock()
			return Response{SchemaVersion: RequestSchema, Accepted: true, Replayed: true, Journal: history}
		}
		if request.Action == ActionRepair {
			daemon.mu.Unlock()
			return rejectedResponse(ErrRepairNotAllowed)
		}
		if request.Action != ActionResume && request.Action != ActionStart {
			daemon.mu.Unlock()
			return rejectedResponse(ErrResumeNotFound)
		}
		if _, exists := daemon.executing[request.OperationID]; exists {
			daemon.mu.Unlock()
			return Response{SchemaVersion: RequestSchema, Accepted: true, Replayed: true, Journal: history}
		}
		if daemon.verifyManifest != nil {
			if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
				daemon.mu.Unlock()
				return rejectedResponse(ErrManifestMismatch)
			}
		}
		daemon.launchExecutionLocked(context.Background(), request)
		daemon.mu.Unlock()
		journal := history
		return Response{SchemaVersion: RequestSchema, Accepted: true, Replayed: true, Journal: journal}
	}
	if currentErr == nil && !IsTerminal(current.Stage) {
		daemon.mu.Unlock()
		return rejectedResponse(ErrOperationInProgress)
	}
	if request.Action == ActionResume {
		daemon.mu.Unlock()
		return rejectedResponse(ErrResumeNotFound)
	}
	if request.Action == ActionRepair {
		daemon.mu.Unlock()
		return rejectedResponse(ErrRepairNotAllowed)
	}
	if daemon.verifyManifest != nil {
		if err := daemon.verifyRequestManifest(request.ManifestDigest); err != nil {
			daemon.mu.Unlock()
			return rejectedResponse(ErrManifestMismatch)
		}
	}
	now := time.Now().UTC()
	journal := Journal{
		SchemaVersion:  JournalSchema,
		OperationID:    request.OperationID,
		ManifestDigest: request.ManifestDigest,
		Stage:          StageQueued,
		StartedAt:      now,
		UpdatedAt:      now,
	}
	if err := daemon.store.Save(journal); err != nil {
		daemon.mu.Unlock()
		return rejectedResponse(err)
	}
	response := Response{SchemaVersion: RequestSchema, Accepted: true, Journal: journal}
	daemon.launchExecutionLocked(context.Background(), request)
	daemon.mu.Unlock()
	return response
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
	daemon.executing[request.OperationID] = struct{}{}
	daemon.wg.Add(1)
	go func() {
		defer daemon.wg.Done()
		err := daemon.executor.Execute(ctx, request, daemon.store)
		if err != nil {
			_ = daemon.failJournal(request, err)
		}
		daemon.mu.Lock()
		delete(daemon.executing, request.OperationID)
		daemon.mu.Unlock()
	}()
}

func (daemon *Daemon) failJournal(request Request, cause error) error {
	diagnostic := "upgrade execution failed"
	if cause != nil {
		// Do not copy arbitrary command output into persistent state.
		if ValidateDiagnostic(cause.Error()) == nil {
			diagnostic = cause.Error()
		}
	}
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
	journal := Journal{
		SchemaVersion:  JournalSchema,
		OperationID:    request.OperationID,
		ManifestDigest: request.ManifestDigest,
		Stage:          stage,
		StartedAt:      startedAt,
		UpdatedAt:      now,
		CompletedAt:    &now,
		RepairStage:    repairStage,
		Diagnostic:     diagnostic,
	}
	if haveCurrent {
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
		default:
			message = err.Error()
		}
	}
	return Response{SchemaVersion: RequestSchema, Error: message}
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
	if err := response.Validate(); err != nil {
		return Response{}, fmt.Errorf("validate upgrader response: %w", err)
	}
	if response.Accepted && (response.Journal.OperationID != request.OperationID || response.Journal.ManifestDigest != request.ManifestDigest) {
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
