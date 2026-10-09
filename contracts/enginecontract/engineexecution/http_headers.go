package engineexecution

import (
	"fmt"
	"strings"

	"golang.org/x/net/http/httpguts"
)

// validateHTTPHeaders checks Name: Value lines without changing their bytes or
// order. Errors identify the field and index, never potentially secret values.
func validateHTTPHeaders(path string, headers []string) error {
	for index, header := range headers {
		name, value, found := strings.Cut(header, ":")
		if !found || !httpguts.ValidHeaderFieldName(name) {
			return fmt.Errorf("%s[%d] must contain a valid HTTP header name followed by a colon", path, index)
		}
		if strings.TrimSpace(value) == "" || !httpguts.ValidHeaderFieldValue(value) {
			return fmt.Errorf("%s[%d] must contain a non-empty valid HTTP header value", path, index)
		}
	}
	return nil
}
