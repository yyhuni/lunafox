package middleware

import (
	"net/http"
	"strings"
)

// NormalizeTrailingSlash returns an http.Handler that strips trailing slashes
// before passing to the Gin router. This must wrap the entire router.
func NormalizeTrailingSlash(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			r.URL.Path = strings.TrimSuffix(path, "/")
		}
		handler.ServeHTTP(w, r)
	})
}
