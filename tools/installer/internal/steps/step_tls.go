package steps

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	_ = ctx
	host := resolveTLSCertHost(installer.options.PublicURL)
	fullchain := filepath.Join(sslDirPath, "fullchain.pem")
	privkey := filepath.Join(sslDirPath, "privkey.pem")
	if err := generateSelfSignedCertificateFiles(host, fullchain, privkey); err != nil {
		return fmt.Errorf("证书生成失败: %w", err)
	}
	return nil
}

func (installer *Installer) syncSSLCertToVolume(ctx context.Context, sslDirPath string) error {
	candidates := syncHelperImageCandidates(installer.options.AgentImageRef, installer.options.WorkerImageRef)
	if len(candidates) == 0 {
		return fmt.Errorf("同步 HTTPS 证书到 Docker 卷失败：缺少可用业务镜像，请先完成镜像准备步骤")
	}

	failures := make([]string, 0, len(candidates))
	for _, helperImage := range candidates {
		command := installer.toolchain.DockerCommand(
			"run", "--rm", "--pull=never",
			"-v", fmt.Sprintf("%s:/src:ro", sslDirPath),
			"-v", fmt.Sprintf("%s:/dst", sslVolumeName),
			helperImage,
			"sh", "-c", "cp /src/fullchain.pem /dst/fullchain.pem && cp /src/privkey.pem /dst/privkey.pem && chmod 644 /dst/fullchain.pem && chmod 600 /dst/privkey.pem",
		)
		if _, err := installer.runner.Run(ctx, command); err == nil {
			installer.printer.Info("使用业务镜像同步证书: %s", helperImage)
			return nil
		} else {
			failures = append(failures, fmt.Sprintf("%s => %s", helperImage, commandErrorMessage(err)))
		}
	}
	return fmt.Errorf("同步 HTTPS 证书到 Docker 卷失败：%s", strings.Join(failures, " ; "))
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

func resolveTLSCertHost(publicURL string) string {
	host := "localhost"
	parsed, err := url.Parse(strings.TrimSpace(publicURL))
	if err != nil {
		return host
	}
	hostname := strings.TrimSpace(parsed.Hostname())
	if hostname != "" {
		host = hostname
	}
	return host
}

func syncHelperImageCandidates(agentImageRef, workerImageRef string) []string {
	raw := []string{
		strings.TrimSpace(agentImageRef),
		strings.TrimSpace(workerImageRef),
	}
	result := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func generateSelfSignedCertificateFiles(host, certPath, keyPath string) error {
	dnsNames, ipAddresses := buildSubjectAltEntries(host)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("生成私钥失败: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return fmt.Errorf("生成证书序列号失败: %w", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"LunaFox"},
			CommonName:   host,
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("生成证书失败: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if len(certPEM) == 0 {
		return fmt.Errorf("证书编码失败")
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("写入证书失败: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if len(keyPEM) == 0 {
		return fmt.Errorf("私钥编码失败")
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("写入私钥失败: %w", err)
	}

	return nil
}

func buildSubjectAltEntries(publicHost string) ([]string, []net.IP) {
	hosts := []string{"localhost"}
	ips := []net.IP{net.ParseIP("127.0.0.1")}

	normalized := strings.Trim(strings.TrimSpace(publicHost), "[]")
	if normalized != "" && !strings.EqualFold(normalized, "localhost") {
		if ip := net.ParseIP(normalized); ip != nil {
			ips = appendUniqueIP(ips, ip)
		} else {
			hosts = appendUnique(hosts, normalized)
		}
	}

	return hosts, ips
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

func appendUniqueIP(items []net.IP, value net.IP) []net.IP {
	if value == nil {
		return items
	}
	normalized := value.String()
	for _, item := range items {
		if item != nil && item.String() == normalized {
			return items
		}
	}
	return append(items, value)
}
