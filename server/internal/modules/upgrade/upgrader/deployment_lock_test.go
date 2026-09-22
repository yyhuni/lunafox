package upgrader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcquireDeploymentLockPublishesReadableMetadata(t *testing.T) {
	root := t.TempDir()
	lock, err := AcquireDeploymentLock(root, "operation-1")
	if err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(root, DeploymentLockDirectory, DeploymentLockMetadataFile)
	payload, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	// The public lifecycle scripts parse this format without jq, so the exact
	// keys are part of the cross-implementation contract.
	for _, expected := range []string{"schema=1\n", "owner=upgrade\n", "operation_id=operation-1\n", "recovery_fence=false\n"} {
		if !strings.Contains(string(payload), expected) {
			t.Fatalf("metadata %q is missing %q", payload, expected)
		}
	}
	info, err := os.Stat(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	if permission := info.Mode().Perm(); permission != 0o644 {
		t.Fatalf("metadata mode=%04o", permission)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(root, DeploymentLockDirectory)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock directory survived release: %v", err)
	}
}

func TestAcquireDeploymentLockEnforcesSingleOwner(t *testing.T) {
	root := t.TempDir()
	first, err := AcquireDeploymentLock(root, "operation-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDeploymentLock(root, "operation-2"); !errors.Is(err, ErrDeploymentLockHeld) {
		t.Fatalf("second operation acquired the lock: %v", err)
	}
	taken, err := AcquireDeploymentLock(root, "operation-1")
	if err != nil {
		t.Fatalf("the owning operation could not resume: %v", err)
	}
	metadata, exists, err := ReadDeploymentLock(root)
	if err != nil || !exists || metadata.OperationID != "operation-1" {
		t.Fatalf("metadata=%#v exists=%v err=%v", metadata, exists, err)
	}
	if err := taken.Release(); err != nil {
		t.Fatal(err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestStaleUpgradeLockReleaseDoesNotRemoveLifecycleOwner(t *testing.T) {
	root := t.TempDir()
	lock, err := AcquireDeploymentLock(root, "operation-1")
	if err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(root, DeploymentLockDirectory, DeploymentLockMetadataFile)
	if err := os.WriteFile(metadataPath, []byte("schema=1\nowner=lifecycle\ncommand=install\noperation_id=\ncreated_at=1700000000\nrecovery_fence=false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	metadata, exists, err := ReadDeploymentLock(root)
	if err != nil || !exists || metadata.Owner != "lifecycle" {
		t.Fatalf("stale release removed lifecycle lock: metadata=%#v exists=%v err=%v", metadata, exists, err)
	}
}

func TestDeploymentLockRejectsLifecycleOwnerAndInvalidMetadata(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, DeploymentLockDirectory)
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(directory, DeploymentLockMetadataFile)
	// A lock written by the Bash lifecycle helper must be honoured, not stolen.
	if err := os.WriteFile(metadataPath, []byte("schema=1\nowner=lifecycle\ncommand=install\noperation_id=\ncreated_at=1700000000\nrecovery_fence=false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDeploymentLock(root, "operation-9"); !errors.Is(err, ErrDeploymentLockHeld) {
		t.Fatalf("lifecycle lock was taken over: %v", err)
	}
	if err := os.WriteFile(metadataPath, []byte("schema=1\nowner=upgrade\noperation_id=operation-9\ncreated_at=1700000000\nrecovery_fence=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDeploymentLock(root, "operation-9"); err != nil {
		t.Fatalf("mode-0600 metadata was rejected: %v", err)
	}
	if err := os.Chmod(metadataPath, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireDeploymentLock(root, "operation-9"); !errors.Is(err, ErrDeploymentLockInvalid) {
		t.Fatalf("permissive metadata was accepted: %v", err)
	}
}

func TestDeploymentLockRejectsSymlinkedPaths(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(root, DeploymentLockDirectory)); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if _, err := AcquireDeploymentLock(root, "operation-1"); !errors.Is(err, ErrDeploymentLockInvalid) {
		t.Fatalf("symlinked lock directory was accepted: %v", err)
	}
}

func TestMarkRecoveryFenceKeepsTheLockAfterFailure(t *testing.T) {
	root := t.TempDir()
	lock, err := AcquireDeploymentLock(root, "operation-3")
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.MarkRecoveryFence(); err != nil {
		t.Fatal(err)
	}
	metadata, exists, err := ReadDeploymentLock(root)
	if err != nil || !exists {
		t.Fatalf("metadata=%#v exists=%v err=%v", metadata, exists, err)
	}
	if !metadata.RecoveryFence || metadata.OperationID != "operation-3" {
		t.Fatalf("metadata=%#v", metadata)
	}
	if requiresRecoveryFence(StageSucceeded) || requiresRecoveryFence(StageFailed) {
		t.Fatal("a terminal non-recovery stage must not keep the fence")
	}
	for _, stage := range []Stage{StageNeedsRecovery, StageNeedsAttention, StageUpdating, StageMigrating} {
		if stage == StageNeedsRecovery || stage == StageNeedsAttention {
			if !requiresRecoveryFence(stage) {
				t.Fatalf("stage %s must keep the fence", stage)
			}
			continue
		}
		if requiresRecoveryFence(stage) {
			t.Fatalf("non-terminal stage %s reported as a terminal fence", stage)
		}
	}
}

func TestAcquireDeploymentLockContextWaitsForRelease(t *testing.T) {
	root := t.TempDir()
	held, err := AcquireDeploymentLock(root, "operation-1")
	if err != nil {
		t.Fatal(err)
	}
	released := make(chan error, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		released <- held.Release()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock, err := AcquireDeploymentLockContext(ctx, root, "operation-2", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-released; err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}
