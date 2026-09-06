package domain

import (
	"path"
	"strings"
	"unicode"
)

const MaxDiagnosticSamples = 20

type DiagnosticSample struct {
	Category     string
	RelativePath string
	ReasonCode   string
}

type Diagnostics struct {
	Samples   []DiagnosticSample
	Total     int
	Truncated bool
}

func (diagnostics Diagnostics) Safe() Diagnostics {
	copyValue := diagnostics
	if copyValue.Total < 0 {
		copyValue.Total = 0
	}
	if len(diagnostics.Samples) > MaxDiagnosticSamples {
		copyValue.Samples = append([]DiagnosticSample(nil), diagnostics.Samples[:MaxDiagnosticSamples]...)
		copyValue.Truncated = true
	} else {
		copyValue.Samples = append([]DiagnosticSample(nil), diagnostics.Samples...)
	}
	if copyValue.Samples == nil {
		copyValue.Samples = []DiagnosticSample{}
	}
	if copyValue.Total < len(copyValue.Samples) {
		copyValue.Total = len(copyValue.Samples)
	}
	for index := range copyValue.Samples {
		copyValue.Samples[index].Category = safeToken(copyValue.Samples[index].Category)
		copyValue.Samples[index].RelativePath = safeRelativePath(copyValue.Samples[index].RelativePath)
		copyValue.Samples[index].ReasonCode = safeToken(copyValue.Samples[index].ReasonCode)
	}
	return copyValue
}

func safeToken(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, character := range value {
		if character < 128 && (unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_' || character == '-' || character == '.') {
			builder.WriteRune(character)
		} else {
			builder.WriteByte('_')
		}
		if builder.Len() >= 64 {
			break
		}
	}
	return builder.String()
}

func safeRelativePath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "/")
	cleaned := path.Clean(value)
	if value == "" || strings.IndexFunc(value, unicode.IsControl) >= 0 || cleaned == "." || strings.HasPrefix(cleaned, "/") || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	if len(cleaned) > 512 {
		return cleaned[:512]
	}
	return cleaned
}

type CleanupStatus string

const (
	CleanupPending  CleanupStatus = "pending"
	CleanupClean    CleanupStatus = "clean"
	CleanupResidual CleanupStatus = "residual"
)
