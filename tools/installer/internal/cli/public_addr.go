package cli

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var domainLabelPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

type NetworkCandidate struct {
	Interface string
	IP        string
	Label     string
}

func NormalizePublicURL(raw string, fallbackPort string) (string, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", fmt.Errorf("--public-url 不能为空")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", fmt.Errorf("--public-url 不合法: %w", err)
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", "", fmt.Errorf("--public-url 仅支持 http 或 https")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", "", fmt.Errorf("--public-url 缺少 host")
	}
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	if port == "" {
		port = strings.TrimSpace(fallbackPort)
	}
	if port == "" {
		port = DefaultPublicPort
	}
	if err := ValidatePublicPort(port); err != nil {
		return "", "", err
	}

	final := fmt.Sprintf("%s://%s", scheme, parsed.Hostname())
	if strings.Contains(host, ":") {
		final = fmt.Sprintf("%s://[%s]", scheme, host)
	}
	final = fmt.Sprintf("%s:%s", final, port)
	return final, port, nil
}

func NormalizePublicHostPort(hostRaw string, portRaw string) (string, string, error) {
	host, err := ParsePublicHostInput(hostRaw)
	if err != nil {
		return "", "", fmt.Errorf("公网主机不合法: %w", err)
	}
	if host == "" {
		return "", "", fmt.Errorf("公网主机不合法: 不能为空")
	}
	port := strings.TrimSpace(portRaw)
	if port == "" {
		port = DefaultPublicPort
	}
	if err := ValidatePublicPort(port); err != nil {
		return "", "", fmt.Errorf("公网端口不合法: %w", err)
	}
	return BuildPublicURL(host, port), port, nil
}

func ValidatePublicPort(raw string) error {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("公网端口必须是数字")
	}
	if value < 1 || value > 65535 {
		return fmt.Errorf("公网端口必须在 1-65535 之间")
	}
	return nil
}

func ParsePublicHostInput(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}

	if strings.Contains(value, "://") {
		return "", fmt.Errorf("不能包含协议，请只填写 IP 或域名")
	}
	if strings.ContainsAny(value, "/?#") {
		return "", fmt.Errorf("不能包含路径或查询参数")
	}
	if strings.HasPrefix(value, "[") != strings.HasSuffix(value, "]") {
		return "", fmt.Errorf("IPv6 方括号格式不完整")
	}

	wrappedIPv6 := strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]")
	if strings.Contains(value, ":") && strings.Count(value, ":") == 1 && !wrappedIPv6 {
		return "", fmt.Errorf("不能包含端口，请在公网端口单独填写")
	}

	host := strings.Trim(strings.TrimSpace(value), "[]")
	if host == "" {
		return "", fmt.Errorf("缺少 host")
	}
	if strings.ContainsAny(host, " \t\r\n") {
		return "", fmt.Errorf("不能包含空白字符")
	}
	if wrappedIPv6 && !strings.Contains(host, ":") {
		return "", fmt.Errorf("方括号仅用于 IPv6 地址")
	}

	if ip := net.ParseIP(host); ip != nil {
		return host, nil
	}
	if looksLikeIPv4(host) {
		return "", fmt.Errorf("IPv4 地址格式不合法")
	}
	if !isValidDomain(host) {
		return "", fmt.Errorf("域名格式不合法，仅支持字母/数字/中划线和点分标签")
	}
	return host, nil
}

func BuildPublicURL(host string, port string) string {
	trimmedHost := strings.Trim(strings.TrimSpace(host), "[]")
	hostWithPort := net.JoinHostPort(trimmedHost, strings.TrimSpace(port))
	return "https://" + hostWithPort
}

func IsLoopbackHost(host string) bool {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]"))
	if normalized == "" {
		return false
	}
	if normalized == "localhost" || normalized == "127.0.0.1" || normalized == "::1" {
		return true
	}
	if ip := net.ParseIP(normalized); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func ListNetworkCandidates() ([]NetworkCandidate, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]NetworkCandidate, 0, 8)
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
			out = append(out, NetworkCandidate{
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

func looksLikeIPv4(host string) bool {
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func isValidDomain(host string) bool {
	trimmed := strings.TrimSpace(host)
	if trimmed == "" || len(trimmed) > 253 {
		return false
	}
	if strings.HasPrefix(trimmed, ".") || strings.HasSuffix(trimmed, ".") {
		return false
	}
	labels := strings.Split(trimmed, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		if !domainLabelPattern.MatchString(label) {
			return false
		}
	}
	return true
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
