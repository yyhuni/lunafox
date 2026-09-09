package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	command := flag.String("command", "", "discover, validate-build-results, build-packages, validate-package-build-results, or validate-package-artifacts")
	engineRoot := flag.String("engine-root", "./extensions/engines", "builtin engine source root")
	engineID := flag.String("engine-id", "", "optional canonical Engine selection for discover and validate-build-results")
	resultsPath := flag.String("build-results", "", "verified runtime image build results JSON")
	packageResultsPath := flag.String("package-build-results", "", "Engine Package build results JSON")
	previousPackageResultsPath := flag.String("previous-package-build-results", "", "optional previous Engine Package build results JSON")
	engineVersion := flag.String("engine-version", "", "Engine Package v2 version")
	outRoot := flag.String("out-root", "./dist/engine-packages", "expanded/archive package output root")
	mode := flag.String("mode", "", "expected build result mode: development or production")
	output := flag.String("output", "", "optional JSON output path; stdout by default")
	flag.Parse()

	if err := runCommand(*command, *engineRoot, *engineID, *resultsPath, *packageResultsPath, *previousPackageResultsPath, *engineVersion, *outRoot, *mode, *output, os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func runCommand(command, engineRoot, selectedEngineID, resultsPath, packageResultsPath, previousPackageResultsPath, version, outRoot, expectedMode, outputPath string, stdout io.Writer) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("-command is required")
	}
	switch command {
	case "discover":
		discovery, err := discoverEngineSources(engineRoot)
		if err != nil {
			return err
		}
		discovery, err = selectEngineDiscovery(discovery, selectedEngineID)
		if err != nil {
			return err
		}
		payload, err := encodeDiscovery(discovery)
		if err != nil {
			return err
		}
		return writeCommandOutput(outputPath, payload, stdout)
	case "validate-package-build-results":
		if err := rejectSelectedEngineForPackageCommand(command, selectedEngineID); err != nil {
			return err
		}
		packageResults, err := readPackageBuildResults(packageResultsPath)
		if err != nil {
			return err
		}
		if err := validatePackageBuildResultsShape(packageResults, strings.TrimSpace(expectedMode)); err != nil {
			return err
		}
		if strings.TrimSpace(previousPackageResultsPath) != "" {
			previous, err := readPackageBuildResults(previousPackageResultsPath)
			if err != nil {
				return err
			}
			if err := validatePackageReleaseEvolution(previous, packageResults); err != nil {
				return err
			}
		}
		payload, err := json.MarshalIndent(packageResults, "", "  ")
		if err != nil {
			return err
		}
		return writeCommandOutput(outputPath, append(payload, '\n'), stdout)
	case "validate-build-results", "build-packages", "validate-package-artifacts":
		if command != "validate-build-results" {
			if err := rejectSelectedEngineForPackageCommand(command, selectedEngineID); err != nil {
				return err
			}
		}
		discovery, err := discoverEngineSources(engineRoot)
		if err != nil {
			return err
		}
		if command == "validate-build-results" {
			discovery, err = selectEngineDiscovery(discovery, selectedEngineID)
			if err != nil {
				return err
			}
		}
		if strings.TrimSpace(resultsPath) == "" {
			return fmt.Errorf("-build-results is required")
		}
		resultsPayload, err := readRegularFile(filepath.Clean(strings.TrimSpace(resultsPath)))
		if err != nil {
			return fmt.Errorf("read image build results: %w", err)
		}
		results, err := decodeBuildResults(resultsPayload, resultsPath)
		if err != nil {
			return err
		}
		if err := validateBuildResults(discovery, results, strings.TrimSpace(expectedMode)); err != nil {
			return err
		}
		if command == "validate-build-results" {
			payload, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			return writeCommandOutput(outputPath, append(payload, '\n'), stdout)
		}
		if command == "validate-package-artifacts" {
			packageResults, err := readPackageBuildResults(packageResultsPath)
			if err != nil {
				return err
			}
			if err := validatePackageArtifacts(discovery, results, packageResults, outRoot, packageResultsPath, strings.TrimSpace(expectedMode)); err != nil {
				return err
			}
			if strings.TrimSpace(previousPackageResultsPath) != "" {
				previous, err := readPackageBuildResults(previousPackageResultsPath)
				if err != nil {
					return err
				}
				if err := validatePackageReleaseEvolution(previous, packageResults); err != nil {
					return err
				}
			}
			payload, err := json.MarshalIndent(packageResults, "", "  ")
			if err != nil {
				return err
			}
			return writeCommandOutput(outputPath, append(payload, '\n'), stdout)
		}
		artifacts, err := buildPackages(discovery, results, version, outRoot)
		if err != nil {
			return err
		}
		payload, err := json.MarshalIndent(PackageBuildResults{
			SchemaVersion: packageBuildResultsSchemaVersion,
			Mode:          results.Mode,
			Packages:      artifacts,
		}, "", "  ")
		if err != nil {
			return err
		}
		return writeCommandOutput(outputPath, append(payload, '\n'), stdout)
	default:
		return fmt.Errorf("unsupported -command %q", command)
	}
}

func selectEngineDiscovery(discovery Discovery, selectedEngineID string) (Discovery, error) {
	if selectedEngineID == "" {
		return discovery, nil
	}
	if strings.TrimSpace(selectedEngineID) != selectedEngineID {
		return Discovery{}, fmt.Errorf("-engine-id must be a canonical discovered engineId without surrounding whitespace")
	}
	for _, source := range discovery.Engines {
		if source.EngineID == selectedEngineID {
			discovery.Engines = []EngineSource{source}
			return discovery, nil
		}
	}
	return Discovery{}, fmt.Errorf("-engine-id %q is not a canonical discovered engineId", selectedEngineID)
}

func rejectSelectedEngineForPackageCommand(command, selectedEngineID string) error {
	if selectedEngineID == "" {
		return nil
	}
	return fmt.Errorf("-engine-id is only supported by discover and validate-build-results; %s requires the complete discovered Engine set", command)
}

func readPackageBuildResults(path string) (PackageBuildResults, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return PackageBuildResults{}, fmt.Errorf("-package-build-results is required")
	}
	payload, err := readRegularFile(filepath.Clean(path))
	if err != nil {
		return PackageBuildResults{}, fmt.Errorf("read package build results: %w", err)
	}
	return decodePackageBuildResults(payload, path)
}

func writeCommandOutput(path string, payload []byte, stdout io.Writer) error {
	path = strings.TrimSpace(path)
	if path == "" {
		if _, err := stdout.Write(append(payload, '\n')); err != nil {
			return fmt.Errorf("write command output: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create command output directory: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write command output %q: %w", path, err)
	}
	return nil
}
