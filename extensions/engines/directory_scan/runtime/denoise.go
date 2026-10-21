package directoryscanruntime

import (
	"net/url"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

// directoryDenoiseOptions controls the opt-in result filters applied while
// parsing FFUF artifacts. Both filters are disabled by default.
type directoryDenoiseOptions struct {
	authRedirect bool
	backupHTML   bool
}

// authRedirectPaths are the only redirect paths treated as a login wall.
// Exact path or child-path matches are allowed; bare substring matches are not.
var authRedirectPaths = []string{
	"/login",
	"/signin",
	"/auth/login",
	"/sso",
	"/cas/login",
	"/oauth2/authorize",
	"/oauth/authorize",
	"/user/login",
	"/admin/login",
	"/wp-login.php",
}

func isRedirectStatus(status int) bool {
	switch status {
	case 301, 302, 303, 307:
		return true
	default:
		return false
	}
}

// filteredByAuthRedirect reports whether a redirect record points at a known
// login/SSO path. The location is resolved against the record URL so relative
// and cross-host Location values are handled consistently.
func (options directoryDenoiseOptions) filteredByAuthRedirect(item enginecontract.Directory, location string) bool {
	if !options.authRedirect || location == "" || !isRedirectStatus(item.Status) {
		return false
	}
	path := resolveRedirectPath(item.URL, location)
	if path == "" {
		return false
	}
	for _, prefix := range authRedirectPaths {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func resolveRedirectPath(baseURL, location string) string {
	if reference, err := url.Parse(location); err == nil {
		if base, err := url.Parse(baseURL); err == nil {
			return strings.ToLower(base.ResolveReference(reference).Path)
		}
		return strings.ToLower(reference.Path)
	}
	text := location
	if cut := strings.IndexAny(text, "?#"); cut >= 0 {
		text = text[:cut]
	}
	return strings.ToLower(text)
}

// filteredByBackupHTML reports whether a backup-suffixed URL is actually an
// HTML page. text/plain, application/*, and empty content types are kept so
// real backup files are not dropped.
func (options directoryDenoiseOptions) filteredByBackupHTML(item enginecontract.Directory) bool {
	if !options.backupHTML {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(item.ContentType))
	if !strings.HasPrefix(contentType, "text/html") {
		return false
	}
	parsed, err := url.Parse(item.URL)
	if err != nil {
		return false
	}
	path := strings.ToLower(parsed.Path)
	for _, suffix := range targetVariantSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}
