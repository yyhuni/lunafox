package main

import (
	"flag"
	"fmt"
	"os"
)

type docGenOptions struct {
	inputFile  string
	outputFile string
	outputDir  string
}

func main() {
	opts, err := parseDocGenOptions()
	if err != nil {
		exitWithError(err)
	}

	outputPath, err := runDocGen(opts)
	if err != nil {
		exitWithError(err)
	}

	fmt.Printf("Documentation generated successfully: %s\n", outputPath)
}

func parseDocGenOptions() (docGenOptions, error) {
	inputFile := flag.String("input", "", "Input templates.yaml file")
	outputFile := flag.String("output", "", "Output Markdown documentation file")
	outputDir := flag.String("output-dir", "", "Output directory for generated Markdown (filename derived from metadata.name)")
	flag.Parse()

	opts := docGenOptions{
		inputFile:  *inputFile,
		outputFile: *outputFile,
		outputDir:  *outputDir,
	}

	if opts.inputFile == "" || (opts.outputFile == "" && opts.outputDir == "") {
		return docGenOptions{}, fmt.Errorf("Usage: doc-gen -input <templates.yaml> -output <config-reference.md> OR -output-dir <dir>")
	}
	if opts.outputFile != "" && opts.outputDir != "" {
		return docGenOptions{}, fmt.Errorf("Error: use either -output or -output-dir, not both")
	}

	return opts, nil
}

func runDocGen(opts docGenOptions) (string, error) {
	templateFile, err := loadTemplateFile(opts.inputFile)
	if err != nil {
		return "", err
	}

	doc, err := generateDocumentation(templateFile)
	if err != nil {
		return "", fmt.Errorf("Error generating documentation: %w", err)
	}

	outputPath, err := resolveOutputPath(opts, templateFile)
	if err != nil {
		return "", err
	}

	if err := writeDocumentation(outputPath, doc); err != nil {
		return "", err
	}

	return outputPath, nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
