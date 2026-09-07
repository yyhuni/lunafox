package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/fingerprint/domain"
	"github.com/yyhuni/lunafox/server/internal/pkg"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/zap"
)

var (
	fingerprintArtifactMeter       = otel.Meter("lunafox.fingerprint.artifacts")
	fingerprintArtifactResolutions = mustArtifactInt64Counter("fingerprint_artifact_resolutions_total")
	fingerprintArtifactBytes       = mustArtifactInt64Counter("fingerprint_artifact_bytes_total")
	fingerprintArtifactDuration    = mustArtifactFloat64Histogram("fingerprint_artifact_resolution_duration_seconds")
)

func mustArtifactInt64Counter(name string) metric.Int64Counter {
	counter, err := fingerprintArtifactMeter.Int64Counter(name)
	if err != nil {
		return metricnoop.Int64Counter{}
	}
	return counter
}

func mustArtifactFloat64Histogram(name string) metric.Float64Histogram {
	histogram, err := fingerprintArtifactMeter.Float64Histogram(name)
	if err != nil {
		return metricnoop.Float64Histogram{}
	}
	return histogram
}

// ArtifactDescriptor is the transport-safe identity of one immutable native
// artifact. Storage paths remain internal to this application boundary.
type ArtifactDescriptor struct {
	Library          domain.Library
	SourceGeneration int64
	SHA256Digest     string
	SizeBytes        int64
	RecordCount      int64
	Filename         string
	ContentType      string
	StorageKey       string
}

// ArtifactStore is deliberately narrower than the fingerprint command store:
// it exposes stable data and metadata, never arbitrary database access.
type ArtifactStore interface {
	CurrentGeneration(context.Context, domain.Library) (int64, error)
	ListAll(context.Context, domain.Library) ([]domain.PersistedRecord, error)
	FindArtifact(context.Context, domain.Library, int64) (*ArtifactDescriptor, error)
	ListArtifacts(context.Context, domain.Library) ([]ArtifactDescriptor, error)
	DeleteArtifact(context.Context, domain.Library, int64) error
	PublishArtifact(context.Context, ArtifactDescriptor) (ArtifactDescriptor, error)
	WithArtifactLibraryLock(context.Context, domain.Library, func() error) error
}

// ArtifactHandle keeps the backing file open so one HTTP or gRPC transfer
// cannot switch files if a newer current artifact is published concurrently.
type ArtifactHandle struct {
	Descriptor ArtifactDescriptor
	Reader     io.ReadCloser
}

type FingerprintArtifactService struct {
	store       ArtifactStore
	root        string
	locks       sync.Map
	readers     sync.Map
	writeNative func(io.Writer, domain.Library, []domain.PersistedRecord) (int64, error)
	syncFile    func(*os.File) error
	renameFile  func(string, string) error
}

func NewFingerprintArtifactService(store ArtifactStore, root string) (*FingerprintArtifactService, error) {
	if store == nil {
		return nil, errors.New("fingerprint artifact store is required")
	}
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || !filepath.IsAbs(root) {
		return nil, errors.New("fingerprint artifact root must be an absolute path")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create fingerprint artifact root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect fingerprint artifact root: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("fingerprint artifact root must be a regular directory")
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("restrict fingerprint artifact root: %w", err)
	}
	return &FingerprintArtifactService{
		store:       store,
		root:        root,
		writeNative: domain.WriteNativeExport,
		syncFile:    func(file *os.File) error { return file.Sync() },
		renameFile:  os.Rename,
	}, nil
}

func (service *FingerprintArtifactService) ResolveCurrent(ctx context.Context, library domain.Library) (ArtifactDescriptor, error) {
	if service == nil || !library.IsSupported() {
		return ArtifactDescriptor{}, errors.New("fingerprint artifact library is unsupported")
	}
	lock := service.libraryLock(library)
	lock.Lock()
	defer lock.Unlock()
	startedAt := time.Now()
	var resolved ArtifactDescriptor
	var outcome = "error"
	defer func() {
		attributes := metric.WithAttributes(attribute.String("fingerprint.library", string(library)), attribute.String("fingerprint.artifact.outcome", outcome))
		fingerprintArtifactResolutions.Add(ctx, 1, attributes)
		fingerprintArtifactDuration.Record(ctx, time.Since(startedAt).Seconds(), attributes)
	}()
	err := service.store.WithArtifactLibraryLock(ctx, library, func() error {
		var resolveErr error
		resolved, resolveErr = service.resolveCurrentLocked(ctx, library)
		return resolveErr
	})
	if err != nil {
		return ArtifactDescriptor{}, err
	}
	outcome = "resolved"
	fingerprintArtifactBytes.Add(ctx, resolved.SizeBytes, metric.WithAttributes(attribute.String("fingerprint.library", string(library))))
	return resolved, nil
}

func (service *FingerprintArtifactService) resolveCurrentLocked(ctx context.Context, library domain.Library) (ArtifactDescriptor, error) {
	for attempt := 0; attempt < 2; attempt++ {
		generation, err := service.store.CurrentGeneration(ctx, library)
		if err != nil {
			return ArtifactDescriptor{}, err
		}
		if artifact, err := service.store.FindArtifact(ctx, library, generation); err != nil {
			return ArtifactDescriptor{}, err
		} else if artifact != nil && service.artifactFileUsable(*artifact) {
			pkg.Info("fingerprint artifact resolved", zap.String("fingerprint.library", string(library)), zap.String("fingerprint.artifact.outcome", "reused"), zap.Int64("fingerprint.artifact.bytes", artifact.SizeBytes), zap.Int64("fingerprint.artifact.records", artifact.RecordCount))
			return *artifact, nil
		}

		records, err := service.store.ListAll(ctx, library)
		if err != nil {
			return ArtifactDescriptor{}, err
		}
		artifact, err := service.writeArtifact(ctx, library, generation, records)
		if err != nil {
			return ArtifactDescriptor{}, err
		}
		artifact, err = service.store.PublishArtifact(ctx, artifact)
		if err != nil {
			return ArtifactDescriptor{}, err
		}
		latest, err := service.store.CurrentGeneration(ctx, library)
		if err != nil {
			return ArtifactDescriptor{}, err
		}
		if latest == generation {
			pkg.Info("fingerprint artifact resolved", zap.String("fingerprint.library", string(library)), zap.String("fingerprint.artifact.outcome", "generated"), zap.Int64("fingerprint.artifact.bytes", artifact.SizeBytes), zap.Int64("fingerprint.artifact.records", artifact.RecordCount))
			return artifact, nil
		}
	}
	return ArtifactDescriptor{}, errors.New("fingerprint library changed repeatedly during artifact generation")
}

func (service *FingerprintArtifactService) OpenCurrent(ctx context.Context, library domain.Library) (ArtifactHandle, error) {
	artifact, err := service.ResolveCurrent(ctx, library)
	if err != nil {
		return ArtifactHandle{}, err
	}
	path, err := service.pathFor(artifact.StorageKey)
	if err != nil {
		return ArtifactHandle{}, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() != artifact.SizeBytes || !service.fileMatchesDigest(path, artifact.SHA256Digest) {
		return ArtifactHandle{}, errors.New("resolved fingerprint artifact file is unavailable")
	}
	file, err := os.Open(path)
	if err != nil {
		return ArtifactHandle{}, fmt.Errorf("open fingerprint artifact: %w", err)
	}
	service.acquireReader(artifact.StorageKey)
	return ArtifactHandle{Descriptor: artifact, Reader: &artifactReadCloser{ReadCloser: file, release: func() { service.releaseReader(artifact.StorageKey) }}}, nil
}

// ReconcileLibrary removes only superseded objects with no active HTTP/gRPC
// reader. Current state remains rebuildable from database payload at any time.
func (service *FingerprintArtifactService) ReconcileLibrary(ctx context.Context, library domain.Library) error {
	if service == nil || !library.IsSupported() {
		return errors.New("fingerprint artifact library is unsupported")
	}
	lock := service.libraryLock(library)
	lock.Lock()
	defer lock.Unlock()
	return service.store.WithArtifactLibraryLock(ctx, library, func() error {
		current, err := service.store.CurrentGeneration(ctx, library)
		if err != nil {
			return err
		}
		artifacts, err := service.store.ListArtifacts(ctx, library)
		if err != nil {
			return err
		}
		currentArtifact, err := service.store.FindArtifact(ctx, library, current)
		if err != nil {
			return err
		}
		for _, artifact := range artifacts {
			if artifact.SourceGeneration == current || service.activeReaders(artifact.StorageKey) != 0 {
				continue
			}
			path, pathErr := service.pathFor(artifact.StorageKey)
			if pathErr != nil {
				return pathErr
			}
			if currentArtifact == nil || currentArtifact.StorageKey != artifact.StorageKey {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove superseded fingerprint artifact: %w", err)
				}
			}
			if err := service.store.DeleteArtifact(ctx, library, artifact.SourceGeneration); err != nil {
				return err
			}
		}
		return nil
	})
}

type artifactReadCloser struct {
	io.ReadCloser
	once    sync.Once
	release func()
}

func (reader *artifactReadCloser) Close() error {
	err := reader.ReadCloser.Close()
	reader.once.Do(reader.release)
	return err
}

func (service *FingerprintArtifactService) acquireReader(storageKey string) {
	value, _ := service.readers.LoadOrStore(storageKey, &atomic.Int64{})
	value.(*atomic.Int64).Add(1)
}

func (service *FingerprintArtifactService) releaseReader(storageKey string) {
	value, ok := service.readers.Load(storageKey)
	if !ok {
		return
	}
	pointer := value.(*atomic.Int64)
	if pointer.Add(-1) == 0 {
		service.readers.Delete(storageKey)
	}
}

func (service *FingerprintArtifactService) activeReaders(storageKey string) int64 {
	value, ok := service.readers.Load(storageKey)
	if !ok {
		return 0
	}
	return value.(*atomic.Int64).Load()
}

func (service *FingerprintArtifactService) libraryLock(library domain.Library) *sync.Mutex {
	value, _ := service.locks.LoadOrStore(string(library), &sync.Mutex{})
	return value.(*sync.Mutex)
}

func (service *FingerprintArtifactService) writeArtifact(ctx context.Context, library domain.Library, generation int64, records []domain.PersistedRecord) (ArtifactDescriptor, error) {
	if err := ctx.Err(); err != nil {
		return ArtifactDescriptor{}, err
	}
	hash := sha256.New()
	countingHash := &artifactCountingWriter{writer: hash}
	if _, err := service.writeNative(countingHash, library, records); err != nil {
		return ArtifactDescriptor{}, err
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	storageKey := filepath.ToSlash(filepath.Join("sha256", strings.TrimPrefix(digest, "sha256:"), library.CanonicalFilename()))
	target, err := service.pathFor(storageKey)
	if err != nil {
		return ArtifactDescriptor{}, err
	}
	targetDirectory := filepath.Dir(target)
	if err := os.MkdirAll(targetDirectory, 0o700); err != nil {
		return ArtifactDescriptor{}, fmt.Errorf("create fingerprint artifact directory: %w", err)
	}
	if err := os.Chmod(targetDirectory, 0o700); err != nil {
		return ArtifactDescriptor{}, fmt.Errorf("restrict fingerprint artifact directory: %w", err)
	}
	targetExists := false
	if existing, statErr := os.Lstat(target); statErr == nil {
		if !existing.Mode().IsRegular() || existing.Mode()&os.ModeSymlink != 0 || existing.Size() != countingHash.n {
			if !existing.Mode().IsRegular() || existing.Mode()&os.ModeSymlink != 0 {
				return ArtifactDescriptor{}, errors.New("fingerprint artifact storage key conflicts with unsafe content")
			}
			if err := os.Remove(target); err != nil {
				return ArtifactDescriptor{}, fmt.Errorf("remove corrupt fingerprint artifact: %w", err)
			}
		} else if service.fileMatchesDigest(target, digest) {
			targetExists = true
		} else if err := os.Remove(target); err != nil {
			return ArtifactDescriptor{}, fmt.Errorf("remove corrupt fingerprint artifact: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return ArtifactDescriptor{}, fmt.Errorf("inspect fingerprint artifact target: %w", statErr)
	}
	if !targetExists {
		// The target directory is digest-derived. A second deterministic pass lets
		// us stage beside the final name, so rename remains same-directory atomic.
		temporary, err := os.CreateTemp(targetDirectory, ".fingerprint-artifact-*")
		if err != nil {
			return ArtifactDescriptor{}, fmt.Errorf("create fingerprint artifact temporary file: %w", err)
		}
		temporaryPath := temporary.Name()
		defer os.Remove(temporaryPath)
		writerHash := sha256.New()
		writer := io.MultiWriter(temporary, writerHash)
		if _, err := service.writeNative(writer, library, records); err != nil {
			_ = temporary.Close()
			return ArtifactDescriptor{}, err
		}
		if err := service.syncFile(temporary); err != nil {
			_ = temporary.Close()
			return ArtifactDescriptor{}, fmt.Errorf("sync fingerprint artifact: %w", err)
		}
		info, err := temporary.Stat()
		if closeErr := temporary.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return ArtifactDescriptor{}, fmt.Errorf("inspect fingerprint artifact: %w", err)
		}
		if info.Size() != countingHash.n || "sha256:"+hex.EncodeToString(writerHash.Sum(nil)) != digest {
			return ArtifactDescriptor{}, errors.New("fingerprint artifact deterministic encoding changed between passes")
		}
		if err := os.Chmod(temporaryPath, 0o400); err != nil {
			return ArtifactDescriptor{}, fmt.Errorf("restrict fingerprint artifact: %w", err)
		}
		if err := service.renameFile(temporaryPath, target); err != nil {
			return ArtifactDescriptor{}, fmt.Errorf("publish fingerprint artifact file: %w", err)
		}
	}

	return ArtifactDescriptor{Library: library, SourceGeneration: generation, SHA256Digest: digest, SizeBytes: countingHash.n, RecordCount: int64(len(records)), Filename: library.CanonicalFilename(), ContentType: library.NativeContentType(), StorageKey: storageKey}, nil
}

func (service *FingerprintArtifactService) artifactFileUsable(artifact ArtifactDescriptor) bool {
	path, err := service.pathFor(artifact.StorageKey)
	if err != nil {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 && info.Size() == artifact.SizeBytes && service.fileMatchesDigest(path, artifact.SHA256Digest)
}

type artifactCountingWriter struct {
	writer io.Writer
	n      int64
}

func (writer *artifactCountingWriter) Write(data []byte) (int, error) {
	n, err := writer.writer.Write(data)
	writer.n += int64(n)
	return n, err
}

func (service *FingerprintArtifactService) fileMatchesDigest(path, expected string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return expected == "sha256:"+hex.EncodeToString(hash.Sum(nil))
}

func (service *FingerprintArtifactService) pathFor(storageKey string) (string, error) {
	if storageKey == "" || filepath.IsAbs(storageKey) {
		return "", errors.New("fingerprint artifact storage key is invalid")
	}
	path := filepath.Join(service.root, filepath.FromSlash(storageKey))
	relative, err := filepath.Rel(service.root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("fingerprint artifact storage key escapes root")
	}
	return path, nil
}
