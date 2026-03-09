package workflowschema

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed *.schema.json
var schemasFS embed.FS

var (
	schemaMu    sync.Mutex
	schemaCache = map[string]*jsonschema.Schema{}
)

func getSchema(workflowID string) (*jsonschema.Schema, error) {
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return nil, fmt.Errorf("workflow is required")
	}

	cacheKey := schemaCacheKey(workflowID)
	schemaMu.Lock()
	cached, ok := schemaCache[cacheKey]
	schemaMu.Unlock()
	if ok {
		return cached, nil
	}

	filename := schemaFilename(workflowID)
	payload, err := schemasFS.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read schema %q: %w", filename, err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(filename, bytes.NewReader(payload)); err != nil {
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

func ListWorkflows() ([]string, error) {
	entries, err := schemasFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read schemas directory: %w", err)
	}

	workflows := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".schema.json") {
			continue
		}

		workflowID, err := workflowIDFromSchemaFilename(name)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[workflowID]; exists {
			return nil, fmt.Errorf("duplicate schema for workflow %q", workflowID)
		}
		seen[workflowID] = struct{}{}
		workflows = append(workflows, workflowID)
	}

	sort.Strings(workflows)
	return workflows, nil
}
