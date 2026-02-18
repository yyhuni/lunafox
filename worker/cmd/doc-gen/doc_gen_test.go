package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyhuni/lunafox/worker/internal/activity"
	"github.com/yyhuni/lunafox/worker/internal/workflow"
)

func TestParseDocGenOptions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      docGenOptions
		errSubstr string
	}{
		{
			name:      "missing required args",
			args:      []string{"doc-gen"},
			errSubstr: "Usage: doc-gen",
		},
		{
			name:      "missing output args",
			args:      []string{"doc-gen", "-input", "templates.yaml"},
			errSubstr: "Usage: doc-gen",
		},
		{
			name: "both output and output-dir provided",
			args: []string{
				"doc-gen",
				"-input", "templates.yaml",
				"-output", "doc.md",
				"-output-dir", "docs",
			},
			errSubstr: "use either -output or -output-dir",
		},
		{
			name: "valid output file",
			args: []string{
				"doc-gen",
				"-input", "templates.yaml",
				"-output", "doc.md",
			},
			want: docGenOptions{
				inputFile:  "templates.yaml",
				outputFile: "doc.md",
			},
		},
		{
			name: "valid output directory",
			args: []string{
				"doc-gen",
				"-input", "templates.yaml",
				"-output-dir", "docs",
			},
			want: docGenOptions{
				inputFile: "templates.yaml",
				outputDir: "docs",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDocGenOptionsForTest(t, tc.args)
			if tc.errSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveOutputPath(t *testing.T) {
	tests := []struct {
		name         string
		opts         docGenOptions
		metadataName string
		want         string
		errSubstr    string
	}{
		{
			name: "use explicit output file",
			opts: docGenOptions{
				outputFile: "/tmp/doc.md",
				outputDir:  "/tmp/docs",
			},
			want: "/tmp/doc.md",
		},
		{
			name: "derive from output-dir and metadata name",
			opts: docGenOptions{
				outputDir: "/tmp/docs",
			},
			metadataName: "subdomain_discovery",
			want:         filepath.Join("/tmp/docs", "subdomain_discovery.md"),
		},
		{
			name: "missing metadata name for output-dir mode",
			opts: docGenOptions{
				outputDir: "/tmp/docs",
			},
			errSubstr: "metadata.name is required when using -output-dir",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			templateFile := minimalTemplateFile()
			templateFile.Metadata.Name = tc.metadataName

			got, err := resolveOutputPath(tc.opts, templateFile)
			if tc.errSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGenerateDocumentationValidation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		templateFile := minimalTemplateFile()

		doc, err := generateDocumentation(templateFile)
		require.NoError(t, err)
		assert.Contains(t, doc, "# Subdomain Discovery")
		assert.Contains(t, doc, "## Scan Workflow")
	})

	t.Run("missing metadata.doc", func(t *testing.T) {
		templateFile := minimalTemplateFile()
		templateFile.Metadata.Doc = nil

		_, err := generateDocumentation(templateFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata.doc is required")
	})

	t.Run("missing required label key", func(t *testing.T) {
		templateFile := minimalTemplateFile()
		delete(templateFile.Metadata.Doc.Labels, "workflow_includes")

		_, err := generateDocumentation(templateFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata.doc.labels missing key: workflow_includes")
	})

	t.Run("unsupported section id", func(t *testing.T) {
		templateFile := minimalTemplateFile()
		templateFile.Metadata.Doc.Sections = []workflow.DocSection{
			{ID: "unknown", Title: "Unknown Section"},
		}

		_, err := generateDocumentation(templateFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata.doc.sections: unsupported section id unknown")
	})

	t.Run("missing section title", func(t *testing.T) {
		templateFile := minimalTemplateFile()
		templateFile.Metadata.Doc.Sections = []workflow.DocSection{
			{ID: "workflow", Title: ""},
		}

		_, err := generateDocumentation(templateFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata.doc.sections: title is required for section workflow")
	})
}

func TestLoadTemplateFileErrors(t *testing.T) {
	t.Run("read file error", func(t *testing.T) {
		_, err := loadTemplateFile(filepath.Join(t.TempDir(), "not-found.yaml"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Error reading input file")
	})

	t.Run("yaml parse error", func(t *testing.T) {
		input := filepath.Join(t.TempDir(), "invalid.yaml")
		err := os.WriteFile(input, []byte("metadata: ["), 0644)
		require.NoError(t, err)

		_, err = loadTemplateFile(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Error parsing YAML")
	})
}

func parseDocGenOptionsForTest(t *testing.T, args []string) (docGenOptions, error) {
	t.Helper()

	originalArgs := os.Args
	originalCommandLine := flag.CommandLine

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flag.CommandLine = fs
	os.Args = args

	defer func() {
		os.Args = originalArgs
		flag.CommandLine = originalCommandLine
	}()

	return parseDocGenOptions()
}

func minimalTemplateFile() TemplateFile {
	return TemplateFile{
		Metadata: workflow.WorkflowMetadata{
			Name:        "subdomain_discovery",
			DisplayName: "Subdomain Discovery",
			Description: "Discover subdomains",
			Version:     "1.0.0",
			TargetTypes: []string{"domain"},
			Stages: []workflow.StageMetadata{
				{
					ID:          "recon",
					Name:        "Reconnaissance",
					Description: "Collect subdomains from passive sources",
					Required:    true,
				},
			},
			Doc: &workflow.DocMetadata{
				Sections: []workflow.DocSection{
					{ID: "workflow", Title: "Scan Workflow"},
				},
				Labels: requiredLabelsForTest(),
			},
		},
		Tools: map[string]activity.CommandTemplate{},
	}
}

func requiredLabelsForTest() map[string]string {
	labels := make(map[string]string, len(requiredLabelKeys))
	for _, key := range requiredLabelKeys {
		labels[key] = key
	}
	return labels
}
