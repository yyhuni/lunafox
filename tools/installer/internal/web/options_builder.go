package web

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

var domainLabelPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

func buildInstallOptions(base cli.Options, req startRequest) (cli.Options, error) {
	publicHostInput := strings.TrimSpace(req.PublicHost)
	publicPort := strings.TrimSpace(req.PublicPort)

	opts := base
	opts.UseGoProxyCN = req.UseGoProxyCN
	if req.UseGoProxyCN {
		opts.GoProxy = "https://goproxy.cn,direct"
	} else {
		opts.GoProxy = "https://proxy.golang.org,direct"
	}

	hostFromInput, err := parsePublicHostInput(publicHostInput)
	if err != nil {
		return cli.Options{}, fmt.Errorf("公网主机不合法: %w", err)
	}
	if publicPort == "" {
		return cli.Options{}, fmt.Errorf("必须填写公网端口")
	}
	if err := validatePort(publicPort); err != nil {
		return cli.Options{}, fmt.Errorf("公网端口不合法: %w", err)
	}

	if hostFromInput == "" {
		return cli.Options{}, fmt.Errorf("必须填写公网主机（IP 或域名）")
	}
	publicURL := buildPublicURL(hostFromInput, publicPort)

	if err := validatePublicURL(publicURL); err != nil {
		return cli.Options{}, fmt.Errorf("PUBLIC_URL 不合法: %w", err)
	}
	opts.PublicURL = publicURL
	opts.PublicPort = publicPort
	opts.AgentServerURL = pick(base.AgentServerURL, cli.DefaultAgentServerURL)
	opts.AgentNetwork = pick(base.AgentNetwork, cli.DefaultAgentNetwork)
	opts.ComposeFile = opts.ComposeProd
	if opts.Mode == cli.ModeDev {
		opts.ComposeFile = opts.ComposeDev
	}
	return opts, nil
}

func validatePublicURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme 仅支持 http 或 https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("缺少 host")
	}
	return nil
}

func validatePort(raw string) error {
	value := strings.TrimSpace(raw)
	port, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("必须是数字")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("必须在 1-65535 之间")
	}
	return nil
}

func parsePublicHostInput(raw string) (string, error) {
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

func parseHostPortFromURL(raw string) (string, string) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return "", ""
	}
	return strings.Trim(strings.TrimSpace(parsed.Hostname()), "[]"), strings.TrimSpace(parsed.Port())
}

func buildPublicURL(host string, port string) string {
	trimmedHost := strings.Trim(strings.TrimSpace(host), "[]")
	hostWithPort := net.JoinHostPort(trimmedHost, strings.TrimSpace(port))
	return "https://" + hostWithPort
}

func pick(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
