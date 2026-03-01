package subdomain_discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	schemaOnce sync.Once
	schema     *jsonschema.Schema
	schemaErr  error
)

func resetTestConfigSchemaCache() {
	schemaOnce = sync.Once{}
	schema = nil
	schemaErr = nil
}

func getConfigSchemaForTest() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		b, err := loadGeneratedSchemaBytesForTest()
		if err != nil {
			schemaErr = err
			return
		}

		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("schema.json", strings.NewReader(string(b))); err != nil {
			schemaErr = fmt.Errorf("add schema resource: %w", err)
			return
		}
		s, err := compiler.Compile("schema.json")
		if err != nil {
			schemaErr = fmt.Errorf("compile schema: %w", err)
			return
		}
		schema = s
	})

	return schema, schemaErr
}

// validateExplicitConfig is test-only schema gate validation helper.
func validateExplicitConfig(config map[string]any) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	s, err := getConfigSchemaForTest()
	if err != nil {
		return err
	}

	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("unmarshal config as json: %w", err)
	}

	if err := s.Validate(v); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

func loadGeneratedSchemaBytesForTest() ([]byte, error) {
	file := fmt.Sprintf("generated/%s-%s-%s.schema.json", Name, ContractAPIVersion, ContractSchemaVer)
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read generated schema %s: %w", file, err)
	}
	return b, nil
}

func TestValidateExplicitConfig_TestHelperCompilesSchema(t *testing.T) {
	resetTestConfigSchemaCache()
	_, err := getConfigSchemaForTest()
	if err != nil {
		t.Fatalf("expected test schema helper to compile schema, got: %v", err)
	}
}
