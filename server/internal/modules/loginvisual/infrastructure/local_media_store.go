package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
	"golang.org/x/image/webp"
)

const (
	maxImageBytes    = 10 << 20
	maxVideoBytes    = 25 << 20
	maxVideoDuration = 20 * time.Second
)

type LocalMediaStore struct{ root string }

func NewLocalMediaStore(root string) (*LocalMediaStore, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("login visual storage root is required")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create login visual storage root: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect login visual storage root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("login visual storage root must be a real directory")
	}
	return &LocalMediaStore{root: root}, nil
}

func (store *LocalMediaStore) Save(ctx context.Context, contents []byte) (domain.Media, error) {
	if len(contents) == 0 || len(contents) > maxVideoBytes {
		return domain.Media{}, application.ErrInvalidMedia
	}
	kind, contentType, err := classifyMedia(contents)
	if err != nil {
		return domain.Media{}, err
	}
	if kind == domain.MediaKindImage && len(contents) > maxImageBytes {
		return domain.Media{}, application.ErrInvalidMedia
	}
	key := uuid.NewString()
	temporary, err := os.CreateTemp(store.root, ".login-visual-*")
	if err != nil {
		return domain.Media{}, fmt.Errorf("create login visual staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return domain.Media{}, fmt.Errorf("write login visual staging file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return domain.Media{}, fmt.Errorf("close login visual staging file: %w", err)
	}
	media := domain.Media{ID: uuid.NewString(), Kind: kind, ContentType: contentType, SizeBytes: int64(len(contents)), StorageKey: key, CreatedAt: time.Now().UTC()}
	if kind == domain.MediaKindVideo {
		duration, posterKey, err := store.validateVideo(ctx, temporaryPath, key)
		if err != nil {
			return domain.Media{}, err
		}
		media.Duration, media.PosterKey = duration, posterKey
	}
	if err := os.Rename(temporaryPath, store.filePath(key)); err != nil {
		return domain.Media{}, fmt.Errorf("promote login visual media: %w", err)
	}
	return media, nil
}

func classifyMedia(contents []byte) (domain.MediaKind, string, error) {
	contentType := http.DetectContentType(contents)
	switch contentType {
	case "image/jpeg", "image/png":
		if _, _, err := image.DecodeConfig(bytes.NewReader(contents)); err != nil {
			return "", "", application.ErrInvalidMedia
		}
		return domain.MediaKindImage, contentType, nil
	case "image/webp":
		if _, err := webp.DecodeConfig(bytes.NewReader(contents)); err != nil {
			return "", "", application.ErrInvalidMedia
		}
		return domain.MediaKindImage, contentType, nil
	case "video/mp4", "video/webm":
		return domain.MediaKindVideo, contentType, nil
	default:
		return "", "", application.ErrInvalidMedia
	}
}

func (store *LocalMediaStore) validateVideo(parent context.Context, staged, key string) (time.Duration, string, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", staged).Output()
	if err != nil || ctx.Err() != nil {
		return 0, "", application.ErrInvalidMedia
	}
	duration, err := time.ParseDuration(strings.TrimSpace(string(output)) + "s")
	if err != nil || duration <= 0 || duration > maxVideoDuration {
		return 0, "", application.ErrInvalidMedia
	}
	posterKey := key + ".png"
	posterPath := store.filePath(posterKey)
	if err := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-y", "-i", staged, "-frames:v", "1", posterPath).Run(); err != nil || ctx.Err() != nil {
		_ = os.Remove(posterPath)
		return 0, "", application.ErrInvalidMedia
	}
	return duration, posterKey, nil
}

func (store *LocalMediaStore) Open(_ context.Context, media domain.Media, poster bool) (io.ReadCloser, error) {
	key := media.StorageKey
	if poster {
		key = media.PosterKey
	}
	if key == "" {
		return nil, application.ErrMediaUnavailable
	}
	file, err := os.Open(store.filePath(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, application.ErrMediaUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("open login visual media: %w", err)
	}
	return file, nil
}

func (store *LocalMediaStore) Delete(_ context.Context, media domain.Media) error {
	for _, key := range []string{media.StorageKey, media.PosterKey} {
		if key == "" {
			continue
		}
		if err := os.Remove(store.filePath(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete login visual media: %w", err)
		}
	}
	return nil
}

func (store *LocalMediaStore) filePath(key string) string {
	if filepath.Base(key) != key || strings.Contains(key, string(filepath.Separator)) {
		return filepath.Join(store.root, ".invalid")
	}
	return filepath.Join(store.root, key)
}

var _ application.MediaStore = (*LocalMediaStore)(nil)
