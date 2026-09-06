package application

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	maxRepositoryFiles = 200_000
	maxYAMLFiles       = 100_000
	maxYAMLBytes       = int64(512 * 1024 * 1024)
	maxSingleYAMLBytes = int64(16 * 1024 * 1024)
	// These limits are shared with the runner/workspace checks. Git's object
	// store and the materialized checkout are bounded independently because a
	// shallow clone can still contain large compressed objects.
	maxGitTransferBytes  = int64(4 * 1024 * 1024 * 1024)
	maxWorkspaceBytes    = int64(2 * 1024 * 1024 * 1024)
	maxParserMemoryBytes = int64(1 * 1024 * 1024 * 1024)
)

type validatedTemplate struct {
	ID          string
	Name        string
	Severity    string
	Tags        []string
	Author      string
	Description string
	CVE         []string
	CWE         []string
	References  []string
	Remediation string
}

// PostgreSQL text and jsonb values cannot contain a NUL byte. Nuclei templates
// commonly use YAML escapes such as "\\0" in request data or explanatory
// metadata; yaml.v3 decodes those escapes before this projection is persisted.
// Keep the meaning visible and searchable without changing the original YAML.
const persistedNUL = `\u0000`

func normalizePersistedText(value string) string {
	return strings.ReplaceAll(value, "\x00", persistedNUL)
}

func parseNucleiTemplate(raw []byte) (validatedTemplate, error) {
	if len(raw) == 0 || int64(len(raw)) > maxSingleYAMLBytes {
		return validatedTemplate{}, fmt.Errorf("YAML_SIZE_EXCEEDED")
	}
	// Decode exactly one YAML document. A second document would make the
	// persisted content and metadata projection disagree.
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return validatedTemplate{}, fmt.Errorf("YAML_PARSE_ERROR")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return validatedTemplate{}, fmt.Errorf("YAML_MULTIPLE_DOCUMENTS")
		}
		return validatedTemplate{}, fmt.Errorf("YAML_PARSE_ERROR")
	}
	if len(document) == 0 {
		return validatedTemplate{}, fmt.Errorf("TEMPLATE_EMPTY")
	}
	id, ok := document["id"].(string)
	id = strings.TrimSpace(id)
	if !ok || id == "" || len(id) > 255 || strings.ContainsAny(id, " \t\r\n\\/") || strings.ContainsRune(id, '\x00') {
		return validatedTemplate{}, fmt.Errorf("TEMPLATE_ID_INVALID")
	}
	info, ok := document["info"].(map[string]any)
	if !ok {
		return validatedTemplate{}, fmt.Errorf("TEMPLATE_INFO_INVALID")
	}
	name, ok := info["name"].(string)
	name = normalizePersistedText(strings.TrimSpace(name))
	if !ok || name == "" {
		return validatedTemplate{}, fmt.Errorf("TEMPLATE_NAME_INVALID")
	}
	severity, _ := info["severity"].(string)
	severity = strings.ToLower(strings.TrimSpace(severity))
	if severity == "" {
		severity = "info"
	}
	switch severity {
	case "info", "low", "medium", "high", "critical":
	default:
		return validatedTemplate{}, fmt.Errorf("TEMPLATE_SEVERITY_INVALID")
	}
	if !hasProtocol(document) {
		return validatedTemplate{}, fmt.Errorf("TEMPLATE_PROTOCOL_MISSING")
	}
	return validatedTemplate{ID: id, Name: name, Severity: severity, Tags: parseStringList(info["tags"]), Author: parseScalar(info["author"]), Description: parseScalar(info["description"]), CVE: parseClassification(info, "cve-id"), CWE: parseClassification(info, "cwe-id"), References: parseStringList(info["reference"]), Remediation: parseScalar(info["remediation"])}, nil
}

func hasProtocol(document map[string]any) bool {
	for _, key := range []string{"http", "dns", "network", "headless", "code", "ssl", "websocket", "whois", "flow", "workflow", "javascript", "global-matchers"} {
		if _, ok := document[key]; ok {
			return true
		}
	}
	return false
}
func parseScalar(value any) string {
	switch value := value.(type) {
	case string:
		return normalizePersistedText(strings.TrimSpace(value))
	case fmt.Stringer:
		return normalizePersistedText(strings.TrimSpace(value.String()))
	default:
		return ""
	}
}
func parseStringList(value any) []string {
	switch value := value.(type) {
	case string:
		parts := strings.Split(value, ",")
		output := make([]string, 0, len(parts))
		for _, part := range parts {
			if part = strings.TrimSpace(part); part != "" {
				output = append(output, normalizePersistedText(part))
			}
		}
		return output
	case []any:
		output := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				output = append(output, normalizePersistedText(strings.TrimSpace(text)))
			}
		}
		return output
	case []string:
		output := make([]string, 0, len(value))
		for _, item := range value {
			if item = strings.TrimSpace(item); item != "" {
				output = append(output, normalizePersistedText(item))
			}
		}
		return output
	default:
		return []string{}
	}
}
func parseClassification(info map[string]any, key string) []string {
	classification, ok := info["classification"].(map[string]any)
	if !ok {
		return []string{}
	}
	return parseStringList(classification[key])
}
