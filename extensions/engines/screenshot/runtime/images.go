package screenshotruntime

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	maxPNGBytes  = 8 * 1024 * 1024
	maxPNGPixels = 8_000_000
	// maxCapturedWebPBytes is a capture-stage budget used to choose the one
	// lower-size cwebp retry. It is not Server result acceptance validation.
	maxCapturedWebPBytes = 256 * 1024
)

type screenshotAttemptOutcome string

const (
	outcomeSubmitted   screenshotAttemptOutcome = "submitted"
	outcomeNavigation  screenshotAttemptOutcome = "navigation"
	outcomeScreenshot  screenshotAttemptOutcome = "screenshot"
	outcomeInvalidPNG  screenshotAttemptOutcome = "invalid_png"
	outcomeConversion  screenshotAttemptOutcome = "conversion"
	outcomeImageBudget screenshotAttemptOutcome = "image_budget"
)

func processScreenshotPNG(ctx context.Context, workspace, pngPath string, runner cwebpRunner, outputBase string) ([]byte, screenshotAttemptOutcome, error) {
	if runner == nil {
		return nil, outcomeConversion, errors.New("cwebp runner is required")
	}
	if ctx == nil {
		return nil, outcomeConversion, errors.New("cwebp context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, outcomeConversion, err
	}
	width, err := validatePNGPath(workspace, pngPath)
	if err != nil {
		if errors.Is(err, errInvalidPNG) {
			return nil, outcomeInvalidPNG, nil
		}
		if errors.Is(err, errScreenshotMissing) {
			return nil, outcomeScreenshot, nil
		}
		return nil, outcomeScreenshot, err
	}
	if err := ctx.Err(); err != nil {
		return nil, outcomeConversion, err
	}
	firstWidth := width
	if firstWidth > 800 {
		firstWidth = 800
	}
	firstOutput := outputBase + "-800.webp"
	if err := runner.Run(ctx, buildCWebPArgs(pngPath, firstOutput, firstWidth, 72)); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, outcomeConversion, ctxErr
		}
		return nil, outcomeConversion, nil
	}
	encoded, err := readValidWebP(firstOutput)
	if err != nil && !errors.Is(err, errWebPOverBudget) {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, outcomeConversion, ctxErr
		}
		return nil, outcomeConversion, nil
	}
	if err == nil && len(encoded) <= maxCapturedWebPBytes {
		return encoded, outcomeSubmitted, nil
	}
	secondWidth := width
	if secondWidth > 640 {
		secondWidth = 640
	}
	secondOutput := outputBase + "-640.webp"
	if err := runner.Run(ctx, buildCWebPArgs(pngPath, secondOutput, secondWidth, 65)); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, outcomeConversion, ctxErr
		}
		return nil, outcomeConversion, nil
	}
	encoded, err = readValidWebP(secondOutput)
	if errors.Is(err, errWebPOverBudget) {
		return nil, outcomeImageBudget, nil
	}
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, outcomeConversion, ctxErr
		}
		return nil, outcomeConversion, nil
	}
	if len(encoded) > maxCapturedWebPBytes {
		return nil, outcomeImageBudget, nil
	}
	return encoded, outcomeSubmitted, nil
}

func buildCWebPArgs(input, output string, width int, quality int) []string {
	return []string{"-quiet", "-resize", strconv.Itoa(width), "0", "-q", strconv.Itoa(quality), input, "-o", output}
}

var errInvalidPNG = errors.New("invalid PNG")
var errScreenshotMissing = errors.New("screenshot PNG is missing")
var errScreenshotPathViolation = errors.New("screenshot PNG path violates workspace boundary")
var errWebPOverBudget = errors.New("screenshot WebP exceeds byte budget")

// validatePNGReclaimPath checks only the workspace boundary needed before
// reclaiming a row that HTTPX marked failed. A failed row is navigation-local
// and need not contain a valid PNG, but it must never make cleanup touch an
// arbitrary path.
func validatePNGReclaimPath(workspace, pngPath string) error {
	if workspace == "" || pngPath == "" {
		return errScreenshotPathViolation
	}
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return errScreenshotPathViolation
	}
	pathAbs, err := filepath.Abs(pngPath)
	if err != nil {
		return errScreenshotPathViolation
	}
	rel, err := filepath.Rel(workspaceAbs, pathAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errScreenshotPathViolation
	}
	workspaceReal, err := filepath.EvalSymlinks(workspaceAbs)
	if err != nil {
		return errScreenshotPathViolation
	}
	pathInfo, err := os.Lstat(pathAbs)
	if errors.Is(err, os.ErrNotExist) {
		// A failed HTTPX row may point to a path that no longer exists. Check
		// the real parent before treating that as an idempotent no-op so a
		// symlinked directory cannot turn a later remove into an outside-
		// workspace deletion.
		parentReal, parentErr := filepath.EvalSymlinks(filepath.Dir(pathAbs))
		if parentErr != nil {
			return errScreenshotPathViolation
		}
		parentReal, parentErr = filepath.Abs(parentReal)
		if parentErr != nil {
			return errScreenshotPathViolation
		}
		parentRel, parentErr := filepath.Rel(workspaceReal, parentReal)
		if parentErr != nil || parentRel == ".." || strings.HasPrefix(parentRel, ".."+string(filepath.Separator)) {
			return errScreenshotPathViolation
		}
		return nil
	}
	if err != nil {
		return err
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.Mode().IsRegular() {
		return errScreenshotPathViolation
	}
	pathReal, err := filepath.EvalSymlinks(pathAbs)
	if err != nil {
		return errScreenshotPathViolation
	}
	pathReal, err = filepath.Abs(pathReal)
	if err != nil {
		return errScreenshotPathViolation
	}
	realRel, err := filepath.Rel(workspaceReal, pathReal)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return errScreenshotPathViolation
	}
	return nil
}

func validatePNGPath(workspace, pngPath string) (int, error) {
	if workspace == "" || pngPath == "" {
		return 0, errScreenshotPathViolation
	}
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return 0, errScreenshotPathViolation
	}
	pathAbs, err := filepath.Abs(pngPath)
	if err != nil {
		return 0, errScreenshotPathViolation
	}
	rel, err := filepath.Rel(workspaceAbs, pathAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return 0, errScreenshotPathViolation
	}
	workspaceReal, err := filepath.EvalSymlinks(workspaceAbs)
	if err != nil {
		return 0, errScreenshotPathViolation
	}
	pathInfo, err := os.Lstat(pathAbs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, errScreenshotMissing
		}
		return 0, err
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.Mode().IsRegular() {
		return 0, errScreenshotPathViolation
	}
	pathReal, err := filepath.EvalSymlinks(pathAbs)
	if err != nil {
		return 0, errScreenshotPathViolation
	}
	pathReal, err = filepath.Abs(pathReal)
	if err != nil {
		return 0, err
	}
	realRel, err := filepath.Rel(workspaceReal, pathReal)
	if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return 0, errScreenshotPathViolation
	}
	if pathInfo.Size() == 0 {
		return 0, errScreenshotMissing
	}
	if pathInfo.Size() > maxPNGBytes {
		return 0, errInvalidPNG
	}
	file, err := os.Open(pathAbs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, errScreenshotMissing
		}
		return 0, err
	}
	defer file.Close()
	config, format, err := imageConfig(file)
	if err != nil || format != "png" || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxPNGPixels {
		return 0, errInvalidPNG
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	if _, err := png.Decode(file); err != nil {
		return 0, errInvalidPNG
	}
	return config.Width, nil
}

func imageConfig(reader io.Reader) (config struct{ Width, Height int }, format string, err error) {
	decoded, err := png.DecodeConfig(reader)
	if err != nil {
		return config, "", err
	}
	return struct{ Width, Height int }{Width: decoded.Width, Height: decoded.Height}, "png", nil
}

func readValidWebP(path string) ([]byte, error) {
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 || !linkInfo.Mode().IsRegular() {
		return nil, errors.New("screenshot WebP output is not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > int64(maxCapturedWebPBytes) {
		return nil, errWebPOverBudget
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maxCapturedWebPBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxCapturedWebPBytes {
		return nil, errWebPOverBudget
	}
	return data, nil
}

func reclaimPNG(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove consumed screenshot PNG: %w", err)
	}
	return nil
}
