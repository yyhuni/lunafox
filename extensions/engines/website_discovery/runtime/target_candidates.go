package websitediscoveryruntime

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	websitediscoverycontract "github.com/yyhuni/lunafox/engines/website_discovery/contract"
)

type hostPortFact struct {
	Host string `json:"host"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// prepareTargetCandidates writes the complete baseline-first HTTPX input in
// the writable workspace. A malformed hostPorts record fails preparation
// instead of changing its scan identity through an IP fallback.
func prepareTargetCandidates(ctx context.Context, target websitediscoverycontract.Target, factsPath, workDir string) (path string, count uint64, err error) {
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
		return "", 0, errors.New("hostPorts facts path is required")
	}

	path = filepath.Join(workDir, "httpx-candidates.txt")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("create HTTPX candidate file: %w", err)
	}
	cleanup := func(cause error) (string, uint64, error) {
		closeErr := file.Close()
		removeErr := os.Remove(path)
		if closeErr != nil {
			cause = errors.Join(cause, fmt.Errorf("close HTTPX candidate file: %w", closeErr))
		}
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			cause = errors.Join(cause, fmt.Errorf("remove incomplete HTTPX candidate file: %w", removeErr))
		}
		return "", 0, cause
	}

	writer := bufio.NewWriterSize(file, 64*1024)
	writeLine := func(line string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if line == "" || strings.ContainsAny(line, "\r\n") {
			return errors.New("HTTPX candidate line is invalid")
		}
		if count == ^uint64(0) {
			return errors.New("HTTPX candidate count overflow")
		}
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n"); err != nil {
			return err
		}
		count++
		return nil
	}

	if err := writeWebsiteTargetBaseline(ctx, target, writeLine); err != nil {
		return cleanup(fmt.Errorf("write Target baseline: %w", err))
	}

	facts, err := os.Open(factsPath)
	if err != nil {
		return cleanup(fmt.Errorf("open hostPorts facts %s: %w", factsPath, err))
	}
	if info, statErr := facts.Stat(); statErr != nil {
		_ = facts.Close()
		return cleanup(fmt.Errorf("stat hostPorts facts %s: %w", factsPath, statErr))
	} else if !info.Mode().IsRegular() {
		_ = facts.Close()
		return cleanup(errors.New("hostPorts facts must be a regular file"))
	}

	if closeErr := facts.Close(); closeErr != nil {
		return cleanup(fmt.Errorf("close hostPorts facts: %w", closeErr))
	}

	sortDir, err := os.MkdirTemp(workDir, ".website-discovery-sort-")
	if err != nil {
		return cleanup(fmt.Errorf("create HostPort sort workspace: %w", err))
	}
	defer func() {
		if removeErr := os.RemoveAll(sortDir); removeErr != nil {
			if err == nil {
				path, count = "", 0
				err = fmt.Errorf("remove HostPort sort workspace: %w", removeErr)
			} else {
				err = errors.Join(err, fmt.Errorf("remove HostPort sort workspace: %w", removeErr))
			}
		}
	}()
	chunks, _, err := writeHostPortSortChunks(ctx, factsPath, sortDir, target)
	if err != nil {
		return cleanup(err)
	}
	mergedPath, err := mergeHostPortChunks(ctx, chunks, sortDir)
	if err != nil {
		return cleanup(err)
	}
	merged, err := os.Open(mergedPath)
	if err != nil {
		return cleanup(fmt.Errorf("open merged HostPort candidates: %w", err))
	}
	if err := copyHostPortCandidates(ctx, merged, writeLine); err != nil {
		_ = merged.Close()
		return cleanup(err)
	}
	if err := merged.Close(); err != nil {
		return cleanup(fmt.Errorf("close merged HostPort candidates: %w", err))
	}
	if err := writer.Flush(); err != nil {
		return cleanup(fmt.Errorf("flush HTTPX candidate file: %w", err))
	}
	if closeErr := file.Close(); closeErr != nil {
		return cleanup(fmt.Errorf("close HTTPX candidate file: %w", closeErr))
	}
	return path, count, nil
}

func parseHostPortFact(payload []byte) (hostPortFact, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return hostPortFact{}, err
	}
	if len(fields) != 3 {
		return hostPortFact{}, errors.New("hostPorts fact must contain exactly host, ip, and port")
	}
	for _, key := range []string{"host", "ip", "port"} {
		if _, ok := fields[key]; !ok {
			return hostPortFact{}, fmt.Errorf("hostPorts fact field %q is required", key)
		}
	}
	var fact hostPortFact
	if err := json.Unmarshal(payload, &fact); err != nil {
		return hostPortFact{}, err
	}
	if fact.Host == "" || strings.TrimSpace(fact.Host) != fact.Host || strings.ContainsAny(fact.Host, "\t\r\n") {
		return hostPortFact{}, errors.New("hostPorts fact host is required to build an HTTPX candidate")
	}
	if !canonicalHost(fact.Host) {
		return hostPortFact{}, errors.New("hostPorts fact host must be canonical")
	}
	ip, err := netip.ParseAddr(fact.IP)
	if err != nil || !ip.Is4() || ip.String() != fact.IP {
		return hostPortFact{}, errors.New("hostPorts fact ip must be a canonical IPv4")
	}
	if fact.Port < 1 || fact.Port > 65535 {
		return hostPortFact{}, errors.New("hostPorts fact port must be between 1 and 65535")
	}
	return fact, nil
}

func writeWebsiteTargetBaseline(ctx context.Context, target websitediscoverycontract.Target, emit func(string) error) error {
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
	case websitediscoverycontract.TargetTypeDomain, websitediscoverycontract.TargetTypeIP:
		if target.Type == websitediscoverycontract.TargetTypeIP {
			parsed := net.ParseIP(target.Value)
			if parsed == nil || parsed.To4() == nil || parsed.String() != target.Value {
				return errors.New("Target IP must be a canonical IPv4")
			}
		}
		if err := emit("http://" + target.Value); err != nil {
			return err
		}
		return emit("https://" + target.Value)
	case websitediscoverycontract.TargetTypeCIDR:
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

func targetBaselineOverlapsHostPort(target websitediscoverycontract.Target, host string, port int) bool {
	if port != 80 && port != 443 {
		return false
	}
	switch target.Type {
	case websitediscoverycontract.TargetTypeDomain, websitediscoverycontract.TargetTypeIP:
		return host == target.Value
	case websitediscoverycontract.TargetTypeCIDR:
		start, end, err := ipv4CIDRBounds(target.Value)
		if err != nil {
			return false
		}
		parsed := net.ParseIP(host)
		if parsed == nil || parsed.To4() == nil || parsed.String() != host {
			return false
		}
		value := binary.BigEndian.Uint32(parsed.To4())
		return value >= start && value <= end
	default:
		return false
	}
}

func hostPortURLs(host string, port int) []string {
	switch port {
	case 80:
		return []string{"http://" + host}
	case 443:
		return []string{"https://" + host}
	default:
		authority := net.JoinHostPort(host, strconv.Itoa(port))
		return []string{"http://" + authority, "https://" + authority}
	}
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
