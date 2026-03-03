package release

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Manifest is the single source of release truth for installer/runtime contracts.
type Manifest struct {
	ReleaseVersion string `yaml:"releaseVersion"`
	AgentVersion   string `yaml:"agentVersion"`
	WorkerVersion  string `yaml:"workerVersion"`
	AgentImageRef  string `yaml:"agentImageRef"`
	WorkerImageRef string `yaml:"workerImageRef"`
}

var schemaVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+([\-+][0-9A-Za-z.+-]+)?$`)

// LoadManifest reads and validates a release manifest.
func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("读取 release manifest 失败: %w", err)
	}

	manifest, err := parseManifest(raw)
	if err != nil {
		return nil, fmt.Errorf("解析 release manifest 失败: %w", err)
	}
	if err := manifest.normalizeAndValidate(); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (manifest *Manifest) normalizeAndValidate() error {
	if manifest == nil {
		return fmt.Errorf("release manifest 不能为空")
	}

	manifest.ReleaseVersion = normalizeVersion(manifest.ReleaseVersion)
	manifest.AgentVersion = normalizeVersion(manifest.AgentVersion)
	manifest.WorkerVersion = normalizeVersion(manifest.WorkerVersion)
	manifest.AgentImageRef = strings.TrimSpace(manifest.AgentImageRef)
	manifest.WorkerImageRef = strings.TrimSpace(manifest.WorkerImageRef)

	if manifest.ReleaseVersion == "" {
		return fmt.Errorf("releaseVersion 不能为空")
	}
	if !isValidSchemaVersion(manifest.ReleaseVersion) {
		return fmt.Errorf("releaseVersion 必须匹配 MAJOR.MINOR.PATCH(+suffix)")
	}
	if manifest.AgentVersion == "" {
		return fmt.Errorf("agentVersion 不能为空")
	}
	if !isValidSchemaVersion(manifest.AgentVersion) {
		return fmt.Errorf("agentVersion 必须匹配 MAJOR.MINOR.PATCH(+suffix)")
	}
	if manifest.WorkerVersion == "" {
		return fmt.Errorf("workerVersion 不能为空")
	}
	if !isValidSchemaVersion(manifest.WorkerVersion) {
		return fmt.Errorf("workerVersion 必须匹配 MAJOR.MINOR.PATCH(+suffix)")
	}
	if manifest.ReleaseVersion != manifest.AgentVersion || manifest.AgentVersion != manifest.WorkerVersion {
		return fmt.Errorf("releaseVersion/agentVersion/workerVersion 必须一致")
	}
	if manifest.AgentImageRef == "" {
		return fmt.Errorf("agentImageRef 不能为空")
	}
	if !isDigestRef(manifest.AgentImageRef) {
		return fmt.Errorf("agentImageRef 必须是 digest 引用（@sha256:...）")
	}
	if manifest.WorkerImageRef == "" {
		return fmt.Errorf("workerImageRef 不能为空")
	}
	if !isDigestRef(manifest.WorkerImageRef) {
		return fmt.Errorf("workerImageRef 必须是 digest 引用（@sha256:...）")
	}
	return nil
}

func isDigestRef(value string) bool {
	ref := strings.TrimSpace(value)
	if ref == "" {
		return false
	}
	at := strings.LastIndex(ref, "@")
	if at <= 0 || at == len(ref)-1 {
		return false
	}
	digest := ref[at+1:]
	if !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	hex := strings.TrimPrefix(digest, "sha256:")
	if len(hex) != 64 {
		return false
	}
	for _, ch := range hex {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') {
			continue
		}
		return false
	}
	return true
}

func parseManifest(raw []byte) (*Manifest, error) {
	result := &Manifest{}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.Contains(trimmed, ":") {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"'")

		switch key {
		case "releaseVersion":
			result.ReleaseVersion = value
		case "agentVersion":
			result.AgentVersion = value
		case "workerVersion":
			result.WorkerVersion = value
		case "agentImageRef":
			result.AgentImageRef = value
		case "workerImageRef":
			result.WorkerImageRef = value
		}
	}
	return result, nil
}

func normalizeVersion(value string) string {
	// Keep release manifest strict: runtime versions must be bare SemVer
	// (MAJOR.MINOR.PATCH(+suffix)) without a leading v/V.
	return strings.TrimSpace(value)
}

func isValidSchemaVersion(value string) bool {
	return schemaVersionPattern.MatchString(normalizeVersion(value))
}
