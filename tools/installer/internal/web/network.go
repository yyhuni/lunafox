package web

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"
)

func listNetworkCandidates() ([]networkCandidate, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]networkCandidate, 0, 8)
	for _, iface := range interfaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		if shouldSkipInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, ok := extractIP(addr)
			if !ok {
				continue
			}
			if ip.To4() == nil || ip.IsLoopback() {
				continue
			}
			ipText := ip.String()
			if _, exists := seen[ipText]; exists {
				continue
			}
			seen[ipText] = struct{}{}
			out = append(out, networkCandidate{
				Interface: iface.Name,
				IP:        ipText,
				Label:     fmt.Sprintf("%s (%s)", ipText, iface.Name),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].IP < out[j].IP
	})
	return out, nil
}

func shouldSkipInterface(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return true
	}
	prefixes := []string{
		"lo", "docker", "br-", "veth", "cni", "flannel", "virbr", "podman",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

func extractIP(addr net.Addr) (net.IP, bool) {
	switch typed := addr.(type) {
	case *net.IPNet:
		return typed.IP, true
	case *net.IPAddr:
		return typed.IP, true
	default:
		return nil, false
	}
}

func isLoopbackHost(host string) bool {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]"))
	if normalized == "" {
		return false
	}
	if normalized == "localhost" {
		return true
	}
	if normalized == "127.0.0.1" || normalized == "::1" {
		return true
	}
	if ip := net.ParseIP(normalized); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func isLocalHost(host string, candidates []networkCandidate) (bool, error) {
	normalized := strings.Trim(strings.TrimSpace(host), "[]")
	if normalized == "" {
		return false, nil
	}
	if isLoopbackHost(normalized) {
		return true, nil
	}

	candidateSet := map[string]struct{}{
		"127.0.0.1": {},
		"::1":       {},
	}
	for _, candidate := range candidates {
		candidateSet[candidate.IP] = struct{}{}
	}

	if ip := net.ParseIP(normalized); ip != nil {
		_, ok := candidateSet[ip.String()]
		return ok, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, normalized)
	if err != nil {
		return false, err
	}
	for _, ip := range ips {
		if _, ok := candidateSet[ip.IP.String()]; ok || ip.IP.IsLoopback() {
			return true, nil
		}
	}
	return false, nil
}

func isTCPPortAvailable(port string) bool {
	address := ":" + strings.TrimSpace(port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
