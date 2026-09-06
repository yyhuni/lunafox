package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
)

func TestLocalMediaStorePersistsValidatedImageAndCleansItUp(t *testing.T) {
	t.Parallel()

	store, err := NewLocalMediaStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	media, err := store.Save(context.Background(), validPNG(t))
	if err != nil {
		t.Fatalf("save image: %v", err)
	}
	if media.Kind != domain.MediaKindImage || media.ContentType != "image/png" || media.StorageKey == "" {
		t.Fatalf("media = %#v", media)
	}

	reader, err := store.Open(context.Background(), media, false)
	if err != nil {
		t.Fatalf("open image: %v", err)
	}
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatalf("read image: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close image: %v", err)
	}
	if err := store.Delete(context.Background(), media); err != nil {
		t.Fatalf("delete image: %v", err)
	}
	if _, err := store.Open(context.Background(), media, false); !errors.Is(err, application.ErrMediaUnavailable) {
		t.Fatalf("open deleted image error = %v, want unavailable", err)
	}
}

func TestLocalMediaStoreRejectsInvalidAndOversizedImagesWithoutWritingFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := NewLocalMediaStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	for _, contents := range [][]byte{
		[]byte("not media"),
		malformedMP4(),
		append(validPNG(t), bytes.Repeat([]byte{0}, maxImageBytes)...),
		append(malformedMP4(), bytes.Repeat([]byte{0}, maxVideoBytes)...),
	} {
		if _, err := store.Save(context.Background(), contents); !errors.Is(err, application.ErrInvalidMedia) {
			t.Fatalf("save invalid media error = %v, want invalid media", err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read storage root: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("storage root contains %d files after rejected uploads", len(entries))
	}
}

func malformedMP4() []byte {
	return []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm', 0, 0, 0, 0, 'i', 's', 'o', 'm'}
}

func validPNG(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buffer.Bytes()
}
