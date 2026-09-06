package versioning

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	semVerPattern = regexp.MustCompile(`^\d+\.\d+\.\d+([\-+][0-9A-Za-z.+-]+)?$`)
)

const (
	SemVerFormatMessage = "version must match MAJOR.MINOR.PATCH(-suffix or +suffix)"
)

func Normalize(value string) string {
	return strings.TrimSpace(value)
}

func IsValidSemVer(value string) bool {
	return semVerPattern.MatchString(Normalize(value))
}

func SemVerFieldMessage(field string) string {
	field = strings.TrimSpace(field)
	if field == "" || field == "version" {
		return SemVerFormatMessage
	}
	return fmt.Sprintf("%s must match MAJOR.MINOR.PATCH(-suffix or +suffix)", field)
}
