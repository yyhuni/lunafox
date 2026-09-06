package urlcollectionruntime

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

	enginecontract "github.com/yyhuni/lunafox/engines/url_collection/contract"
)

const maxWebsiteURLFactLineBytes = 4 * 1024 * 1024

// prepareTargetSeeds materializes the complete baseline-first Katana seed in
// the writable workspace. Confirmed Website URL facts are copied as-is after
// applying only the existing bounded baseline-overlap rules.
func prepareTargetSeeds(ctx context.Context, target enginecontract.Target, factsPath, workDir string) (path string, count uint64, err error) {
	if ctx == nil {
		return "", 0, errors.New("seed context is required")
	}
	if workDir == "" {
		return "", 0, errors.New("seed workspace is required")
	}
	info, statErr := os.Stat(workDir)
	if statErr != nil {
		return "", 0, fmt.Errorf("seed workspace: %w", statErr)
	}
	if !info.IsDir() {
		return "", 0, errors.New("seed workspace is not a directory")
	}
	if factsPath == "" {
		return "", 0, errors.New("websiteURLs facts path is required")
	}

	path = filepath.Join(workDir, "katana-seeds.txt")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create Katana seed file: %w", err)
	}
	cleanup := func(cause error) (string, uint64, error) {
		closeErr := file.Close()
		removeErr := os.Remove(path)
		if closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close Katana seed file: %w", closeErr))
		}
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			cause = errors.Join(cause, fmt.Errorf("remove incomplete Katana seed file: %w", removeErr))
		}
		return "", 0, cause
	}

	writer := bufio.NewWriterSize(file, 64*1024)
	writeLine := func(line string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := incrementUint64(&count, "Katana seed record count"); err != nil {
			return err
		}
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
		return nil
	}

	if err := writeURLTargetBaseline(ctx, target, writeLine); err != nil {
		return cleanup(fmt.Errorf("write Target baseline: %w", err))
	}

	facts, err := os.Open(factsPath)
	if err != nil {
		return cleanup(fmt.Errorf("open websiteURLs facts %s: %w", factsPath, err))
	}
	if info, statErr := facts.Stat(); statErr != nil {
		_ = facts.Close()
		return cleanup(fmt.Errorf("stat websiteURLs facts %s: %w", factsPath, statErr))
	} else if !info.Mode().IsRegular() {
		_ = facts.Close()
		return cleanup(errors.New("websiteURLs facts must be a regular file"))
	}

	if readErr := readWebsiteURLFactLines(facts, func(line string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if shouldSkipTargetBaselineURL(target, line) {
			return nil
		}
		if err := writeLine(line); err != nil {
			return fmt.Errorf("write websiteURLs fact: %w", err)
		}
		return nil
	}); readErr != nil {
		_ = facts.Close()
		return cleanup(fmt.Errorf("read websiteURLs facts: %w", readErr))
	}
	if closeErr := facts.Close(); closeErr != nil {
		return cleanup(fmt.Errorf("close websiteURLs facts: %w", closeErr))
	}
	if flushErr := writer.Flush(); flushErr != nil {
		return cleanup(fmt.Errorf("flush Katana seed file: %w", flushErr))
	}
	if closeErr := file.Close(); closeErr != nil {
		return cleanup(fmt.Errorf("close Katana seed file: %w", closeErr))
	}
	return path, count, nil
}

func incrementUint64(value *uint64, label string) error {
	if value == nil {
		return errors.New("counter is required")
	}
	if *value == ^uint64(0) {
		return fmt.Errorf("%s overflow", label)
	}
	*value++
	return nil
}

// readWebsiteURLFactLines preserves each confirmed URL byte-for-byte while
// enforcing only the line framing needed to produce safe Katana input. URL
// syntax, host ownership, and byte limits remain Server concerns.
func readWebsiteURLFactLines(reader io.Reader, visit func(string) error) error {
	if reader == nil || visit == nil {
		return errors.New("websiteURLs fact reader and visitor are required")
	}
	buffered := bufio.NewReaderSize(reader, maxWebsiteURLFactLineBytes+1)
	for {
		line, err := buffered.ReadSlice('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maxWebsiteURLFactLineBytes)
		}
		if err != nil {
			return fmt.Errorf("read websiteURLs facts: %w", err)
		}
		if len(line) == 0 || line[len(line)-1] != '\n' {
			return errors.New("websiteURLs fact line must be LF-terminated")
		}
		line = line[:len(line)-1]
		if len(line) > maxWebsiteURLFactLineBytes {
			return fmt.Errorf("websiteURLs fact line exceeds %d bytes", maxWebsiteURLFactLineBytes)
		}
		if strings.ContainsAny(string(line), "\x00\r") {
			return errors.New("websiteURLs fact line contains unsafe control characters")
		}
		if err := visit(string(line)); err != nil {
			return err
		}
	}
}

func writeURLTargetBaseline(ctx context.Context, target enginecontract.Target, emit func(string) error) error {
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
			parsed := net.ParseIP(target.Value)
			if parsed == nil || parsed.To4() == nil || parsed.String() != target.Value {
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
			ip := ipv4Uint32String(current)
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

func shouldSkipTargetBaselineURL(target enginecontract.Target, raw string) bool {
	switch target.Type {
	case enginecontract.TargetTypeDomain, enginecontract.TargetTypeIP:
		// Only exact raw root values overlap the generated baseline. Explicit
		// ports, paths, queries, fragments, case variants, and percent spelling
		// remain confirmed facts.
		return raw == "http://"+target.Value || raw == "https://"+target.Value
	case enginecontract.TargetTypeCIDR:
		return isCIDRRootBaselineURL(target.Value, raw)
	default:
		return false
	}
}

// isCIDRRootBaselineURL recognizes only the byte form emitted by
// writeURLTargetBaseline. It deliberately does not use a general URL parser:
// parser-hostile bytes in a confirmed path or query must remain a candidate.
func isCIDRRootBaselineURL(cidr, raw string) bool {
	var address string
	switch {
	case strings.HasPrefix(raw, "http://"):
		address = raw[len("http://"):]
	case strings.HasPrefix(raw, "https://"):
		address = raw[len("https://"):]
	default:
		return false
	}
	if address == "" || strings.ContainsAny(address, "/:?#@") {
		return false
	}
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() == nil || ip.String() != address {
		return false
	}
	start, end, err := ipv4CIDRBounds(cidr)
	if err != nil {
		return false
	}
	value := binary.BigEndian.Uint32(ip.To4())
	return value >= start && value <= end
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

func ipv4Uint32String(value uint32) string {
	var bytes [4]byte
	binary.BigEndian.PutUint32(bytes[:], value)
	return net.IP(bytes[:]).String()
}
