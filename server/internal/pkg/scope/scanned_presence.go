package scope

import "strings"

// Scanned presence is a boolean over a timestamp column. Callers pass a fixed
// column name; this package does not accept the column from the filter string.
type scannedPresence int

const (
	scannedPresenceNone scannedPresence = iota
	scannedPresenceNotNull
	scannedPresenceNull
	scannedPresenceAll
)

// ScannedPresenceAllowed reports whether every scanned predicate is an exact
// true/false check, and is only OR-combined with other scanned predicates.
func ScannedPresenceAllowed(groups []FilterGroup) bool {
	for index, group := range groups {
		if !strings.EqualFold(group.Filter.Field, "scanned") {
			continue
		}
		if group.Filter.Operator != "==" || !isScannedPresenceValue(group.Filter.Value) {
			return false
		}
		if index > 0 && group.LogicalOp == LogicalOr && !strings.EqualFold(groups[index-1].Filter.Field, "scanned") {
			return false
		}
		if index+1 < len(groups) && groups[index+1].LogicalOp == LogicalOr && !strings.EqualFold(groups[index+1].Filter.Field, "scanned") {
			return false
		}
	}
	return true
}

// ScannedPresenceSQL maps scanned=="true" and scanned=="false" onto column
// presence. An empty result means the predicate does not restrict rows. A
// false allowed flag means the caller must fail closed.
func ScannedPresenceSQL(groups []FilterGroup, column string) (string, bool) {
	if !ScannedPresenceAllowed(groups) {
		return "", false
	}
	switch evaluateScannedPresence(groups) {
	case scannedPresenceNone:
		return "1 = 0", true
	case scannedPresenceNotNull:
		return column + " IS NOT NULL", true
	case scannedPresenceNull:
		return column + " IS NULL", true
	default:
		return "", true
	}
}

func isScannedPresenceValue(value string) bool {
	return value == "true" || value == "false"
}

func evaluateScannedPresence(groups []FilterGroup) scannedPresence {
	state := scannedPresenceAll
	seen := false
	for _, group := range groups {
		if !strings.EqualFold(group.Filter.Field, "scanned") {
			continue
		}
		literal := scannedPresenceNull
		if group.Filter.Value == "true" {
			literal = scannedPresenceNotNull
		}
		if !seen || group.LogicalOp == LogicalAnd {
			state = intersectScannedPresence(state, literal)
		} else {
			state = unionScannedPresence(state, literal)
		}
		seen = true
	}
	if !seen {
		return scannedPresenceAll
	}
	return state
}

func intersectScannedPresence(current, literal scannedPresence) scannedPresence {
	switch current {
	case scannedPresenceAll:
		return literal
	case scannedPresenceNone:
		return scannedPresenceNone
	default:
		if current == literal {
			return current
		}
		return scannedPresenceNone
	}
}

func unionScannedPresence(current, literal scannedPresence) scannedPresence {
	switch current {
	case scannedPresenceNone:
		return literal
	case scannedPresenceAll:
		return scannedPresenceAll
	default:
		if current == literal {
			return current
		}
		return scannedPresenceAll
	}
}
