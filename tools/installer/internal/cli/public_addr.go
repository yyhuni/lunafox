package cli

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

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
	normalizedHost, err := ParsePublicHostInput(host)
	if err != nil {
		return "", "", fmt.Errorf("--public-url host 不合法: %w", err)
	}
	if normalizedHost == "" {
		return "", "", fmt.Errorf("--public-url host 不合法: 不能为空")
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

	final := fmt.Sprintf("%s://%s", scheme, normalizedHost)
	if strings.Contains(normalizedHost, ":") {
		final = fmt.Sprintf("%s://[%s]", scheme, normalizedHost)
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
		return "", fmt.Errorf("不能包含协议，请只填写 localhost 或 IP")
	}
	if strings.ContainsAny(value, "/?#") {
		return "", fmt.Errorf("不能包含路径或查询参数")
	}
	if strings.Contains(value, ":") && strings.Count(value, ":") == 1 {
		return "", fmt.Errorf("不能包含端口，请在公网端口单独填写")
	}

	host := strings.TrimSpace(value)
	if host == "" {
		return "", fmt.Errorf("缺少 host")
	}
	if strings.ContainsAny(host, " \t\r\n") {
		return "", fmt.Errorf("不能包含空白字符")
	}

	if strings.EqualFold(host, "localhost") {
		return "localhost", nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4.String(), nil
		}
		return "", fmt.Errorf("暂不支持 IPv6，请填写 localhost 或 IPv4")
	}
	if looksLikeIPv4(host) {
		return "", fmt.Errorf("IPv4 地址格式不合法")
	}
	return "", fmt.Errorf("仅支持 localhost 或 IPv4 地址")
}

func BuildPublicURL(host string, port string) string {
	trimmedHost := strings.TrimSpace(host)
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
