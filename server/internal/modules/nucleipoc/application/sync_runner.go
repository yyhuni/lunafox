package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

type SyncRunner struct {
	store              Store
	resolver           SourceResolver
	workspace          Workspace
	git                GitRunner
	queue              chan uuid.UUID
	startOnce          sync.Once
	done               chan struct{}
	ctx                context.Context
	runsMu             sync.Mutex
	runs               map[uuid.UUID]*syncRun
	leaseOwner         string
	leaseDuration      time.Duration
	leaseRenewInterval time.Duration
}

type syncRun struct {
	cancel    context.CancelFunc
	requested bool
}

const (
	syncCancellationPollInterval = 500 * time.Millisecond
	syncTaskLeaseDuration        = 30 * time.Second
	syncTaskLeaseRenewInterval   = 5 * time.Second
)

func NewSyncRunner(store Store, resolver SourceResolver, workspace Workspace, git GitRunner) *SyncRunner {
	return &SyncRunner{
		store:              store,
		resolver:           resolver,
		workspace:          workspace,
		git:                git,
		queue:              make(chan uuid.UUID, 8),
		done:               make(chan struct{}),
		runs:               make(map[uuid.UUID]*syncRun),
		leaseOwner:         "nuclei-poc-sync-" + uuid.NewString(),
		leaseDuration:      syncTaskLeaseDuration,
		leaseRenewInterval: syncTaskLeaseRenewInterval,
	}
}

func (runner *SyncRunner) Start(ctx context.Context) {
	if runner == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runner.startOnce.Do(func() {
		runner.ctx = ctx
		go runner.loop(ctx)
	})
}

func (runner *SyncRunner) Done() <-chan struct{} {
	if runner == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return runner.done
}

func (runner *SyncRunner) Enqueue(taskID uuid.UUID) {
	if runner == nil || taskID == uuid.Nil {
		return
	}
	select {
	case runner.queue <- taskID:
	default:
		go runner.RunOnce(context.Background(), taskID)
	}
}

// Cancel requests the in-process run to stop. The durable CANCELLING state is
// written by the application service before this fast path is invoked; a
// runner in another Server process will observe that state through polling.
func (runner *SyncRunner) Cancel(taskID uuid.UUID) {
	if runner == nil || taskID == uuid.Nil {
		return
	}
	runner.runsMu.Lock()
	run := runner.runs[taskID]
	if run != nil {
		run.requested = true
	}
	runner.runsMu.Unlock()
	if run != nil && run.cancel != nil {
		run.cancel()
	}
}

// stopRun ends local execution without asserting that an operator requested a
// cancellation. Lease loss and a terminal result written by recovery must stop
// the process tree, but must not overwrite the durable terminal winner.
func (runner *SyncRunner) stopRun(taskID uuid.UUID) {
	if runner == nil || taskID == uuid.Nil {
		return
	}
	runner.runsMu.Lock()
	run := runner.runs[taskID]
	runner.runsMu.Unlock()
	if run != nil && run.cancel != nil {
		run.cancel()
	}
}

func (runner *SyncRunner) registerRun(taskID uuid.UUID, run *syncRun) bool {
	runner.runsMu.Lock()
	defer runner.runsMu.Unlock()
	if _, exists := runner.runs[taskID]; exists {
		return false
	}
	runner.runs[taskID] = run
	return true
}

func (runner *SyncRunner) unregisterRun(taskID uuid.UUID, run *syncRun) {
	runner.runsMu.Lock()
	defer runner.runsMu.Unlock()
	if current, ok := runner.runs[taskID]; ok && current == run {
		delete(runner.runs, taskID)
	}
}

func (runner *SyncRunner) runCancellationRequested(taskID uuid.UUID) bool {
	runner.runsMu.Lock()
	run := runner.runs[taskID]
	requested := run != nil && run.requested
	runner.runsMu.Unlock()
	if requested {
		return true
	}
	return runner.cancellationPersisted(taskID)
}

func (runner *SyncRunner) cancellationPersisted(taskID uuid.UUID) bool {
	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	task, err := runner.store.GetSyncTask(checkCtx, taskID)
	return err == nil && task != nil && (task.State == domain.SyncTaskCancelling || task.State == domain.SyncTaskCancelled)
}

// watchCancellation bridges requests served by a different Server process to
// the local context. There is at most one active sync, so a bounded poll keeps
// the cross-process path simple without introducing a second queue/transport.
func (runner *SyncRunner) watchCancellation(ctx context.Context, taskID uuid.UUID) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(syncCancellationPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				task, err := runner.store.GetSyncTask(checkCtx, taskID)
				cancel()
				if err != nil || task == nil {
					continue
				}
				if task.State == domain.SyncTaskCancelling {
					runner.Cancel(taskID)
					return
				}
				if task.State.Terminal() {
					runner.stopRun(taskID)
					return
				}
			}
		}
	}()
	return done
}

// watchLease keeps recovery from treating a live runner as interrupted. If the
// durable lease cannot be renewed, the process must stop before another Server
// is allowed to recover the task after expiry.
func (runner *SyncRunner) watchLease(ctx context.Context, taskID uuid.UUID) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(runner.leaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				renewCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err := runner.store.RenewTaskLease(renewCtx, taskID, runner.leaseOwner, time.Now().UTC(), runner.leaseDuration)
				cancel()
				if err != nil {
					runner.stopRun(taskID)
					return
				}
			}
		}
	}()
	return done
}

func (runner *SyncRunner) loop(ctx context.Context) {
	defer close(runner.done)
	for {
		select {
		case <-ctx.Done():
			return
		case taskID := <-runner.queue:
			if taskID != uuid.Nil {
				_ = runner.RunOnce(ctx, taskID)
			}
		}
	}
}

func (runner *SyncRunner) RunOnce(parent context.Context, taskID uuid.UUID) error {
	if runner == nil || runner.store == nil || runner.resolver == nil || runner.workspace == nil || runner.git == nil {
		return fmt.Errorf("sync runner is not configured")
	}
	if parent == nil {
		parent = context.Background()
	}
	deadline := defaultSyncDeadline
	ctx, cancel := context.WithTimeout(parent, deadline)
	defer cancel()
	run := &syncRun{cancel: cancel}
	if !runner.registerRun(taskID, run) {
		return nil
	}
	defer runner.unregisterRun(taskID, run)
	task, err := runner.store.GetSyncTask(ctx, taskID)
	if err != nil {
		if runner.runCancellationRequested(taskID) {
			return runner.cancelled(taskID, domain.Diagnostics{}, string(domain.CleanupClean))
		}
		return err
	}
	if task == nil {
		return fmt.Errorf("sync task %s was not returned by the store", taskID)
	}
	if task.State.Terminal() {
		return nil
	}
	if task.State == domain.SyncTaskCancelling {
		return runner.cancelled(taskID, task.Diagnostics, cancellationCleanupStatus(task))
	}
	started := time.Now().UTC()
	claimed, err := runner.store.ClaimTask(ctx, taskID, domain.SyncTaskValidatingSource, started, runner.leaseOwner, runner.leaseDuration)
	if err != nil {
		if runner.runCancellationRequested(taskID) {
			return runner.cancelled(taskID, task.Diagnostics, cancellationCleanupStatus(task))
		}
		return err
	}
	if !claimed {
		if runner.runCancellationRequested(taskID) {
			return runner.cancelled(taskID, task.Diagnostics, cancellationCleanupStatus(task))
		}
		return nil
	}
	cancellationWatchDone := runner.watchCancellation(ctx, taskID)
	leaseWatchDone := runner.watchLease(ctx, taskID)
	defer func() {
		cancel()
		<-cancellationWatchDone
		<-leaseWatchDone
	}()
	normalizedURL, err := runner.resolver.Validate(ctx, task.SourceType, task.RepoURL)
	if err != nil {
		return runner.fail(ctx, taskID, failureCodeWithContext(ctx, err), safeFailureSummaryFor(err), domain.Diagnostics{}, "")
	}
	workspacePath, err := runner.workspace.Create(ctx, taskID)
	if err != nil {
		return runner.fail(ctx, taskID, "WORKSPACE_UNAVAILABLE", "The isolated sync workspace could not be created.", domain.Diagnostics{}, "")
	}
	cleanup := func() string {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), time.Minute)
		defer cancelCleanup()
		if err := runner.workspace.Remove(cleanupCtx, workspacePath); err != nil {
			return string(domain.CleanupResidual)
		}
		return string(domain.CleanupClean)
	}
	workspaceKey := filepath.Base(workspacePath)
	if err := runner.store.SetTaskWorkspaceKey(ctx, taskID, workspaceKey); err != nil {
		return runner.fail(ctx, taskID, "SYNC_FAILED", "The sync task could not persist its workspace lease.", domain.Diagnostics{}, cleanup())
	}
	if err := runner.store.UpdateTaskProgress(ctx, taskID, domain.SyncTaskCloning, domain.SyncCounters{}); err != nil {
		return runner.fail(ctx, taskID, "SYNC_FAILED", "The sync task could not advance.", domain.Diagnostics{}, cleanup())
	}
	commitSHA, err := runner.git.Clone(ctx, normalizedURL, workspacePath)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return runner.fail(ctx, taskID, "DEADLINE_EXCEEDED", "The sync task exceeded its deadline.", domain.Diagnostics{}, cleanup())
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return runner.fail(ctx, taskID, "PROCESS_INTERRUPTED", "The sync task was interrupted and was not committed.", domain.Diagnostics{}, cleanup())
		}
		return runner.fail(ctx, taskID, failureCodeWithContext(ctx, err), safeFailureSummaryFor(err), domain.Diagnostics{}, cleanup())
	}
	if size, sizeErr := directorySize(ctx, workspacePath); sizeErr != nil {
		code := failureCodeWithContext(ctx, sizeErr)
		if code == "SYNC_FAILED" {
			code = failureCode(sizeErr)
		}
		return runner.fail(ctx, taskID, code, safeFailureSummaryFor(errors.New(code)), domain.Diagnostics{}, cleanup())
	} else if size > maxGitTransferBytes {
		return runner.fail(ctx, taskID, "GIT_QUOTA_EXCEEDED", "The repository exceeds the sync resource quota.", domain.Diagnostics{}, cleanup())
	} else if size > maxWorkspaceBytes {
		return runner.fail(ctx, taskID, "WORKSPACE_QUOTA_EXCEEDED", "The repository exceeds the sync resource quota.", domain.Diagnostics{}, cleanup())
	}
	filesSeen, yamlFilesSeen, bytesRead, candidates, diagnostics, err := scanAndValidateRepository(ctx, workspacePath, task)
	setCounters := domain.SyncCounters{FilesSeen: &filesSeen, YAMLFilesSeen: &yamlFilesSeen, BytesRead: &bytesRead}
	if progressErr := runner.store.UpdateTaskProgress(ctx, taskID, domain.SyncTaskScanningFiles, setCounters); progressErr != nil {
		return runner.fail(ctx, taskID, "SYNC_FAILED", "The sync task could not advance.", diagnostics, cleanup())
	}
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return runner.fail(ctx, taskID, "DEADLINE_EXCEEDED", "The sync task exceeded its deadline.", diagnostics, cleanup())
		}
		return runner.fail(ctx, taskID, failureCodeWithContext(ctx, err), safeFailureSummaryFor(err), diagnostics, cleanup())
	}
	validated := int64(len(candidates))
	setCounters.TemplatesValidated = &validated
	if err := runner.store.UpdateTaskProgress(ctx, taskID, domain.SyncTaskValidatingTemplates, setCounters); err != nil {
		return runner.fail(ctx, taskID, "SYNC_FAILED", "The sync task could not advance.", diagnostics, cleanup())
	}
	for _, candidate := range candidates {
		if err := runner.store.StageCandidate(ctx, candidate); err != nil {
			code := failureCode(err)
			if errors.Is(err, domain.ErrDuplicateTemplateID) {
				code = "DUPLICATE_TEMPLATE_ID"
			}
			if code == "SYNC_FAILED" {
				code = "CANDIDATE_STAGE_FAILED"
			}
			return runner.fail(ctx, taskID, code, safeFailureSummaryFor(err), diagnostics, cleanup())
		}
	}
	if len(candidates) == 0 {
		return runner.fail(ctx, taskID, "EMPTY_CANDIDATE", "The repository did not contain a valid Nuclei template.", diagnostics, cleanup())
	}
	if err := runner.store.UpdateTaskProgress(ctx, taskID, domain.SyncTaskCommitting, setCounters); err != nil {
		return runner.fail(ctx, taskID, "SYNC_FAILED", "The sync task could not advance.", diagnostics, cleanup())
	}
	source := domain.Source{ID: task.SourceID, SourceType: task.SourceType, RepoURL: normalizedURL}
	if err := runner.store.UpdateTaskProgress(ctx, taskID, domain.SyncTaskCleaning, setCounters); err != nil {
		return runner.fail(ctx, taskID, "SYNC_FAILED", "The sync task could not advance.", diagnostics, cleanup())
	}
	// Cleanup is intentionally completed before promotion so the transaction
	// can freeze the task's terminal result and cleanup warning together.
	cleanupStatus := cleanup()
	count, err := runner.store.PromoteCandidates(ctx, CandidatePromotion{TaskID: taskID, Source: source, CommitSHA: commitSHA, SyncedAt: time.Now().UTC(), CleanupStatus: cleanupStatus})
	if err != nil {
		return runner.fail(ctx, taskID, failureCodeWithContext(ctx, err), safeFailureSummaryFor(err), diagnostics, cleanupStatus)
	}
	_ = count
	return nil
}

func (runner *SyncRunner) fail(ctx context.Context, taskID uuid.UUID, code, summary string, diagnostics domain.Diagnostics, cleanupStatus string) error {
	if runner.runCancellationRequested(taskID) {
		return runner.cancelled(taskID, diagnostics, cleanupStatus)
	}
	if cleanupStatus == "" {
		cleanupStatus = string(domain.CleanupClean)
	}
	completed := time.Now().UTC()
	// The parent task context may already be cancelled by the two-hour
	// deadline. Terminal persistence must use a short independent context so a
	// deadline or Git interruption cannot strand the singleton active slot.
	terminalCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = runner.store.DeleteCandidatesForTask(terminalCtx, taskID)
	if err := runner.store.MarkTaskTerminal(terminalCtx, taskID, domain.SyncTaskFailed, code, summary, diagnostics, cleanupStatus, completed); err != nil {
		return err
	}
	// MarkTaskTerminal intentionally leaves a concurrent cancellation request
	// untouched. Re-read the durable state so that request still reaches its
	// cleanup-aware terminal transition instead of stranding the singleton.
	if runner.cancellationPersisted(taskID) {
		return runner.cancelled(taskID, diagnostics, cleanupStatus)
	}
	return fmt.Errorf("%w: %s", ErrSyncFailure, code)
}

func (runner *SyncRunner) cancelled(taskID uuid.UUID, diagnostics domain.Diagnostics, cleanupStatus string) error {
	if cleanupStatus == "" {
		cleanupStatus = string(domain.CleanupClean)
	}
	terminalCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = runner.store.DeleteCandidatesForTask(terminalCtx, taskID)
	if err := runner.store.MarkTaskCancelled(terminalCtx, taskID, diagnostics, cleanupStatus, time.Now().UTC()); err != nil {
		return err
	}
	return fmt.Errorf("%w: SYNC_CANCELLED", ErrSyncFailure)
}

func cancellationCleanupStatus(task *domain.SyncTask) string {
	if task != nil && strings.TrimSpace(task.WorkspaceKey) != "" {
		return string(domain.CleanupResidual)
	}
	return string(domain.CleanupClean)
}

func scanAndValidateRepository(ctx context.Context, root string, task *domain.SyncTask) (int64, int64, int64, []domain.CandidatePOC, domain.Diagnostics, error) {
	var filesSeen, yamlFilesSeen, bytesRead int64
	var workspaceBytes int64
	var candidates []domain.CandidatePOC
	diagnostics := domain.Diagnostics{}
	seenIDs := map[string]struct{}{}
	realRoot, realRootErr := filepath.EvalSymlinks(root)
	if realRootErr != nil {
		return 0, 0, 0, nil, diagnostics.Safe(), fmt.Errorf("PATH_ESCAPE")
	}
	realRoot = filepath.Clean(realRoot)
	if task == nil || task.ID == uuid.Nil || task.SourceID == uuid.Nil {
		return 0, 0, 0, nil, diagnostics.Safe(), domain.ErrInvalidPOC
	}
	addDiagnostic := func(path, reason string) {
		diagnostics.Total++
		if len(diagnostics.Samples) < domain.MaxDiagnosticSamples {
			diagnostics.Samples = append(diagnostics.Samples, domain.DiagnosticSample{Category: "template", RelativePath: path, ReasonCode: reason})
		} else {
			diagnostics.Truncated = true
		}
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(relative, "..") {
			return fmt.Errorf("PATH_ESCAPE")
		}
		if relative == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			addDiagnostic(filepath.ToSlash(relative), "SYMLINK_REJECTED")
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return fmt.Errorf("SYMLINK_REJECTED")
		}
		// WalkDir reports the directory entry without following it. Re-check
		// lstat/real-path containment for every entry to close replacement and
		// checkout path-escape races.
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			addDiagnostic(filepath.ToSlash(relative), "SYMLINK_REJECTED")
			return fmt.Errorf("SYMLINK_REJECTED")
		}
		realPath, realErr := filepath.EvalSymlinks(path)
		if realErr != nil || !isWithin(realRoot, filepath.Clean(realPath)) {
			addDiagnostic(filepath.ToSlash(relative), "PATH_ESCAPE")
			return fmt.Errorf("PATH_ESCAPE")
		}
		if entry.Name() == ".gitmodules" {
			addDiagnostic(filepath.ToSlash(relative), "SUBMODULE_REJECTED")
			return fmt.Errorf("SUBMODULE_REJECTED")
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			// A checked-out submodule is represented by a directory containing a
			// .git file. --no-recurse-submodules leaves that marker in place, so
			// reject it before any nested content is considered.
			gitMarker := filepath.Join(path, ".git")
			if marker, markerErr := os.Lstat(gitMarker); markerErr == nil && !marker.IsDir() {
				addDiagnostic(filepath.ToSlash(relative), "SUBMODULE_REJECTED")
				return fmt.Errorf("SUBMODULE_REJECTED")
			}
			return nil
		}
		filesSeen++
		if filesSeen > maxRepositoryFiles {
			return fmt.Errorf("FILE_QUOTA_EXCEEDED")
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		workspaceBytes += entryInfo.Size()
		if workspaceBytes > maxWorkspaceBytes {
			return fmt.Errorf("WORKSPACE_QUOTA_EXCEEDED")
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		yamlFilesSeen++
		if yamlFilesSeen > maxYAMLFiles {
			return fmt.Errorf("YAML_FILE_QUOTA_EXCEEDED")
		}
		if entryInfo.Size() < 0 || entryInfo.Size() > maxSingleYAMLBytes {
			return fmt.Errorf("YAML_SIZE_EXCEEDED")
		}
		raw, err := readBoundedYAMLFile(path, entryInfo.Size())
		if err != nil {
			return err
		}
		bytesRead += int64(len(raw))
		if bytesRead > maxYAMLBytes {
			return fmt.Errorf("YAML_QUOTA_EXCEEDED")
		}
		parsed, err := parseNucleiTemplate(raw)
		if err != nil {
			addDiagnostic(filepath.ToSlash(relative), strings.TrimPrefix(err.Error(), "YAML_PARSE_ERROR: "))
			// Repositories commonly contain YAML metadata alongside Nuclei
			// templates. A file that cannot be interpreted as one valid template
			// is isolated to that file; valid templates in the same repository
			// must still be eligible for atomic promotion.
			return nil
		}
		if _, exists := seenIDs[parsed.ID]; exists {
			// WalkDir is deterministic, so the first valid template for an ID
			// remains the import candidate and later duplicates are isolated.
			addDiagnostic(filepath.ToSlash(relative), "DUPLICATE_TEMPLATE_ID")
			return nil
		}
		seenIDs[parsed.ID] = struct{}{}
		digest := sha256.Sum256(raw)
		candidates = append(candidates, domain.CandidatePOC{ID: uuid.New(), TaskID: task.ID, SourceID: task.SourceID, TemplateID: parsed.ID, DisplayName: parsed.Name, Severity: parsed.Severity, Tags: parsed.Tags, Author: parsed.Author, Description: parsed.Description, CVE: parsed.CVE, CWE: parsed.CWE, References: parsed.References, Remediation: parsed.Remediation, RelativePath: filepath.ToSlash(relative), ContentSHA256: hex.EncodeToString(digest[:]), Content: string(raw), CreatedAt: time.Now().UTC()})
		return nil
	})
	if err != nil {
		return filesSeen, yamlFilesSeen, bytesRead, candidates, diagnostics.Safe(), err
	}
	return filesSeen, yamlFilesSeen, bytesRead, candidates, diagnostics.Safe(), nil
}

// readBoundedYAMLFile keeps the pre-read stat check meaningful if a checkout
// entry changes between WalkDir and ReadFile. It also rejects special files so
// a named pipe cannot block the runner indefinitely.
func readBoundedYAMLFile(path string, expectedSize int64) ([]byte, error) {
	if expectedSize < 0 || expectedSize > maxSingleYAMLBytes {
		return nil, fmt.Errorf("YAML_SIZE_EXCEEDED")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("PATH_ESCAPE")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("PATH_ESCAPE")
	}
	// The parser budget is intentionally stricter than the process-wide quota;
	// maxSingleYAMLBytes currently dominates it, but keep the invariant explicit
	// if either limit changes independently in a later change.
	readLimit := maxSingleYAMLBytes
	if maxParserMemoryBytes < readLimit {
		readLimit = maxParserMemoryBytes
	}
	raw, err := io.ReadAll(io.LimitReader(file, readLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > readLimit || int64(len(raw)) != opened.Size() {
		return nil, fmt.Errorf("YAML_SIZE_EXCEEDED")
	}
	return raw, nil
}

func failureCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "DEADLINE_EXCEEDED"
	}
	code := strings.TrimSpace(err.Error())
	if strings.Contains(code, "DEADLINE") {
		return "DEADLINE_EXCEEDED"
	}
	for _, allowed := range []string{"EMPTY_CANDIDATE", "DUPLICATE_TEMPLATE_ID", "TEMPLATE_INVALID", "FILE_QUOTA_EXCEEDED", "YAML_FILE_QUOTA_EXCEEDED", "YAML_QUOTA_EXCEEDED", "YAML_SIZE_EXCEEDED", "WORKSPACE_QUOTA_EXCEEDED", "PATH_ESCAPE", "SYMLINK_REJECTED", "SUBMODULE_REJECTED"} {
		if strings.Contains(code, allowed) {
			return allowed
		}
	}
	if strings.Contains(code, "GIT_QUOTA_EXCEEDED") {
		return "GIT_QUOTA_EXCEEDED"
	}
	if strings.Contains(code, "git clone failed") {
		return "GIT_CLONE_FAILED"
	}
	if strings.Contains(code, "git commit lookup failed") {
		return "GIT_COMMIT_LOOKUP_FAILED"
	}
	return "SYNC_FAILED"
}

func failureCodeWithContext(ctx context.Context, err error) string {
	if ctx != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "DEADLINE_EXCEEDED"
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return "PROCESS_INTERRUPTED"
		}
	}
	return failureCode(err)
}
func safeFailureSummaryFor(err error) string {
	switch failureCode(err) {
	case "FILE_QUOTA_EXCEEDED", "YAML_FILE_QUOTA_EXCEEDED", "YAML_QUOTA_EXCEEDED", "YAML_SIZE_EXCEEDED", "WORKSPACE_QUOTA_EXCEEDED":
		return "The repository exceeds the sync resource quota."
	case "SUBMODULE_REJECTED":
		return "The repository contains a Git submodule, which is not supported."
	case "TEMPLATE_INVALID":
		return "One or more Nuclei templates failed validation."
	case "DUPLICATE_TEMPLATE_ID":
		return "The repository contains duplicate template identities."
	case "EMPTY_CANDIDATE":
		return "The repository did not contain a valid Nuclei template."
	case "GIT_QUOTA_EXCEEDED":
		return "The repository exceeds the sync resource quota."
	case "GIT_CLONE_FAILED", "GIT_COMMIT_LOOKUP_FAILED":
		return "The repository could not be read."
	default:
		return "The Nuclei POC sync could not be completed."
	}
}

func directorySize(ctx context.Context, root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("SYMLINK_REJECTED")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		if total > maxGitTransferBytes {
			return fmt.Errorf("GIT_QUOTA_EXCEEDED")
		}
		return nil
	})
	return total, err
}
