package steps

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/execx"
)

type stepTLS struct{}

func (stepTLS) Title() string {
	return "生成 HTTPS 证书"
}

func (stepTLS) Run(ctx context.Context, installer *Installer) error {
	sslDirPath := sslDir(installer.options.DockerDir)
	fullchain := filepath.Join(sslDirPath, "fullchain.pem")
	privkey := filepath.Join(sslDirPath, "privkey.pem")

	if fileExists(fullchain) && fileExists(privkey) {
		installer.printer.Info("检测到已有 HTTPS 证书，跳过生成")
	} else {
		if err := os.MkdirAll(sslDirPath, 0o755); err != nil {
			return fmt.Errorf("创建证书目录失败: %w", err)
		}
		if err := installer.generateSSLCert(ctx, sslDirPath); err != nil {
			return err
		}
	}

	if !fileExists(fullchain) || !fileExists(privkey) {
		return fmt.Errorf("证书生成失败，请手动放置证书到 %s", sslDirPath)
	}
	if err := installer.syncSSLCertToVolume(ctx, sslDirPath); err != nil {
		return err
	}

	config, err := loadTLSConfigFromCA(fullchain)
	if err != nil {
		return err
	}
	installer.tlsConfig = config
	installer.printer.Success("证书已就绪")
	return nil
}

func (installer *Installer) generateSSLCert(ctx context.Context, sslDirPath string) error {
	host := "localhost"
	if parsed, err := url.Parse(strings.TrimSpace(installer.options.PublicURL)); err == nil && strings.TrimSpace(parsed.Hostname()) != "" {
		host = strings.TrimSpace(parsed.Hostname())
	}
	subjectAltName := buildSubjectAltName(host)

	command := installer.toolchain.DockerCommand(
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/ssl", sslDirPath),
		"alpine/openssl",
		"req", "-x509", "-nodes", "-newkey", "rsa:2048", "-days", "365",
		"-keyout", "/ssl/privkey.pem",
		"-out", "/ssl/fullchain.pem",
		"-subj", fmt.Sprintf("/C=CN/ST=NA/L=NA/O=LunaFox/CN=%s", host),
		"-addext", subjectAltName,
	)
	if _, err := installer.runner.Run(ctx, command); err != nil {
		return wrapCommandFailure(err, "证书生成失败，请检查 Docker 与 openssl 镜像是否可用", 260)
	}
	return nil
}

func (installer *Installer) syncSSLCertToVolume(ctx context.Context, sslDirPath string) error {
	command := installer.toolchain.DockerCommand(
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/src:ro", sslDirPath),
		"-v", fmt.Sprintf("%s:/dst", sslVolumeName),
		"alpine",
		"sh", "-c", "cp /src/fullchain.pem /dst/fullchain.pem && cp /src/privkey.pem /dst/privkey.pem && chmod 644 /dst/fullchain.pem && chmod 600 /dst/privkey.pem",
	)
	if _, err := installer.runner.Run(ctx, command); err != nil {
		return wrapCommandFailure(err, "同步 HTTPS 证书到 Docker 卷失败", 260)
	}
	return nil
}

func wrapCommandFailure(err error, message string, maxDetail int) error {
	var execErr *execx.ExecError
	if errors.As(err, &execErr) {
		detail := trimCommandErrorDetail(execErr.Result.Stderr, execErr.Result.Stdout, maxDetail)
		if detail != "" {
			return fmt.Errorf("%s（%s）", message, detail)
		}
	}
	return fmt.Errorf("%s: %w", message, err)
}

func loadTLSConfigFromCA(caPath string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("读取 HTTPS 证书失败: %w", err)
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caPEM); !ok {
		return nil, fmt.Errorf("加载 HTTPS 证书失败，请检查 %s", caPath)
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	}, nil
}

func buildSubjectAltName(publicHost string) string {
	hosts := []string{"localhost"}
	ips := []string{"127.0.0.1"}
	normalized := strings.Trim(strings.TrimSpace(publicHost), "[]")
	if normalized != "" && !strings.EqualFold(normalized, "localhost") {
		if ip := net.ParseIP(normalized); ip != nil {
			ips = appendUnique(ips, ip.String())
		} else {
			hosts = appendUnique(hosts, normalized)
		}
	}

	sanParts := make([]string, 0, len(hosts)+len(ips))
	for _, host := range hosts {
		sanParts = append(sanParts, "DNS:"+host)
	}
	for _, ip := range ips {
		sanParts = append(sanParts, "IP:"+ip)
	}
	return "subjectAltName=" + strings.Join(sanParts, ",")
}

func appendUnique(items []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return items
	}
	for _, item := range items {
		if strings.EqualFold(item, trimmed) {
			return items
		}
	}
	return append(items, trimmed)
}
