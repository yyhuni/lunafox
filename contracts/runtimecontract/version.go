package runtimecontract

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	apiVersionPattern    = regexp.MustCompile(`^v[0-9]+$`)
	schemaVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+([\-+][0-9A-Za-z.+-]+)?$`)
)

const (
	APIVersionFormatMessage    = "apiVersion must match v<major>"
	SchemaVersionFormatMessage = "schemaVersion must match MAJOR.MINOR.PATCH(+suffix)"
)

// NormalizeVersion trims spaces only.
// Runtime schema/release versions must use bare SemVer (e.g. 1.2.3) and must
// not include a leading v/V. Keep this strict to avoid mixed formats across
// server/agent/worker/invoker boundaries.
func NormalizeVersion(value string) string {
	return strings.TrimSpace(value)
}

// IsValidAPIVersion checks apiVersion format: v<major>.
func IsValidAPIVersion(value string) bool {
	return apiVersionPattern.MatchString(strings.TrimSpace(value))
}

// IsValidSchemaVersion checks schemaVersion format: MAJOR.MINOR.PATCH(+suffix).
// Note: unlike apiVersion (v<major>), schemaVersion does not allow leading v/V.
func IsValidSchemaVersion(value string) bool {
	return schemaVersionPattern.MatchString(NormalizeVersion(value))
}

// APIVersionFieldMessage returns a stable apiVersion format error message for a field.
func APIVersionFieldMessage(field string) string {
	field = strings.TrimSpace(field)
	if field == "" || field == "apiVersion" {
		return APIVersionFormatMessage
	}
	return fmt.Sprintf("%s must match v<major>", field)
}

// SchemaVersionFieldMessage returns a stable schemaVersion format error message for a field.
func SchemaVersionFieldMessage(field string) string {
	field = strings.TrimSpace(field)
	if field == "" || field == "schemaVersion" {
		return SchemaVersionFormatMessage
	}
	return fmt.Sprintf("%s must match MAJOR.MINOR.PATCH(+suffix)", field)
}
