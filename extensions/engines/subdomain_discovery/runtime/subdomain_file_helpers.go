package subdomaindiscoveryruntime

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (run *discoveryRun) allocateWorkspaceFilePath(namePrefix string) (string, error) {
	if run.allocatedWorkspaceFilePaths == nil {
		run.allocatedWorkspaceFilePaths = map[string]struct{}{}
	}
	return allocateWorkspaceFilePath(run.workspaceDir, namePrefix, run.allocatedWorkspaceFilePaths)
}

func allocateWorkspaceFilePath(workspaceAbs, namePrefix string, reserved map[string]struct{}) (string, error) {
	for i := 1; i <= 999999; i++ {
		path := filepath.Join(workspaceAbs, fmt.Sprintf("%s_%03d.txt", namePrefix, i))
		if _, ok := reserved[path]; ok {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			reserved[path] = struct{}{}
			return path, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect workspace file candidate: %w", err)
		}
	}
	return "", fmt.Errorf("allocate workspace file: exhausted generated names")
}

func requireWorkspaceAbs(workspaceDir, label string) (string, error) {
	workspace := workspaceDir
	if workspace == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	if workspace != strings.TrimSpace(workspace) {
		return "", fmt.Errorf("%s must not have surrounding whitespace", label)
	}
	if !filepath.IsAbs(workspace) {
		return "", fmt.Errorf("%s must be absolute", label)
	}
	// A non-clean workspace could move Engine-local outputs outside the intended directory.
	if filepath.Clean(workspace) != workspace {
		return "", fmt.Errorf("%s must be clean", label)
	}
	info, err := os.Stat(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s must be an existing directory", label)
		}
		return "", fmt.Errorf("inspect %s: %w", label, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s must be an existing directory", label)
	}
	return workspace, nil
}

func countFileLines(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer func() { _ = file.Close() }()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("failed to count non-empty file lines file=%s error=%v", filePath, err)
		return 0
	}
	return count
}
