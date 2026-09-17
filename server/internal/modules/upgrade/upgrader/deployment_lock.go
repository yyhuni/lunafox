package upgrader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The deployment lock is the only mutual-exclusion primitive shared by the
// public Bash lifecycle scripts and the upgrader. Bash cannot portably read the
// named-volume upgrade journal, so the lock carries the little structured state
// both sides need: who owns the deployment right now, for which operation, and
// whether the owner is a recovery fence.
//
// It is a directory, so mkdir is the atomic primitive on every supported host
// without flock or a daemon. It lives on the deployment bind mount, never under
// the upgrade-state volume, because that volume is invisible to the host user
// who runs the lifecycle scripts.
const (
	DeploymentLockDirectory    = ".lunafox-lifecycle.lock"
	DeploymentLockMetadataFile = "owner"
	DeploymentLockSchema       = "1"

	deploymentLockOwnerUpgrade = "upgrade"
)

var (
	// ErrDeploymentLockHeld means another owner holds the deployment lock. The
	// caller must fail closed instead of mutating the deployment.
	ErrDeploymentLockHeld = errors.New("deployment lock is held by another owner")
	// ErrDeploymentLockInvalid means the lock exists but cannot be trusted. It is
	// never repaired automatically, because a crash residue must stay visible.
	ErrDeploymentLockInvalid = errors.New("deployment lock metadata is invalid")
)

// DeploymentLockMetadata is the host-readable lock content. It deliberately
// carries no credentials.
type DeploymentLockMetadata struct {
	Schema        string
	Owner         string
	Command       string
	OperationID   string
	CreatedAt     int64
	RecoveryFence bool
}

func deploymentLockPath(root string) string {
	return filepath.Join(root, DeploymentLockDirectory)
}

// ReadDeploymentLock returns the current lock metadata. The boolean reports
// whether a lock exists at all.
func ReadDeploymentLock(root string) (DeploymentLockMetadata, bool, error) {
	path := deploymentLockPath(root)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return DeploymentLockMetadata{}, false, nil
	}
	if err != nil {
		return DeploymentLockMetadata{}, false, fmt.Errorf("inspect deployment lock: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return DeploymentLockMetadata{}, true, fmt.Errorf("%w: %s is a symbolic link", ErrDeploymentLockInvalid, DeploymentLockDirectory)
	}
	if !info.IsDir() {
		return DeploymentLockMetadata{}, true, fmt.Errorf("%w: %s is not a directory", ErrDeploymentLockInvalid, DeploymentLockDirectory)
	}
	metadata, err := readDeploymentLockMetadata(path)
	if err != nil {
		return DeploymentLockMetadata{}, true, err
	}
	return metadata, true, nil
}

func readDeploymentLockMetadata(directory string) (DeploymentLockMetadata, error) {
	path := filepath.Join(directory, DeploymentLockMetadataFile)
	info, err := os.Lstat(path)
	if err != nil {
		return DeploymentLockMetadata{}, fmt.Errorf("%w: %s/%s is unreadable", ErrDeploymentLockInvalid, DeploymentLockDirectory, DeploymentLockMetadataFile)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return DeploymentLockMetadata{}, fmt.Errorf("%w: %s/%s is a symbolic link", ErrDeploymentLockInvalid, DeploymentLockDirectory, DeploymentLockMetadataFile)
	}
	if !info.Mode().IsRegular() {
		return DeploymentLockMetadata{}, fmt.Errorf("%w: %s/%s is not a regular file", ErrDeploymentLockInvalid, DeploymentLockDirectory, DeploymentLockMetadataFile)
	}
	if permission := info.Mode().Perm(); permission != 0o644 && permission != 0o600 {
		return DeploymentLockMetadata{}, fmt.Errorf("%w: %s/%s has mode %04o", ErrDeploymentLockInvalid, DeploymentLockDirectory, DeploymentLockMetadataFile, permission)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return DeploymentLockMetadata{}, fmt.Errorf("%w: read %s/%s", ErrDeploymentLockInvalid, DeploymentLockDirectory, DeploymentLockMetadataFile)
	}
	metadata := DeploymentLockMetadata{}
	for _, line := range strings.Split(string(payload), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			return DeploymentLockMetadata{}, fmt.Errorf("%w: malformed %s line", ErrDeploymentLockInvalid, DeploymentLockMetadataFile)
		}
		switch key {
		case "schema":
			metadata.Schema = value
		case "owner":
			metadata.Owner = value
		case "command":
			metadata.Command = value
		case "operation_id":
			metadata.OperationID = value
		case "created_at":
			seconds, parseErr := strconv.ParseInt(value, 10, 64)
			if parseErr != nil {
				return DeploymentLockMetadata{}, fmt.Errorf("%w: invalid created_at", ErrDeploymentLockInvalid)
			}
			metadata.CreatedAt = seconds
		case "recovery_fence":
			metadata.RecoveryFence = value == "true"
		}
	}
	if metadata.Schema != DeploymentLockSchema || metadata.Owner == "" {
		return DeploymentLockMetadata{}, fmt.Errorf("%w: unexpected schema or owner", ErrDeploymentLockInvalid)
	}
	return metadata, nil
}

// DeploymentLock is an acquired deployment lock. Release clears it; any other
// process can only take over by matching the recorded operation identity.
type DeploymentLock struct {
	directory   string
	operationID string
}

// OperationID reports which Upgrade Operation owns the lock.
func (lock *DeploymentLock) OperationID() string {
	if lock == nil {
		return ""
	}
	return lock.operationID
}

// AcquireDeploymentLock takes the deployment lock for one Upgrade Operation.
// An existing lock is only taken over when it belongs to the same operation,
// which keeps an interrupted operation resumable without reopening the window
// for a concurrent lifecycle mutation.
func AcquireDeploymentLock(root, operationID string) (*DeploymentLock, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return nil, fmt.Errorf("upgrade operation ID is required to acquire the deployment lock")
	}
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("deployment root is required to acquire the deployment lock")
	}
	directory := deploymentLockPath(root)
	if err := os.Mkdir(directory, 0o755); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create deployment lock: %w", err)
		}
		metadata, _, readErr := ReadDeploymentLock(root)
		if readErr != nil {
			return nil, readErr
		}
		if metadata.Owner != deploymentLockOwnerUpgrade || metadata.OperationID != operationID {
			return nil, fmt.Errorf("%w: owner=%s command=%s operation=%s", ErrDeploymentLockHeld, metadata.Owner, metadata.Command, metadata.OperationID)
		}
		// The same operation already owns the lock; keep the existing metadata so
		// a recovery fence recorded earlier is not silently downgraded.
		return &DeploymentLock{directory: directory, operationID: operationID}, nil
	}
	lock := &DeploymentLock{directory: directory, operationID: operationID}
	if err := lock.writeMetadata(DeploymentLockMetadata{
		Schema:      DeploymentLockSchema,
		Owner:       deploymentLockOwnerUpgrade,
		OperationID: operationID,
		CreatedAt:   time.Now().UTC().Unix(),
	}); err != nil {
		_ = os.Remove(directory)
		return nil, err
	}
	return lock, nil
}

// AcquireDeploymentLockContext retries until the lock is free or ctx is done.
// Waiting is the correct response to a lifecycle command holding the lock: the
// upgrader must neither fail its own startup nor mutate concurrently.
func AcquireDeploymentLockContext(ctx context.Context, root, operationID string, interval time.Duration) (*DeploymentLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		lock, err := AcquireDeploymentLock(root, operationID)
		if err == nil {
			return lock, nil
		}
		if !errors.Is(err, ErrDeploymentLockHeld) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for the deployment lock: %w", ctx.Err())
		case <-time.After(interval):
		}
	}
}

// MarkRecoveryFence records that the owning Operation needs recovery. The lock
// stays in place so a lifecycle command cannot mutate a deployment whose
// database outcome is unknown.
func (lock *DeploymentLock) MarkRecoveryFence() error {
	if lock == nil {
		return nil
	}
	return lock.writeMetadata(DeploymentLockMetadata{
		Schema:        DeploymentLockSchema,
		Owner:         deploymentLockOwnerUpgrade,
		OperationID:   lock.operationID,
		CreatedAt:     time.Now().UTC().Unix(),
		RecoveryFence: true,
	})
}

// Release clears the lock. Only the owning operation calls it, and only after
// the operation reached a terminal stage that needs no recovery.
func (lock *DeploymentLock) Release() error {
	if lock == nil {
		return nil
	}
	var firstErr error
	if err := os.Remove(filepath.Join(lock.directory, DeploymentLockMetadataFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		firstErr = err
	}
	if err := os.Remove(lock.directory); err != nil && !errors.Is(err, os.ErrNotExist) && firstErr == nil {
		firstErr = err
	}
	if firstErr != nil {
		return fmt.Errorf("release deployment lock: %w", firstErr)
	}
	return nil
}

func (lock *DeploymentLock) writeMetadata(metadata DeploymentLockMetadata) error {
	payload := strings.Builder{}
	payload.WriteString("schema=" + metadata.Schema + "\n")
	payload.WriteString("owner=" + metadata.Owner + "\n")
	payload.WriteString("command=" + metadata.Command + "\n")
	payload.WriteString("operation_id=" + metadata.OperationID + "\n")
	payload.WriteString("created_at=" + strconv.FormatInt(metadata.CreatedAt, 10) + "\n")
	payload.WriteString("recovery_fence=" + strconv.FormatBool(metadata.RecoveryFence) + "\n")

	temporary, err := os.CreateTemp(lock.directory, ".owner.tmp-")
	if err != nil {
		return fmt.Errorf("stage deployment lock metadata: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("restrict deployment lock metadata: %w", err)
	}
	if _, err := temporary.WriteString(payload.String()); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write deployment lock metadata: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync deployment lock metadata: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close deployment lock metadata: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(lock.directory, DeploymentLockMetadataFile)); err != nil {
		return fmt.Errorf("publish deployment lock metadata: %w", err)
	}
	return nil
}

// requiresRecoveryFence reports whether a terminal journal still needs the lock
// as a recovery fence.
func requiresRecoveryFence(stage Stage) bool {
	return stage == StageNeedsRecovery || stage == StageNeedsAttention
}
