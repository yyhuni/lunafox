package screenshotruntime

import (
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

type cwebpTestRunner struct {
	outputs [][]string
	data    [][]byte
}

func (runner *cwebpTestRunner) Run(_ context.Context, args []string) error {
	runner.outputs = append(runner.outputs, append([]string(nil), args...))
	index := len(runner.outputs) - 1
	return os.WriteFile(args[len(args)-1], runner.data[index], 0o600)
}

func testWebP(width, height uint16) []byte {
	payload := []byte{0, 0, 0, 0x9d, 0x01, 0x2a, byte(width), byte(width >> 8), byte(height), byte(height >> 8)}
	return testWebPPayload(payload)
}

func testWebPPayload(payload []byte) []byte {
	data := make([]byte, 20+len(payload))
	copy(data, []byte("RIFF"))
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	copy(data[8:], []byte("WEBPVP8 "))
	binary.LittleEndian.PutUint32(data[16:20], uint32(len(payload)))
	copy(data[20:], payload)
	return data
}

func writeTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 80, A: 255})
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}

func TestProcessScreenshotPNGUsesOneFallbackOnly(t *testing.T) {
	workspace := t.TempDir()
	pngPath := filepath.Join(workspace, "capture.png")
	writeTestPNG(t, pngPath, 1200, 800)
	overBudgetPayload := append([]byte{0, 0, 0, 0x9d, 0x01, 0x2a, 0x20, 0x03, 0x20, 0x03}, make([]byte, 256*1024)...)
	runner := &cwebpTestRunner{data: [][]byte{testWebPPayload(overBudgetPayload), testWebP(640, 427)}}
	encoded, outcome, err := processScreenshotPNG(context.Background(), workspace, pngPath, runner, filepath.Join(workspace, "out"))
	if err != nil || outcome != outcomeSubmitted || len(encoded) == 0 {
		t.Fatalf("processScreenshotPNG() = %d/%v/%v", len(encoded), outcome, err)
	}
	if len(runner.outputs) != 2 || runner.outputs[0][1] != "-resize" || runner.outputs[0][2] != "800" || runner.outputs[1][2] != "640" || runner.outputs[1][5] != "65" {
		t.Fatalf("cwebp calls = %#v", runner.outputs)
	}
}

func TestProcessScreenshotPNGUsesByteBudgetForMalformedWebPWithoutStructureValidation(t *testing.T) {
	workspace := t.TempDir()
	pngPath := filepath.Join(workspace, "capture.png")
	writeTestPNG(t, pngPath, 1200, 800)
	malformed := make([]byte, maxCapturedWebPBytes+1024)
	copy(malformed, []byte("RIFF"))
	binary.LittleEndian.PutUint32(malformed[4:8], uint32(len(malformed)-8))
	copy(malformed[8:12], []byte("WEBP"))
	runner := &cwebpTestRunner{data: [][]byte{malformed, testWebP(640, 427)}}
	_, outcome, err := processScreenshotPNG(context.Background(), workspace, pngPath, runner, filepath.Join(workspace, "out"))
	if err != nil || outcome != outcomeSubmitted {
		t.Fatalf("processScreenshotPNG() = %v/%v, want byte-budget fallback submission", outcome, err)
	}
	if len(runner.outputs) != 2 || runner.outputs[1][2] != "640" {
		t.Fatalf("malformed oversized WebP did not use the bounded fallback: %#v", runner.outputs)
	}
}

func TestValidatePNGPathDistinguishesMissingAndWorkspaceEscape(t *testing.T) {
	workspace := t.TempDir()
	if _, err := validatePNGPath(workspace, filepath.Join(workspace, "missing.png")); !errors.Is(err, errScreenshotMissing) {
		t.Fatalf("missing PNG error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside.png")
	if _, err := validatePNGPath(workspace, outside); !errors.Is(err, errScreenshotPathViolation) {
		t.Fatalf("outside PNG error = %v", err)
	}
	outsideDir := t.TempDir()
	link := filepath.Join(workspace, "escape")
	if err := os.Symlink(outsideDir, link); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}
	if err := validatePNGReclaimPath(workspace, filepath.Join(link, "missing.png")); !errors.Is(err, errScreenshotPathViolation) {
		t.Fatalf("missing PNG below symlink was accepted: %v", err)
	}
}

func TestReclaimPNGIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.png")
	if err := os.WriteFile(path, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := reclaimPNG(path); err != nil {
		t.Fatal(err)
	}
	if err := reclaimPNG(path); err != nil {
		t.Fatal(err)
	}
}

func TestReclaimPNGPropagatesRemoveFailure(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "child"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := reclaimPNG(directory); err == nil {
		t.Fatal("reclaimPNG accepted a remove failure")
	}
}
