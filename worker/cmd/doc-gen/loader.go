package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func loadTemplateFile(inputPath string) (TemplateFile, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return TemplateFile{}, fmt.Errorf("Error reading input file: %w", err)
	}

	var templateFile TemplateFile
	if err := yaml.Unmarshal(data, &templateFile); err != nil {
		return TemplateFile{}, fmt.Errorf("Error parsing YAML: %w", err)
	}

	return templateFile, nil
}
