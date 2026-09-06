package workflowmanifest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	workflowDefinitionsRootMu sync.RWMutex
	workflowDefinitionsRoot   string
)

// ConfigureWorkflowDefinitionsRoot sets the release input consumed only by
// bootstrap built-in synchronization; normal request paths read PostgreSQL.
func ConfigureWorkflowDefinitionsRoot(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("workflow definitions root is required")
	}
	workflowDefinitionsRootMu.Lock()
	defer workflowDefinitionsRootMu.Unlock()
	workflowDefinitionsRoot = root
	return nil
}

// WorkflowDefinitionsRoot returns the configured runtime workflow-definition root.
func WorkflowDefinitionsRoot() string {
	workflowDefinitionsRootMu.RLock()
	defer workflowDefinitionsRootMu.RUnlock()
	return workflowDefinitionsRoot
}

func manifestFilename(scanWorkflowID string) string {
	return fmt.Sprintf("%s.scan-workflow.json", scanWorkflowID)
}

func ListManifests() ([]Manifest, error) {
	definitions, err := loadWorkflowDefinitions()
	if err != nil {
		return nil, err
	}
	manifests := make([]Manifest, 0, len(definitions))
	for _, definition := range definitions {
		if err := validateManifest(definition); err != nil {
			return nil, fmt.Errorf("validate scan workflow definition %q: %w", definition.ScanWorkflowID, err)
		}
		manifests = append(manifests, definition)
	}
	if err := validateManifestList(manifests); err != nil {
		return nil, err
	}
	if !containsDefaultWorkflow(manifests) {
		return nil, fmt.Errorf("release workflow definitions must include scanWorkflows/default")
	}
	sort.Slice(manifests, func(i, j int) bool { return manifests[i].ScanWorkflowID < manifests[j].ScanWorkflowID })
	return manifests, nil
}

func containsDefaultWorkflow(manifests []Manifest) bool {
	for _, manifest := range manifests {
		if manifest.ScanWorkflowID == "default" {
			return true
		}
	}
	return false
}

func GetManifest(scanWorkflowID string) (Manifest, error) {
	scanWorkflowID = strings.TrimSpace(scanWorkflowID)
	if scanWorkflowID == "" {
		return Manifest{}, fmt.Errorf("scanWorkflowId is required")
	}
	items, err := ListManifests()
	if err != nil {
		return Manifest{}, err
	}
	for _, item := range items {
		if item.ScanWorkflowID == scanWorkflowID {
			return item, nil
		}
	}
	return Manifest{}, fmt.Errorf("scan workflow definition %q not found", scanWorkflowID)
}

func loadWorkflowDefinitions() ([]Manifest, error) {
	root := WorkflowDefinitionsRoot()
	if root == "" {
		return nil, fmt.Errorf("workflow definitions root is required")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read workflow definitions %q: %w", root, err)
	}
	manifests := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".scan-workflow.json") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read workflow definition %q: %w", path, err)
		}
		manifest, err := decodeManifest(payload, path)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}
