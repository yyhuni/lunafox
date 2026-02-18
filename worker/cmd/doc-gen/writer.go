package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveOutputPath(opts docGenOptions, templateFile TemplateFile) (string, error) {
	if opts.outputFile != "" {
		return opts.outputFile, nil
	}

	if templateFile.Metadata.Name == "" {
		return "", fmt.Errorf("Error: metadata.name is required when using -output-dir")
	}

	return filepath.Join(opts.outputDir, fmt.Sprintf("%s.md", templateFile.Metadata.Name)), nil
}

func writeDocumentation(outputPath, doc string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("Error creating output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(doc), 0644); err != nil {
		return fmt.Errorf("Error writing output file: %w", err)
	}

	return nil
}
