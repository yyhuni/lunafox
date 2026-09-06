package screenshotruntime

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/screenshot/contract"
)

const maxScreenshotFactLineBytes = 4 * 1024 * 1024

// materializeCandidates writes the Engine-owned baseline-first candidate set.
// The finalized facts file is read-only and remains task-private to the Agent.
func materializeCandidates(ctx context.Context, target enginecontract.Target, factsPath, workspace string) (path string, count uint64, err error) {
	if ctx == nil {
		return "", 0, errors.New("candidate context is required")
	}
	if workspace == "" {
		return "", 0, errors.New("candidate workspace is required")
	}
	info, err := os.Stat(workspace)
	if err != nil {
		return "", 0, fmt.Errorf("candidate workspace: %w", err)
	}
	if !info.IsDir() {
		return "", 0, errors.New("candidate workspace is not a directory")
	}
	if factsPath == "" {
		return "", 0, errors.New("websiteURLs facts path is required")
	}
	path = filepath.Join(workspace, "httpx-candidates.txt")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create HTTPX candidate file: %w", err)
	}
	writer := bufio.NewWriterSize(file, 64*1024)
	count = 0
	write := func(raw string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := validateCandidateTarget(target, raw); err != nil {
			return fmt.Errorf("candidate URL is invalid: %w", err)
		}
		if count == ^uint64(0) {
			return errors.New("HTTPX candidate count overflow")
		}
		if _, err := writer.WriteString(raw + "\n"); err != nil {
			return err
		}
		count++
		return nil
	}
	cleanup := func(cause error) (string, uint64, error) {
		if flushErr := writer.Flush(); flushErr != nil {
			cause = errors.Join(cause, fmt.Errorf("flush HTTPX candidate file: %w", flushErr))
		}
		if closeErr := file.Close(); closeErr != nil {
			cause = errors.Join(cause, closeErr)
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			cause = errors.Join(cause, fmt.Errorf("remove incomplete HTTPX candidate file: %w", removeErr))
		}
		return "", 0, cause
	}
	if err := writeTargetBaseline(ctx, target, write); err != nil {
		return cleanup(fmt.Errorf("write Target baseline: %w", err))
	}
	facts, err := os.Open(factsPath)
	if err != nil {
		return cleanup(fmt.Errorf("open websiteURLs facts: %w", err))
	}
	factsInfo, err := facts.Stat()
	if err != nil {
		_ = facts.Close()
		return cleanup(fmt.Errorf("stat websiteURLs facts: %w", err))
	}
	if !factsInfo.Mode().IsRegular() {
		_ = facts.Close()
		return cleanup(errors.New("websiteURLs facts must be a regular file"))
	}
	readErr := readWebsiteURLFacts(facts, func(line string) error {
		if err := write(line); err != nil {
			return fmt.Errorf("write finalized website URL: %w", err)
		}
		return nil
	})
	if readErr != nil {
		_ = facts.Close()
		return cleanup(fmt.Errorf("read websiteURLs facts: %w", readErr))
	}
	if err := facts.Close(); err != nil {
		return cleanup(fmt.Errorf("close websiteURLs facts: %w", err))
	}
	if err := writer.Flush(); err != nil {
		return cleanup(fmt.Errorf("flush HTTPX candidate file: %w", err))
	}
	if err := file.Close(); err != nil {
		return cleanup(fmt.Errorf("close HTTPX candidate file: %w", err))
	}
	return path, count, nil
}

// readWebsiteURLFacts keeps confirmed URL bytes intact while enforcing only
// the LF framing and bounded record size required by the HTTPX input file.
func readWebsiteURLFacts(reader io.Reader, visit func(string) error) error {
	if reader == nil || visit == nil {
		return errors.New("websiteURLs fact reader and visitor are required")
	}
	buffered := bufio.NewReaderSize(reader, maxScreenshotFactLineBytes+1)
	for {
		line, err := buffered.ReadSlice('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maxScreenshotFactLineBytes)
		}
		if err != nil {
			return fmt.Errorf("read websiteURLs facts: %w", err)
		}
		if len(line) == 0 || line[len(line)-1] != '\n' {
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		line = line[:len(line)-1]
		if len(line) > maxScreenshotFactLineBytes {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maxScreenshotFactLineBytes)
		}
		if strings.ContainsAny(string(line), "\x00\r") {
			return errors.New("websiteURLs fact line contains unsafe control characters")
		}
		if err := visit(string(line)); err != nil {
			return err
		}
	}
}

func writeTargetBaseline(ctx context.Context, target enginecontract.Target, emit func(string) error) error {
	if ctx == nil || emit == nil {
		return errors.New("Target baseline context and emitter are required")
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
			ip := net.ParseIP(target.Value)
			if ip == nil || ip.To4() == nil || ip.String() != target.Value {
				return errors.New("Target IP must be a canonical IPv4")
			}
		}
		if err := emit("http://" + target.Value); err != nil {
			return err
		}
		return emit("https://" + target.Value)
	case enginecontract.TargetTypeCIDR:
		start, end, err := ipv4CIDRBounds(target.Value)
		if err != nil {
			return err
		}
		for current := start; ; current++ {
			ip := ipv4String(current)
			if err := emit("http://" + ip); err != nil {
				return err
			}
			if err := emit("https://" + ip); err != nil {
				return err
			}
			if current == end {
				return nil
			}
		}
	default:
		return fmt.Errorf("unsupported Target type %q", target.Type)
	}
}

// validateCandidateTarget derives authority only for Target applicability. It
// deliberately never parses or rebuilds the stored URL, whose bytes are the
// Screenshot identity passed to HTTPX.
func validateCandidateTarget(target enginecontract.Target, raw string) error {
	host, err := deriveCandidateHost(raw)
	if err != nil {
		return err
	}
	switch target.Type {
	case enginecontract.TargetTypeDomain:
		if strings.EqualFold(host, target.Value) || strings.HasSuffix(strings.ToLower(host), "."+strings.ToLower(target.Value)) {
			return nil
		}
	case enginecontract.TargetTypeIP:
		if host == target.Value {
			return nil
		}
	case enginecontract.TargetTypeCIDR:
		ip := net.ParseIP(host)
		if ip != nil && ip.To4() != nil {
			start, end, boundsErr := ipv4CIDRBounds(target.Value)
			if boundsErr == nil {
				value := binary.BigEndian.Uint32(ip.To4())
				if value >= start && value <= end {
					return nil
				}
			}
		}
	}
	return errors.New("URL authority is outside the Target")
}

// deriveCandidateHost extracts only the authority needed for Target scope.
// It intentionally leaves the path, query, and fragment untouched because
// their raw spelling is part of the URL passed to HTTPX.
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
		switch raw[index] {
		case '/', '?', '#':
			authorityEnd = index
			index = len(raw)
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

func ipv4CIDRBounds(value string) (uint32, uint32, error) {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		return 0, 0, errors.New("Target must be a canonical IPv4 CIDR")
	}
	ones, bits := network.Mask.Size()
	if bits != 32 || network.IP.To4() == nil {
		return 0, 0, errors.New("Target must be a canonical IPv4 CIDR")
	}
	start := binary.BigEndian.Uint32(network.IP.To4())
	size := uint64(1) << (32 - ones)
	end := uint64(start) + size - 1
	if end > uint64(^uint32(0)) {
		return 0, 0, errors.New("Target IPv4 CIDR upper bound overflow")
	}
	return start, uint32(end), nil
}

func ipv4String(value uint32) string {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	return net.IP(raw[:]).String()
}
