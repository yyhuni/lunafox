package portscanruntime

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

	portscancontract "github.com/yyhuni/lunafox/engines/port_scan/contract"
)

// prepareTargetCandidates materializes the complete Naabu input in the
// writable task workspace. The mounted fact file remains read-only and is
// copied record-for-record after the Engine-owned Target baseline.
func prepareTargetCandidates(ctx context.Context, target portscancontract.Target, factsPath, workDir string) (path string, count uint64, err error) {
	if ctx == nil {
		return "", 0, errors.New("candidate context is required")
	}
	if workDir == "" {
		return "", 0, errors.New("candidate workspace is required")
	}
	info, statErr := os.Stat(workDir)
	if statErr != nil {
		return "", 0, fmt.Errorf("candidate workspace: %w", statErr)
	}
	if !info.IsDir() {
		return "", 0, errors.New("candidate workspace is not a directory")
	}
	if factsPath == "" {
		return "", 0, errors.New("subdomains facts path is required")
	}

	path = filepath.Join(workDir, "naabu-candidates.txt")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create Naabu candidate file: %w", err)
	}
	cleanup := func(cause error) (string, uint64, error) {
		closeErr := file.Close()
		removeErr := os.Remove(path)
		if closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close Naabu candidate file: %w", closeErr))
		}
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			cause = errors.Join(cause, fmt.Errorf("remove incomplete Naabu candidate file: %w", removeErr))
		}
		return "", 0, cause
	}

	writer := bufio.NewWriterSize(file, 64*1024)
	writeLine := func(line string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if line == "" || strings.TrimSpace(line) != line || strings.ContainsAny(line, "\r\n") {
			return errors.New("subdomains fact contains an invalid candidate line")
		}
		if err := incrementUint64(&count, "Naabu candidate count"); err != nil {
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

	if err := writeTargetBaseline(ctx, target, writeLine); err != nil {
		return cleanup(fmt.Errorf("write Target baseline: %w", err))
	}

	facts, err := os.Open(factsPath)
	if err != nil {
		return cleanup(fmt.Errorf("open subdomains facts %s: %w", factsPath, err))
	}
	if info, statErr := facts.Stat(); statErr != nil {
		_ = facts.Close()
		return cleanup(fmt.Errorf("stat subdomains facts %s: %w", factsPath, statErr))
	} else if !info.Mode().IsRegular() {
		_ = facts.Close()
		return cleanup(errors.New("subdomains facts must be a regular file"))
	}
	reader := bufio.NewReaderSize(facts, 4*1024*1024+1)
	for {
		line, readErr := reader.ReadString('\n')
		if errors.Is(readErr, io.EOF) {
			if line == "" {
				break
			}
			_ = facts.Close()
			return cleanup(errors.New("subdomains fact line must be LF-terminated"))
		}
		if errors.Is(readErr, bufio.ErrBufferFull) {
			_ = facts.Close()
			return cleanup(errors.New("subdomains fact line exceeds 4 MiB"))
		}
		if readErr != nil {
			_ = facts.Close()
			return cleanup(fmt.Errorf("read subdomains facts: %w", readErr))
		}
		line = strings.TrimSuffix(line, "\n")
		if len(line) > 4*1024*1024 {
			_ = facts.Close()
			return cleanup(errors.New("subdomains fact line exceeds 4 MiB"))
		}
		if err := writeLine(line); err != nil {
			_ = facts.Close()
			return cleanup(fmt.Errorf("write subdomains fact: %w", err))
		}
	}
	if closeErr := facts.Close(); closeErr != nil {
		return cleanup(fmt.Errorf("close subdomains facts: %w", closeErr))
	}
	if flushErr := writer.Flush(); flushErr != nil {
		return cleanup(fmt.Errorf("flush Naabu candidate file: %w", flushErr))
	}
	if closeErr := file.Close(); closeErr != nil {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("close Naabu candidate file: %w", closeErr)
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

func writeTargetBaseline(ctx context.Context, target portscancontract.Target, emit func(string) error) error {
	if ctx == nil || emit == nil {
		return errors.New("Target baseline writer is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if target.Value == "" {
		return errors.New("Target value is required")
	}
	switch target.Type {
	case portscancontract.TargetTypeDomain:
		return emit(target.Value)
	case portscancontract.TargetTypeIP:
		parsed := net.ParseIP(target.Value)
		if parsed == nil || parsed.To4() == nil || parsed.String() != target.Value {
			return errors.New("Target IP must be a canonical IPv4")
		}
		return emit(target.Value)
	case portscancontract.TargetTypeCIDR:
		start, end, err := ipv4CIDRBounds(target.Value)
		if err != nil {
			return err
		}
		for current := start; ; current++ {
			if err := emit(ipv4Uint32String(current)); err != nil {
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

func ipv4CIDRBounds(value string) (uint32, uint32, error) {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		return 0, 0, fmt.Errorf("Target must be a canonical IPv4 CIDR")
	}
	ones, bits := network.Mask.Size()
	if bits != 32 || network.IP.To4() == nil {
		return 0, 0, errors.New("Target must be a canonical IPv4 CIDR")
	}
	start := binary.BigEndian.Uint32(network.IP.To4())
	hostBits := 32 - ones
	size := uint64(1) << hostBits
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
