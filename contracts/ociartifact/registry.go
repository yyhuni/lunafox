package ociartifact

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidateRegistryAuthority accepts the host[:port] authority used by OCI
// clients. It excludes schemes, paths, credentials, and empty values.
func ValidateRegistryAuthority(value string) error {
	if value == "" || value != strings.TrimSpace(value) {
		return fmt.Errorf("registry authority must be a non-empty canonical value")
	}
	if strings.ContainsAny(value, "/?#@") || strings.Contains(value, "://") {
		return fmt.Errorf("registry authority %q must be host[:port] without scheme or path", value)
	}
	parsed, err := ParseDigestReference(value + "/repository@sha256:" + strings.Repeat("a", 64))
	if err != nil || parsed.Registry != value || parsed.Repository != "repository" {
		return fmt.Errorf("invalid Registry authority %q", value)
	}
	host := value
	if separator := strings.LastIndexByte(value, ':'); separator >= 0 {
		host = value[:separator]
		port, err := strconv.Atoi(value[separator+1:])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid Registry port in %q", value)
		}
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("invalid Registry host in %q", value)
		}
	}
	return nil
}

// ReplaceRegistry preserves repository and digest identity while changing only
// the network authority used for a request.
func ReplaceRegistry(reference string, registry string) (string, error) {
	if err := ValidateRegistryAuthority(registry); err != nil {
		return "", err
	}
	parsed, err := ParseDigestReference(reference)
	if err == nil && parsed.String() != reference {
		return "", fmt.Errorf("OCI digest reference must be canonical")
	}
	if err != nil {
		return "", err
	}
	return registry + "/" + parsed.Repository + "@" + parsed.Digest, nil
}

// RegistryCandidates rewrites transport addresses without altering repositories
// or digests. Equal mirror addresses are attempted once, with no source fallback.
func RegistryCandidates(refs []string, registry string) ([]string, error) {
	if err := ValidateRegistryAuthority(registry); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, fmt.Errorf("OCI digest references are required")
	}
	result := make([]string, 0, len(refs))
	seen := make(map[string]bool)
	for _, ref := range refs {
		mapped, err := ReplaceRegistry(ref, registry)
		if err != nil {
			return nil, err
		}
		if !seen[mapped] {
			result = append(result, mapped)
			seen[mapped] = true
		}
	}
	return result, nil
}
