package webscope

import (
	"fmt"
	"strings"

	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
)

// ExtractFilterScope extracts the optional exact Website URL boundary from a
// collection filter and leaves ordinary user filters for the asset query.
func ExtractFilterScope(filter string) (string, *Scope, error) {
	remaining, matches, err := scope.ExtractField(filter, "websiteUrl")
	if err != nil {
		return "", nil, err
	}
	if len(matches) == 0 {
		return remaining, nil, nil
	}
	if len(matches) != 1 {
		return "", nil, fmt.Errorf("websiteUrl scope must appear exactly once")
	}

	groups := scope.ParseFilter(strings.TrimSpace(filter))
	for index, group := range groups {
		if !strings.EqualFold(group.Filter.Field, "websiteUrl") {
			continue
		}
		if group.Filter.Operator != "==" {
			return "", nil, fmt.Errorf("websiteUrl scope requires ==")
		}
		if group.LogicalOp == scope.LogicalOr || (index+1 < len(groups) && groups[index+1].LogicalOp == scope.LogicalOr) {
			return "", nil, fmt.Errorf("websiteUrl scope cannot be part of an OR expression")
		}
	}

	parsed, err := Parse(matches[0].Filter.Value)
	if err != nil {
		return "", nil, err
	}
	return remaining, &parsed, nil
}
