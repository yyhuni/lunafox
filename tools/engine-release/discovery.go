package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	enginemanifest "github.com/yyhuni/lunafox/contracts/enginemanifest"
	"github.com/yyhuni/lunafox/contracts/enginemanifest/repositoryname"
)

const discoverySchemaVersion = "lunafox.engine-release-discovery.v1"

// Discovery is an ephemeral build input. It is generated from source
// engine.json files and Dockerfiles; it is never packaged or consumed by the
// Server runtime. Registry/image identity is intentionally absent from the
// source manifest and is derived only by the publisher after discovery.
type Discovery struct {
	SchemaVersion string         `json:"schemaVersion"`
	EngineRoot    string         `json:"engineRoot"`
	Engines       []EngineSource `json:"engines"`
}

type EngineSource struct {
	EngineID     string `json:"engineId"`
	Directory    string `json:"directory"`
	Dockerfile   string `json:"dockerfile"`
	BuildContext string `json:"buildContext"`
	Repository   string `json:"repository"`
}

func discoverEngineSources(root string) (Discovery, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Discovery{}, fmt.Errorf("engine definition root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return Discovery{}, fmt.Errorf("resolve engine definition root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Discovery{}, fmt.Errorf("inspect engine definition root %q: %w", root, err)
	}
	if !info.IsDir() {
		return Discovery{}, fmt.Errorf("engine definition root %q must be a directory", root)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return Discovery{}, fmt.Errorf("read engine definition root %q: %w", root, err)
	}
	sources := make([]EngineSource, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		definitionPath := filepath.Join(root, entry.Name(), "engine.json")
		if _, statErr := os.Lstat(definitionPath); statErr != nil {
			if !os.IsNotExist(statErr) {
				return Discovery{}, fmt.Errorf("inspect engine definition %q: %w", definitionPath, statErr)
			}
			// extensions/engines/container and similar support directories are
			// not Engine sources. A directory that advertises a Dockerfile but
			// omits engine.json is an incomplete source and must fail closed.
			if _, dockerfileErr := os.Stat(filepath.Join(root, entry.Name(), "Dockerfile")); dockerfileErr == nil {
				return Discovery{}, fmt.Errorf("engine directory %q has a Dockerfile but no engine.json", entry.Name())
			}
			continue
		}
		source, err := discoverOneEngine(root, entry.Name())
		if err != nil {
			return Discovery{}, err
		}
		sources = append(sources, source)
	}
	if len(sources) == 0 {
		return Discovery{}, fmt.Errorf("engine definition root %q contains no engine directories", root)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].EngineID < sources[j].EngineID })
	seenIDs := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if _, exists := seenIDs[source.EngineID]; exists {
			return Discovery{}, fmt.Errorf("duplicate discovered engineId %q", source.EngineID)
		}
		seenIDs[source.EngineID] = struct{}{}
	}
	return Discovery{SchemaVersion: discoverySchemaVersion, EngineRoot: root, Engines: sources}, nil
}

func discoverOneEngine(root, directory string) (EngineSource, error) {
	if directory == "" || directory == "." || directory != filepath.Base(directory) {
		return EngineSource{}, fmt.Errorf("invalid engine directory %q", directory)
	}
	engineDir := filepath.Join(root, directory)
	engineJSONPath := filepath.Join(engineDir, "engine.json")
	payload, err := readRegularFile(engineJSONPath)
	if err != nil {
		return EngineSource{}, fmt.Errorf("read %s: %w", filepath.ToSlash(filepath.Join(directory, "engine.json")), err)
	}
	definition, err := enginemanifest.DecodeEngineDefinition(payload, filepath.ToSlash(filepath.Join(directory, "engine.json")))
	if err != nil {
		return EngineSource{}, fmt.Errorf("validate %s: %w", filepath.ToSlash(filepath.Join(directory, "engine.json")), err)
	}
	localName := strings.TrimPrefix(definition.EngineID, "engine.lunafox.")
	if localName == definition.EngineID || localName != directory {
		return EngineSource{}, fmt.Errorf("engineId %q must match source directory %q", definition.EngineID, directory)
	}
	repository, err := repositoryname.FirstPartyRuntimeImageRepositoryName(definition.EngineID)
	if err != nil {
		return EngineSource{}, fmt.Errorf("derive Runtime Image repository for %q: %w", definition.EngineID, err)
	}
	dockerfile := filepath.Join(engineDir, "Dockerfile")
	dockerInfo, err := os.Lstat(dockerfile)
	if err != nil {
		return EngineSource{}, fmt.Errorf("engine %q Dockerfile is required: %w", definition.EngineID, err)
	}
	if dockerInfo.Mode()&fs.ModeSymlink != 0 || !dockerInfo.Mode().IsRegular() {
		return EngineSource{}, fmt.Errorf("engine %q Dockerfile must be a regular non-symlink file", definition.EngineID)
	}
	return EngineSource{
		EngineID:   definition.EngineID,
		Directory:  directory,
		Dockerfile: filepath.ToSlash(filepath.Join(directory, "Dockerfile")),
		// Builtin Dockerfiles share the extensions/engines Go module, so their
		// canonical author-owned context is the definition root. The Dockerfile
		// path still selects the individual Engine's build definition.
		BuildContext: ".",
		Repository:   repository,
	}, nil
}

func encodeDiscovery(discovery Discovery) ([]byte, error) {
	if discovery.SchemaVersion == "" {
		discovery.SchemaVersion = discoverySchemaVersion
	}
	if discovery.SchemaVersion != discoverySchemaVersion {
		return nil, fmt.Errorf("unsupported discovery schemaVersion %q", discovery.SchemaVersion)
	}
	return json.MarshalIndent(discovery, "", "  ")
}
