package engineinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

var installLocks sync.Map

type CacheInstaller struct {
	Root            string
	MaxArchiveBytes int64
}

func promotePath(from, to string) error {
	if _, err := os.Stat(to); err == nil {
		return fmt.Errorf("refusing to promote over existing cache path %q", to)
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(from, to)
}

func lockForDigest(digest string) *sync.Mutex {
	lock, _ := installLocks.LoadOrStore(digest, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func acquireCacheProcessLock(root, cacheKey string) (func(), error) {
	locksRoot := filepath.Join(root, ".locks")
	if err := os.MkdirAll(locksRoot, 0o750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(locksRoot, cacheKey+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
