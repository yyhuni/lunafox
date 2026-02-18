package steps

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
)

type stepHealth struct{}

func (stepHealth) Title() string {
	return "等待服务就绪"
}

func (stepHealth) Run(ctx context.Context, installer *Installer) error {
	if installer.tlsConfig == nil {
		return fmt.Errorf("HTTPS 信任链未初始化，请检查证书配置")
	}

	httpClient := newHTTPClient(installer.tlsConfig, 6*time.Second)
	backendURL := strings.TrimRight(installer.options.PublicURL, "/") + "/health"
	frontendURL := strings.TrimRight(installer.options.PublicURL, "/") + "/"

	for attempt := 1; attempt <= 120; attempt++ {
		if checkURLReady(httpClient, backendURL) && checkURLReady(httpClient, frontendURL) {
			if err := installer.checkContainerNetworkReachability(ctx); err != nil {
				return err
			}
			installer.printer.Success("服务已就绪")
			if err := installer.prewarmFrontend(ctx); err != nil {
				return err
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("服务启动超时，请检查日志")
}

func (installer *Installer) checkContainerNetworkReachability(ctx context.Context) error {
	networkName := strings.TrimSpace(installer.options.AgentNetwork)
	if networkName == "" || networkName == "off" || networkName == "none" {
		return fmt.Errorf("Agent 网络配置无效，无法执行容器视角可达性检查")
	}

	targetHost, targetPort, err := parseHostPortForProbe(installer.options.PublicURL)
	if err != nil {
		return err
	}

	target := net.JoinHostPort(targetHost, targetPort)
	healthURL := strings.TrimRight(installer.options.PublicURL, "/") + "/health"
	caPath := filepath.Join(sslDir(installer.options.DockerDir), "fullchain.pem")
	if !fileExists(caPath) {
		return fmt.Errorf("缺少 HTTPS 证书，无法执行容器视角可达性检查: %s", caPath)
	}

	installer.printer.Info("容器网络可达性检查: %s", healthURL)
	command := installer.toolchain.DockerCommand(
		"run", "--rm",
		"--network", networkName,
		"-e", "LUNAFOX_TLS_TARGET="+target,
		"-e", "LUNAFOX_TLS_SERVER_NAME="+targetHost,
		"-v", fmt.Sprintf("%s:/ca/fullchain.pem:ro", caPath),
		"alpine/openssl",
		"sh", "-ec",
		`printf "GET /health HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n" "$LUNAFOX_TLS_SERVER_NAME" | openssl s_client -verify_return_error -CAfile /ca/fullchain.pem -connect "$LUNAFOX_TLS_TARGET" -servername "$LUNAFOX_TLS_SERVER_NAME" 2>/dev/null | grep -Eq "HTTP/1\.[01] [23][0-9]{2}"`,
	)

	if _, err := installer.runner.Run(ctx, command); err != nil {
		detail := ""
		var execErr *execx.ExecError
		if errors.As(err, &execErr) {
			detail = trimCommandErrorDetail(execErr.Result.Stderr, execErr.Result.Stdout, 260)
		}
		if detail != "" {
			return fmt.Errorf("容器网络可达性检查失败：Agent/Worker 可能无法访问 %s（%s）", healthURL, detail)
		}
		return fmt.Errorf("容器网络可达性检查失败：Agent/Worker 可能无法访问 %s，请确认公网主机、端口和 DNS", healthURL)
	}

	installer.printer.Success("容器网络可达性检查通过")
	return nil
}

func parseHostPortForProbe(publicURL string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(publicURL))
	if err != nil {
		return "", "", fmt.Errorf("PUBLIC_URL 解析失败: %w", err)
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", "", fmt.Errorf("PUBLIC_URL 缺少 host")
	}
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		switch strings.ToLower(strings.TrimSpace(parsed.Scheme)) {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return "", "", fmt.Errorf("PUBLIC_URL 缺少端口且 scheme 不支持默认端口: %s", parsed.Scheme)
		}
	}
	return host, port, nil
}

func trimCommandErrorDetail(stderr string, stdout string, maxLen int) string {
	detail := strings.TrimSpace(stderr)
	if detail == "" {
		detail = strings.TrimSpace(stdout)
	}
	if detail == "" {
		return ""
	}
	if len(detail) <= maxLen {
		return detail
	}
	return detail[:maxLen] + "..."
}

func (installer *Installer) prewarmFrontend(ctx context.Context) error {
	if installer.options.Mode != cli.ModeDev {
		return nil
	}

	httpClient := newHTTPClient(installer.tlsConfig, 6*time.Second)
	paths := []string{"/zh/login", "/zh/dashboard/"}
	baseURL := strings.TrimRight(installer.options.PublicURL, "/")

	for _, path := range paths {
		targetURL := baseURL + path
		for attempt := 1; attempt <= 30; attempt++ {
			if checkURLWarm(httpClient, targetURL) {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
	}
	return nil
}
