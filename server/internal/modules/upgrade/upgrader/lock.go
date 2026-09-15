package upgrader

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

var ErrAlreadyRunning = errors.New("lunafox upgrader is already running")

type fileLock struct {
	file *os.File
}

func acquireFileLock(path string) (*fileLock, error) {
	if path == "" {
		return nil, fmt.Errorf("upgrader lock path is required")
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("upgrader lock must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect upgrader lock: %w", err)
	}
	// O_NOFOLLOW closes the check-then-open race for a hostile replacement of
	// the lock path. The subsequent Lstat remains useful for platforms/filesystem
	// combinations that ignore the flag.
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open upgrader lock: %w", err)
	}
	// Re-check the directory entry after opening. This catches a pre-existing
	// symlink and avoids treating an arbitrary target as the process lock.
	if info, statErr := os.Lstat(path); statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		_ = file.Close()
		if statErr != nil {
			return nil, fmt.Errorf("verify upgrader lock: %w", statErr)
		}
		return nil, fmt.Errorf("upgrader lock must be a regular file")
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("secure upgrader lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("acquire upgrader lock: %w", err)
	}
	return &fileLock{file: file}, nil
}

func (lock *fileLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	unlockErr := syscall.Flock(int(lock.file.Fd()), syscall.LOCK_UN)
	closeErr := lock.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
