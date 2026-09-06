package application

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/domain"
)

const officialNucleiTemplatesURL = "https://github.com/projectdiscovery/nuclei-templates.git"

// OfficialNucleiTemplatesURL returns the default public Nuclei templates URL.
func OfficialNucleiTemplatesURL() string { return officialNucleiTemplatesURL }

// ValidateSourceURL performs only deterministic URL-shape validation. Network
// resolution and the actual egress path belong to Git and its configured
// transport, which may be a proxy or a private network route.
func ValidateSourceURL(sourceType domain.SourceType, rawURL string) (string, error) {
	if !sourceType.Valid() {
		return "", fmt.Errorf("%w: unsupported source type", ErrInvalidArgument)
	}
	value := strings.TrimSpace(rawURL)
	if value == "" || strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return "", fmt.Errorf("%w: repository URL is required", ErrInvalidSourceURL)
	}
	if strings.ContainsAny(value, "\\\x00\r\n\t") {
		return "", fmt.Errorf("%w: repository URL contains unsupported characters", ErrInvalidSourceURL)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("%w: only anonymous HTTPS repository URLs are supported", ErrInvalidSourceURL)
	}
	port := parsed.Port()
	if strings.HasSuffix(parsed.Host, ":") {
		return "", fmt.Errorf("%w: repository port is invalid", ErrInvalidSourceURL)
	}
	if port != "" {
		portNumber, parseErr := strconv.ParseUint(port, 10, 16)
		if parseErr != nil || portNumber == 0 {
			return "", fmt.Errorf("%w: repository port is invalid", ErrInvalidSourceURL)
		}
		port = strconv.FormatUint(portNumber, 10)
		if port == "443" {
			port = ""
		}
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: query and fragment are not allowed", ErrInvalidSourceURL)
	}
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(parsed.Hostname())), ".")
	isIP := net.ParseIP(host) != nil
	if host == "" || strings.Contains(host, "%") || (!isIP && !isStrictHostname(host)) || (strings.HasPrefix(parsed.Host, "[") && !isIP) {
		return "", fmt.Errorf("%w: repository host is invalid", ErrInvalidSourceURL)
	}
	if sourceType == domain.SourceTypeGitee && host != "gitee.com" {
		return "", fmt.Errorf("%w: Gitee source must use gitee.com", ErrInvalidSourceURL)
	}
	if parsed.Path == "" || parsed.Path == "/" || strings.Contains(parsed.Path, "\\") {
		return "", fmt.Errorf("%w: repository path is required", ErrInvalidSourceURL)
	}
	decodedPath, decodeErr := url.PathUnescape(parsed.EscapedPath())
	if decodeErr != nil {
		return "", fmt.Errorf("%w: repository path encoding is invalid", ErrInvalidSourceURL)
	}
	if strings.Contains(decodedPath, "\\") {
		return "", fmt.Errorf("%w: repository path contains unsupported characters", ErrInvalidSourceURL)
	}
	for _, segment := range strings.Split(decodedPath, "/") {
		if segment == ".." || segment == "." {
			return "", fmt.Errorf("%w: repository path traversal is not allowed", ErrInvalidSourceURL)
		}
	}

	parsed.Scheme = "https"
	parsed.Host = normalizedHost(host, port)
	return parsed.String(), nil
}

func normalizedHost(host, port string) string {
	if strings.Contains(host, ":") {
		if port == "" {
			return "[" + host + "]"
		}
		return net.JoinHostPort(host, port)
	}
	if port == "" {
		return host
	}
	return host + ":" + port
}

func isStrictHostname(host string) bool {
	if len(host) > 253 || !isASCII(host) {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}

func isASCII(value string) bool {
	for _, character := range value {
		if character > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// PublicSourceResolver is kept as the application resolver boundary so the
// runner remains injectable in tests. It deliberately performs no network I/O.
type PublicSourceResolver struct{}

// NewPublicSourceResolver constructs the proxy-compatible source resolver.
func NewPublicSourceResolver() *PublicSourceResolver { return &PublicSourceResolver{} }

// Validate normalizes a source URL without resolving or probing its host.
func (*PublicSourceResolver) Validate(_ context.Context, sourceType domain.SourceType, rawURL string) (string, error) {
	return ValidateSourceURL(sourceType, rawURL)
}
