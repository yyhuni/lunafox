package fingerprintdetectionruntime

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
)

const maxWebsiteFactLineBytes = 4 * 1024 * 1024

// Candidate is retained as a small test/identity value. Production execution
// does not build a collection of Candidates; it writes each identity directly
// to the task-local Observer Ward input file.
type Candidate struct {
	OriginalURL  string
	ExecutionURL string
}

// MaterializeCandidates streams the immutable Website URL product into the
// task-local Observer Ward input and returns only its scalar line count.
func MaterializeCandidates(ctx context.Context, target enginecontract.Target, factsPath, workspace string) (path string, count uint64, err error) {
	if ctx == nil {
		return "", 0, errors.New("candidate context is required")
	}
	if workspace == "" {
		return "", 0, errors.New("candidate workspace is required")
	}
	workspaceInfo, err := os.Stat(workspace)
	if err != nil {
		return "", 0, fmt.Errorf("candidate workspace: %w", err)
	}
	if !workspaceInfo.IsDir() {
		return "", 0, errors.New("candidate workspace is not a directory")
	}
	if factsPath == "" {
		return "", 0, errors.New("websiteURLs facts path is required")
	}
	facts, err := os.Open(factsPath)
	if err != nil {
		return "", 0, fmt.Errorf("open websiteURLs facts: %w", err)
	}
	info, err := facts.Stat()
	if err != nil {
		_ = facts.Close()
		return "", 0, fmt.Errorf("stat websiteURLs facts: %w", err)
	}
	if !info.Mode().IsRegular() {
		_ = facts.Close()
		return "", 0, errors.New("websiteURLs facts must be a regular file")
	}
	path = filepath.Join(workspace, "observer-ward-candidates.txt")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		_ = facts.Close()
		return "", 0, fmt.Errorf("create Observer Ward candidate file: %w", err)
	}
	writer := bufio.NewWriterSize(file, 64*1024)
	count = 0
	cleanup := func(cause error) (string, uint64, error) {
		if flushErr := writer.Flush(); flushErr != nil {
			cause = errors.Join(cause, fmt.Errorf("flush Observer Ward candidate file: %w", flushErr))
		}
		if closeErr := file.Close(); closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close Observer Ward candidate file: %w", closeErr))
		}
		if closeErr := facts.Close(); closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close websiteURLs facts: %w", closeErr))
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			cause = errors.Join(cause, fmt.Errorf("remove incomplete Observer Ward candidate file: %w", removeErr))
		}
		return "", 0, cause
	}
	emit := func(raw string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := observedCandidateValidation(target, raw); err != nil {
			return err
		}
		if count == ^uint64(0) {
			return errors.New("Observer Ward candidate count overflow")
		}
		if _, err := writer.WriteString(raw + "\n"); err != nil {
			return err
		}
		count++
		return nil
	}
	if err := emitTargetBaselines(ctx, target, emit); err != nil {
		return cleanup(fmt.Errorf("write Target baseline: %w", err))
	}
	if err := readWebsiteURLFacts(facts, func(line string) error {
		if err := emit(line); err != nil {
			return fmt.Errorf("write finalized website URL: %w", err)
		}
		return nil
	}); err != nil {
		return cleanup(fmt.Errorf("read websiteURLs facts: %w", err))
	}
	if err := facts.Close(); err != nil {
		return cleanup(fmt.Errorf("close websiteURLs facts: %w", err))
	}
	if err := writer.Flush(); err != nil {
		return cleanup(fmt.Errorf("flush Observer Ward candidate file: %w", err))
	}
	if err := file.Close(); err != nil {
		return cleanup(fmt.Errorf("close Observer Ward candidate file: %w", err))
	}
	return path, count, nil
}

func cleanupObserverCandidateArtifacts(workspace string) error {
	if workspace == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(workspace, "observer-ward-candidates.txt"))
}

func emitTargetBaselines(ctx context.Context, target enginecontract.Target, emit func(string) error) error {
	if ctx == nil || emit == nil {
		return errors.New("candidate context and emitter are required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if target.Value == "" {
		return errors.New("Target value is required")
	}
	switch target.Type {
	case enginecontract.TargetTypeDomain, enginecontract.TargetTypeIP:
		if target.Type == enginecontract.TargetTypeIP {
			address, err := netip.ParseAddr(target.Value)
			if err != nil || !address.Is4() || address.String() != target.Value {
				return errors.New("Target IP must be a canonical IPv4")
			}
		}
		if err := emit("http://" + target.Value); err != nil {
			return err
		}
		return emit("https://" + target.Value)
	case enginecontract.TargetTypeCIDR:
		prefix, err := netip.ParsePrefix(target.Value)
		if err != nil || !prefix.Addr().Is4() || prefix != prefix.Masked() || prefix.String() != target.Value {
			return errors.New("Target CIDR must be a canonical IPv4 CIDR")
		}
		startBytes := prefix.Addr().As4()
		start := binary.BigEndian.Uint32(startBytes[:])
		span := uint64(1) << uint(32-prefix.Bits())
		end := uint64(start) + span - 1
		for current := uint64(start); current <= end; current++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			address := netip.AddrFrom4([4]byte{byte(current >> 24), byte(current >> 16), byte(current >> 8), byte(current)}).String()
			if err := emit("http://" + address); err != nil {
				return err
			}
			if err := emit("https://" + address); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported Target type %q", target.Type)
	}
}

func observedCandidateValidation(target enginecontract.Target, raw string) error {
	host, err := deriveCandidateHost(raw)
	if err != nil {
		return err
	}
	if !candidateHostMatchesTarget(target, host) {
		return errors.New("URL authority is outside the Target")
	}
	return nil
}

// deriveCandidateHost extracts only the authority needed for Target scope.
// It intentionally leaves all bytes after the authority untouched.
func deriveCandidateHost(raw string) (string, error) {
	if strings.ContainsAny(raw, "\x00\r\n") {
		return "", errors.New("candidate URL contains unsafe control characters")
	}
	offset := 0
	switch {
	case len(raw) >= len("http://") && strings.EqualFold(raw[:len("http://")], "http://"):
		offset = len("http://")
	case len(raw) >= len("https://") && strings.EqualFold(raw[:len("https://")], "https://"):
		offset = len("https://")
	default:
		return "", errors.New("candidate must be an HTTP URL with an authority")
	}
	authorityEnd := len(raw)
	for index := offset; index < len(raw); index++ {
		if strings.ContainsRune("/?#", rune(raw[index])) {
			authorityEnd = index
			break
		}
	}
	authority := raw[offset:authorityEnd]
	if authority == "" {
		return "", errors.New("candidate URL authority is required")
	}
	hostPort := authority
	if at := strings.LastIndexByte(hostPort, '@'); at >= 0 {
		hostPort = hostPort[at+1:]
	}
	if hostPort == "" || strings.TrimSpace(hostPort) != hostPort || strings.HasPrefix(hostPort, "[") || strings.Count(hostPort, ":") > 1 {
		return "", errors.New("candidate URL authority host is invalid")
	}
	host := hostPort
	if colon := strings.IndexByte(hostPort, ':'); colon >= 0 {
		host = hostPort[:colon]
	}
	if host == "" || strings.TrimSpace(host) != host {
		return "", errors.New("candidate URL authority host is invalid")
	}
	return host, nil
}

func readWebsiteURLFacts(reader io.Reader, visit func(string) error) error {
	if reader == nil || visit == nil {
		return errors.New("websiteURLs fact reader and visitor are required")
	}
	buffered := bufio.NewReaderSize(reader, maxWebsiteFactLineBytes+1)
	for {
		line, err := buffered.ReadSlice('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maxWebsiteFactLineBytes)
		}
		if err != nil {
			return fmt.Errorf("read websiteURLs facts: %w", err)
		}
		line = line[:len(line)-1]
		if len(line) > maxWebsiteFactLineBytes {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maxWebsiteFactLineBytes)
		}
		value := string(line)
		if value == "" || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("websiteURLs fact line contains unsafe control characters")
		}
		if err := visit(value); err != nil {
			return err
		}
	}
}

func candidateHostMatchesTarget(target enginecontract.Target, host string) bool {
	switch target.Type {
	case enginecontract.TargetTypeDomain:
		return strings.EqualFold(host, target.Value) || strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(target.Value))
	case enginecontract.TargetTypeIP:
		return host == target.Value
	case enginecontract.TargetTypeCIDR:
		prefix, err := netip.ParsePrefix(target.Value)
		address, addressErr := netip.ParseAddr(host)
		return err == nil && addressErr == nil && address.Is4() && prefix.Contains(address)
	default:
		return false
	}
}
