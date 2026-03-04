package engineschema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/yyhuni/lunafox/contracts/runtimecontract"
	"gopkg.in/yaml.v3"
)

//go:embed *.schema.json
var schemasFS embed.FS

var (
	schemaMu    sync.Mutex
	schemaCache = map[string]*jsonschema.Schema{}
)

func schemaCacheKey(engine, apiVersion, schemaVersion string) string {
	return engine + "|" + apiVersion + "|" + schemaVersion
}

func schemaFilename(engine, apiVersion, schemaVersion string) string {
	return fmt.Sprintf("%s-%s-%s.schema.json", engine, apiVersion, schemaVersion)
}

func getSchema(engine, apiVersion, schemaVersion string) (*jsonschema.Schema, error) {
	if engine == "" {
		return nil, fmt.Errorf("engine is required")
	}
	if strings.TrimSpace(apiVersion) == "" {
		return nil, fmt.Errorf("apiVersion is required")
	}
	if strings.TrimSpace(schemaVersion) == "" {
		return nil, fmt.Errorf("schemaVersion is required")
	}

	cacheKey := schemaCacheKey(engine, apiVersion, schemaVersion)
	schemaMu.Lock()
	cached, ok := schemaCache[cacheKey]
	schemaMu.Unlock()
	if ok {
		return cached, nil
	}

	filename := schemaFilename(engine, apiVersion, schemaVersion)
	b, err := schemasFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read schema %q: %w", filename, err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(filename, bytes.NewReader(b)); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	compiled, err := compiler.Compile(filename)
	if err != nil {
		return nil, fmt.Errorf("compile schema %q: %w", filename, err)
	}

	schemaMu.Lock()
	schemaCache[cacheKey] = compiled
	schemaMu.Unlock()

	return compiled, nil
}

// Validate validates a config map (already extracted for the engine) against the engine schema.
func Validate(engine string, config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	apiVersion, schemaVersion, err := extractSchemaVersion(config)
	if err != nil {
		return err
	}

	s, err := getSchema(engine, apiVersion, schemaVersion)
	if err != nil {
		return err
	}

	// Convert YAML-decoded types (e.g., int) into JSON-native types (e.g., float64)
	// to match JSON Schema validators' expectations.
	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("unmarshal config as json: %w", err)
	}

	if err := s.Validate(v); err != nil {
		return fmt.Errorf("invalid %s config: %w", engine, err)
	}

	return nil
}

func extractSchemaVersion(config map[string]any) (apiVersion, schemaVersion string, err error) {
	rawAPIVersion, ok := config["apiVersion"]
	if !ok {
		return "", "", fmt.Errorf("apiVersion is required")
	}
	apiVersion, ok = rawAPIVersion.(string)
	if !ok || strings.TrimSpace(apiVersion) == "" {
		return "", "", fmt.Errorf("apiVersion must be non-empty string")
	}
	if !runtimecontract.IsValidAPIVersion(apiVersion) {
		return "", "", fmt.Errorf("%s", runtimecontract.APIVersionFieldMessage("apiVersion"))
	}

	rawSchemaVersion, ok := config["schemaVersion"]
	if !ok {
		return "", "", fmt.Errorf("schemaVersion is required")
	}
	schemaVersion, ok = rawSchemaVersion.(string)
	if !ok || strings.TrimSpace(schemaVersion) == "" {
		return "", "", fmt.Errorf("schemaVersion must be non-empty string")
	}
	if !runtimecontract.IsValidSchemaVersion(schemaVersion) {
		return "", "", fmt.Errorf("%s", runtimecontract.SchemaVersionFieldMessage("schemaVersion"))
	}

	return strings.TrimSpace(apiVersion), strings.TrimSpace(schemaVersion), nil
}

// ValidateYAML validates a YAML config blob against the engine schema.
// If the YAML is nested under the engine name (e.g. {subdomain_discovery: {...}}),
// the nested object will be validated; otherwise the top-level mapping is validated.
func ValidateYAML(engine string, yamlBytes []byte) error {
	if len(yamlBytes) == 0 {
		return fmt.Errorf("yaml is required")
	}

	var root map[string]any
	if err := yaml.Unmarshal(yamlBytes, &root); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	if root == nil {
		return fmt.Errorf("yaml must be a mapping")
	}

	config := root
	if nested, ok := root[engine]; ok {
		if m, ok := nested.(map[string]any); ok {
			config = m
		}
	}

	return Validate(engine, config)
}

// ListEngines returns all available engine names by scanning embedded schema files.
func ListEngines() ([]string, error) {
	entries, err := schemasFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read schemas directory: %w", err)
	}

	var engines []string
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".schema.json") {
			continue
		}
		engine, metaErr := readEngineNameFromSchema(name)
		if metaErr != nil || strings.TrimSpace(engine) == "" {
			continue
		}
		engine = strings.TrimSpace(engine)
		if _, exists := seen[engine]; exists {
			continue
		}
		seen[engine] = struct{}{}
		engines = append(engines, engine)
	}

	return engines, nil
}

func readEngineNameFromSchema(filename string) (string, error) {
	payload, err := schemasFS.ReadFile(filename)
	if err != nil {
		return "", err
	}
	var meta struct {
		Engine string `json:"x-engine"`
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return "", err
	}
	return strings.TrimSpace(meta.Engine), nil
}
