package subdomain_discovery

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// countFileLines counts non-empty lines in a file.
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
	return count
}

// sanitizeFilename removes or replaces characters that are invalid in filenames.
func sanitizeFilename(name string) string {
	// Replace common problematic characters.
	re := regexp.MustCompile(`[<>:"/\\|?*\s]`)
	return re.ReplaceAllString(name, "_")
}
