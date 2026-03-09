package workflowmanifest

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

//go:embed *.manifest.json
var manifestsFS embed.FS

func manifestFilename(workflowID string) string {
	return fmt.Sprintf("%s.manifest.json", workflowID)
}

func ListWorkflows() ([]string, error) {
	items, err := ListWorkflowMetadata()
	if err != nil {
		return nil, err
	}

	workflows := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.WorkflowID) == "" {
			continue
		}
		workflows = append(workflows, item.WorkflowID)
	}

	return workflows, nil
}

func ListWorkflowMetadata() ([]WorkflowMetadata, error) {
	manifests, err := ListManifests()
	if err != nil {
		return nil, err
	}

	metadata := make([]WorkflowMetadata, 0, len(manifests))
	for _, manifest := range manifests {
		metadata = append(metadata, WorkflowMetadata{
			WorkflowID:  manifest.WorkflowID,
			DisplayName: manifest.DisplayName,
			Description: manifest.Description,
		})
	}

	return metadata, nil
}

func ListManifests() ([]Manifest, error) {
	entries, err := manifestsFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read manifests directory: %w", err)
	}

	knownProfileIDs, err := loadKnownProfileIDs()
	if err != nil {
		return nil, err
	}

	manifests := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".manifest.json") {
			continue
		}

		payload, err := manifestsFS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read manifest %q: %w", name, err)
		}
		manifest, err := decodeManifest(payload, name)
		if err != nil {
			return nil, err
		}
		if err := validateManifest(manifest, knownProfileIDs); err != nil {
			return nil, fmt.Errorf("validate manifest %q: %w", name, err)
		}
		manifests = append(manifests, manifest)
	}

	if err := validateManifestList(manifests); err != nil {
		return nil, err
	}

	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].WorkflowID < manifests[j].WorkflowID
	})

	return manifests, nil
}

func GetManifest(workflowID string) (Manifest, error) {
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return Manifest{}, fmt.Errorf("workflowId is required")
	}

	knownProfileIDs, err := loadKnownProfileIDs()
	if err != nil {
		return Manifest{}, err
	}

	filename := manifestFilename(workflowID)
	payload, err := manifestsFS.ReadFile(filename)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %q: %w", workflowID, err)
	}

	manifest, err := decodeManifest(payload, filename)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateManifest(manifest, knownProfileIDs); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func decodeManifest(payload []byte, source string) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest %q: %w", source, err)
	}
	if err := consumeJSONEOF(decoder); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest %q: %w", source, err)
	}

	return manifest, nil
}

func consumeJSONEOF(decoder *json.Decoder) error {
	if decoder == nil {
		return nil
	}
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing JSON content")
}
