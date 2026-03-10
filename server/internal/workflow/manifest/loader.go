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
	manifests, err := ListManifests()
	if err != nil {
		return nil, err
	}

	workflows := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		workflowID := strings.TrimSpace(manifest.WorkflowID)
		if workflowID == "" {
			continue
		}
		workflows = append(workflows, workflowID)
	}

	return workflows, nil
}

func ListManifests() ([]Manifest, error) {
	entries, err := manifestsFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read manifests directory: %w", err)
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
		if err := validateManifest(manifest); err != nil {
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

	filename := manifestFilename(workflowID)
	payload, err := manifestsFS.ReadFile(filename)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest %q: %w", workflowID, err)
	}

	manifest, err := decodeManifest(payload, filename)
	if err != nil {
		return Manifest{}, err
	}
	if err := validateManifest(manifest); err != nil {
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
