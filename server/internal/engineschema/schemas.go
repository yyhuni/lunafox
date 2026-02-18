package engineschema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed *.schema.json
var schemasFS embed.FS

var (
	schemaMu    sync.Mutex
	schemaCache = map[string]*jsonschema.Schema{}
)

func getSchema(engine string) (*jsonschema.Schema, error) {
	if engine == "" {
		return nil, fmt.Errorf("engine is required")
	}

	schemaMu.Lock()
	cached, ok := schemaCache[engine]
	schemaMu.Unlock()
	if ok {
		return cached, nil
	}

	filename := engine + ".schema.json"
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
	schemaCache[engine] = compiled
	schemaMu.Unlock()

	return compiled, nil
}

// Validate validates a config map (already extracted for the engine) against the engine schema.
func Validate(engine string, config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	s, err := getSchema(engine)
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
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Extract engine name from filename: "subdomain_discovery.schema.json" -> "subdomain_discovery"
		if strings.HasSuffix(name, ".schema.json") {
			engine := strings.TrimSuffix(name, ".schema.json")
			engines = append(engines, engine)
		}
	}

	return engines, nil
}
