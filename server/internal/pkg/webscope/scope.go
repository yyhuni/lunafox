// Package webscope defines the read-time URL scope used by Website detail.
package webscope

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
)

// Scope is an immutable, parsed HTTP(S) URL boundary. It deliberately keeps
// persistence out of the model so Website scoping remains a read-time concern.
type Scope struct {
	scheme        string
	hostname      string
	effectivePort string
	path          string
}

// Parse derives a temporary Website relation scope from an accepted observed
// URL. It does not normalize, trim, or rebuild the stored URL value; the
// resulting fields are used only for same-origin/path related-asset reads.
func Parse(rawURL string) (Scope, error) {
	parts, err := parseScopeParts(rawURL)
	if err != nil {
		return Scope{}, err
	}
	return Scope{scheme: parts.scheme, hostname: parts.hostname, effectivePort: parts.port, path: parts.path}, nil
}

// Contains reports whether candidate belongs to the same origin and path
// boundary. Query and fragment do not affect the hierarchy.
func (scope Scope) Contains(candidate string) bool {
	parts, err := parseScopeParts(candidate)
	if err != nil {
		return false
	}
	if parts.scheme != scope.scheme || parts.hostname != scope.hostname {
		return false
	}
	if parts.port != scope.effectivePort {
		return false
	}

	if scope.path == "/" {
		return true
	}

	return parts.path == scope.path || strings.HasPrefix(parts.path, scope.path+"/")
}

type scopeParts struct {
	scheme   string
	hostname string
	port     string
	path     string
}

func parseScopeParts(rawURL string) (scopeParts, error) {
	if _, err := contractresults.ValidateObservedAssetURL(rawURL); err != nil {
		return scopeParts{}, fmt.Errorf("Website URL cannot form a relation scope: %w", err)
	}

	scheme, authorityOffset := observedURLScheme(rawURL)
	authorityEnd := len(rawURL)
	for index := authorityOffset; index < len(rawURL); index++ {
		switch rawURL[index] {
		case '/', '?', '#':
			authorityEnd = index
			index = len(rawURL)
		}
	}
	authority := rawURL[authorityOffset:authorityEnd]
	if authority == "" || strings.Contains(authority, "@") {
		return scopeParts{}, fmt.Errorf("Website URL cannot form a relation scope from user info or empty authority")
	}

	hostname, err := contractresults.DeriveObservedAssetURLHost(rawURL)
	if err != nil {
		return scopeParts{}, fmt.Errorf("Website URL cannot form a relation scope: %w", err)
	}
	port, err := observedURLAuthorityPort(authority)
	if err != nil {
		return scopeParts{}, fmt.Errorf("Website URL cannot form a relation scope: %w", err)
	}
	if port == "" {
		if scheme == "http" {
			port = "80"
		} else {
			port = "443"
		}
	}

	pathEnd := len(rawURL)
	for index := authorityEnd; index < len(rawURL); index++ {
		if rawURL[index] == '?' || rawURL[index] == '#' {
			pathEnd = index
			break
		}
	}
	path := ""
	if authorityEnd < pathEnd && rawURL[authorityEnd] == '/' {
		path = rawURL[authorityEnd:pathEnd]
	}
	if path == "" {
		path = "/"
	} else if path != "/" {
		path = strings.TrimRight(path, "/")
		if path == "" {
			path = "/"
		}
	}

	return scopeParts{scheme: scheme, hostname: hostname, port: port, path: path}, nil
}

func observedURLScheme(rawURL string) (string, int) {
	if strings.EqualFold(rawURL[:len("http://")], "http://") {
		return "http", len("http://")
	}
	return "https", len("https://")
}

func observedURLAuthorityPort(authority string) (string, error) {
	if strings.HasPrefix(authority, "[") || strings.Count(authority, ":") > 1 {
		return "", fmt.Errorf("Website URL authority port is unsupported")
	}
	colon := strings.IndexByte(authority, ':')
	if colon < 0 {
		return "", nil
	}
	port := authority[colon+1:]
	if port == "" || strings.TrimSpace(port) != port {
		return "", fmt.Errorf("Website URL authority port is invalid")
	}
	return port, nil
}

// Scheme returns the derived lowercase scheme for relation-query construction.
func (scope Scope) Scheme() string {
	return scope.scheme
}

// Authorities returns all authority spellings that have the scope's effective
// port. HTTP(S) default ports can be explicit or omitted in persisted URLs.
func (scope Scope) Authorities() []string {
	host := scope.hostname
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	defaultPort := (scope.scheme == "http" && scope.effectivePort == "80") ||
		(scope.scheme == "https" && scope.effectivePort == "443")
	if defaultPort {
		return []string{host, net.JoinHostPort(scope.hostname, scope.effectivePort)}
	}
	return []string{net.JoinHostPort(scope.hostname, scope.effectivePort)}
}

// Path returns the derived relation path boundary for repository query construction.
func (scope Scope) Path() string {
	return scope.path
}

// SQLPredicate returns a parameterized PostgreSQL predicate for a persisted
// complete URL column. The caller supplies only a trusted column identifier.
func (scope Scope) SQLPredicate(column string) (string, []any) {
	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(column) {
		return "1 = 0", nil
	}
	// GORM treats every question mark in a SQL string as a bind marker, even
	// when it appears in a quoted regular expression. Build that delimiter in
	// PostgreSQL so the predicate's bind-marker count remains exact.
	authorityExpression := "LOWER(substring(%s from '^[^:]+://([^/' || chr(63) || '#]+)'))"
	pathExpressionTemplate := "COALESCE(substring(%s from '^[^:]+://[^/' || chr(63) || '#]+([^' || chr(63) || '#]*)'), '')"
	authorities := scope.Authorities()
	placeholders := make([]string, len(authorities))
	args := make([]any, 0, len(authorities)*2+3)
	urlPrefilters := make([]string, len(authorities))
	for index, authority := range authorities {
		prefix := scope.scheme + "://" + authority
		if scope.path != "/" {
			prefix += scope.path
		}
		// The URL trigram index narrows the candidate set. The structured checks
		// below remain authoritative because a prefix alone includes lookalike hosts.
		urlPrefilters[index] = fmt.Sprintf("%s ILIKE ? ESCAPE E'\\\\'", column)
		args = append(args, escapeLike(prefix)+"%")
	}
	for index, authority := range authorities {
		placeholders[index] = "?"
		args = append(args, authority)
	}
	predicate := fmt.Sprintf(
		"(%s) AND "+authorityExpression+" IN (%s) AND LOWER(substring(%s from '^([^:]+)://')) = ?",
		strings.Join(urlPrefilters, " OR "),
		column,
		strings.Join(placeholders, ", "),
		column,
	)
	args = append(args, scope.scheme)
	if scope.path == "/" {
		return predicate, args
	}

	pathExpression := fmt.Sprintf(pathExpressionTemplate, column)
	escapedPath := escapeLike(scope.path)
	predicate += fmt.Sprintf(" AND (%s = ? OR %s LIKE ? ESCAPE E'\\\\')", pathExpression, pathExpression)
	args = append(args, scope.path, escapedPath+"/%")
	return predicate, args
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\`+`\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	return strings.ReplaceAll(value, "_", `\_`)
}
