package directoryscanruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// directoryWordlistLineLimit bounds one physical wordlist line so the
	// normalized copy never needs unbounded memory. It matches the JSONL
	// parser's per-record bound.
	directoryWordlistLineLimit  = 4 * 1024 * 1024
	directoryNormalizedListName = "directory-wordlist.txt"
	directoryWordlistSubdir     = "directory-wordlist"
)

// normalizeDirectoryTarget appends exactly one path separator so that the FFUF
// target `normalized + "FUZZ"` stays on the target host with a real path.
// Scheme, authority, query, and fragment bytes are preserved; only the path
// component gains a trailing slash when it is empty or lacks one.
func normalizeDirectoryTarget(candidate string) (string, error) {
	if candidate == "" {
		return "", errors.New("FFUF candidate is required")
	}
	schemeEnd := strings.Index(candidate, "://")
	if schemeEnd < 0 {
		return "", errors.New("FFUF candidate must be an absolute URL")
	}
	authorityStart := schemeEnd + len("://")
	rest := candidate[authorityStart:]
	authorityEnd := len(rest)
	for index := 0; index < len(rest); index++ {
		switch rest[index] {
		case '/', '?', '#':
			authorityEnd = index
		}
		if authorityEnd != len(rest) {
			break
		}
	}
	if authorityEnd == 0 {
		return "", errors.New("FFUF candidate authority is required")
	}
	prefix := candidate[:authorityStart+authorityEnd]
	remainder := rest[authorityEnd:]
	mark := len(remainder)
	for index := 0; index < len(remainder); index++ {
		if remainder[index] == '?' || remainder[index] == '#' {
			mark = index
			break
		}
	}
	pathPart := remainder[:mark]
	suffix := remainder[mark:]
	if !strings.HasSuffix(pathPart, "/") {
		pathPart += "/"
	}
	return prefix + pathPart + suffix, nil
}

// normalizedWordlist is an Engine-private copy of the selected wordlist with
// at most one leading separator removed from every payload.
type normalizedWordlist struct {
	path string
}

func (wordlist *normalizedWordlist) Cleanup() error {
	if wordlist == nil || wordlist.path == "" {
		return nil
	}
	if err := os.Remove(wordlist.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func validateWordlistSource(sourcePath string) error {
	if sourcePath == "" {
		return errors.New("wordlist source path is required")
	}
	if !filepath.IsAbs(sourcePath) {
		return errors.New("wordlist source path must be absolute")
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect wordlist source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("wordlist source must be a regular file")
	}
	return nil
}

// prepareCandidateWordlist builds one per-candidate wordlist at the deterministic
// workspace path directory-wordlist/<ordinal>.txt. The file is the normalized
// source wordlist plus deterministic target-derived variants (deduplicated
// against the source) when target variants are enabled.
func prepareCandidateWordlist(
	ctx context.Context,
	sourcePath string,
	candidate string,
	workspace string,
	ordinal uint64,
) (string, error) {
	if ctx == nil {
		return "", errors.New("wordlist normalization context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	workspacePath, err := requireDirectoryWorkspace(workspace)
	if err != nil {
		return "", err
	}
	if err := validateWordlistSource(sourcePath); err != nil {
		return "", err
	}
	wordlistDir := filepath.Join(workspacePath, directoryWordlistSubdir)
	if err := os.MkdirAll(wordlistDir, 0o700); err != nil {
		return "", fmt.Errorf("create candidate wordlist directory: %w", err)
	}
	destinationPath := filepath.Join(wordlistDir, fmt.Sprintf("%020d.txt", ordinal))
	if err := normalizeWordlistFileWithVariants(ctx, sourcePath, destinationPath, targetVariantPayloads(candidate)); err != nil {
		_ = os.Remove(destinationPath)
		return "", err
	}
	return destinationPath, nil
}

func prepareNormalizedWordlist(
	ctx context.Context,
	sourcePath string,
	workspace string,
) (*normalizedWordlist, error) {
	if ctx == nil {
		return nil, errors.New("wordlist normalization context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	workspacePath, err := requireDirectoryWorkspace(workspace)
	if err != nil {
		return nil, err
	}
	if err := validateWordlistSource(sourcePath); err != nil {
		return nil, err
	}

	wordlist := &normalizedWordlist{path: filepath.Join(workspacePath, directoryNormalizedListName)}
	if err := normalizeWordlistFile(ctx, sourcePath, wordlist.path); err != nil {
		_ = os.Remove(wordlist.path)
		return nil, err
	}
	return wordlist, nil
}

// normalizeWordlistFile copies the source wordlist line by line, removing at
// most one leading "/" from the payload part of each line.
func normalizeWordlistFile(ctx context.Context, sourcePath, destinationPath string) error {
	return normalizeWordlistFileWithVariants(ctx, sourcePath, destinationPath, nil)
}

// normalizeWordlistFileWithVariants additionally appends target-derived
// variants that do not already occur in the normalized source payloads.
// Appended payloads use LF framing; a source whose final line lacks a newline
// gains exactly one before the first appended variant.
func normalizeWordlistFileWithVariants(ctx context.Context, sourcePath, destinationPath string, variants []string) (err error) {
	if ctx == nil {
		return errors.New("wordlist normalization context is required")
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open wordlist source: %w", err)
	}
	defer func() {
		if closeErr := source.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close wordlist source: %w", closeErr))
		}
	}()

	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create normalized wordlist: %w", err)
	}
	defer func() {
		if closeErr := destination.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close normalized wordlist: %w", closeErr))
		}
	}()

	variantSet := make(map[string]struct{}, len(variants))
	for _, variant := range variants {
		variantSet[variant] = struct{}{}
	}
	seenVariants := make(map[string]struct{}, len(variants))
	lastTerminator := []byte(nil)

	reader := bufio.NewReaderSize(source, 64*1024)
	writer := bufio.NewWriterSize(destination, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, readErr := readWordlistLine(reader)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if len(line) == 0 && errors.Is(readErr, io.EOF) {
			break
		}
		payload, terminator := splitWordlistLine(line)
		if len(payload) > 1 && payload[0] == '/' {
			payload = payload[1:]
		}
		if _, tracked := variantSet[string(payload)]; tracked {
			seenVariants[string(payload)] = struct{}{}
		}
		if _, writeErr := writer.Write(payload); writeErr != nil {
			return fmt.Errorf("write normalized wordlist: %w", writeErr)
		}
		if _, writeErr := writer.Write(terminator); writeErr != nil {
			return fmt.Errorf("write normalized wordlist: %w", writeErr)
		}
		lastTerminator = terminator
		if errors.Is(readErr, io.EOF) {
			break
		}
	}

	if len(variants) > 0 && len(lastTerminator) == 0 && writer.Buffered() > 0 {
		if _, writeErr := writer.WriteString("\n"); writeErr != nil {
			return fmt.Errorf("write normalized wordlist: %w", writeErr)
		}
	}
	for _, variant := range variants {
		if _, already := seenVariants[variant]; already {
			continue
		}
		if _, writeErr := writer.WriteString(variant); writeErr != nil {
			return fmt.Errorf("write normalized wordlist: %w", writeErr)
		}
		if _, writeErr := writer.WriteString("\n"); writeErr != nil {
			return fmt.Errorf("write normalized wordlist: %w", writeErr)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush normalized wordlist: %w", err)
	}
	return nil
}

// readWordlistLine returns one physical line including its line terminator when
// present. The returned bytes are a private copy.
func readWordlistLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(line)+len(fragment) > directoryWordlistLineLimit {
			return nil, fmt.Errorf("wordlist line exceeds %d bytes", directoryWordlistLineLimit)
		}
		line = append(line, fragment...)
		switch {
		case err == nil:
			return line, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			return line, io.EOF
		default:
			return nil, fmt.Errorf("read wordlist: %w", err)
		}
	}
}

func splitWordlistLine(line []byte) (payload, terminator []byte) {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		terminator = line[len(line)-1:]
		payload = line[:len(line)-1]
		if len(payload) > 0 && payload[len(payload)-1] == '\r' {
			terminator = line[len(payload)-1:]
			payload = payload[:len(payload)-1]
		}
		return payload, terminator
	}
	return line, nil
}
