package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// BuildWebSocketURL derives the agent WebSocket endpoint from the server URL.
func BuildWebSocketURL(serverURL string) (string, error) {
	trimmed := strings.TrimSpace(serverURL)
	if trimmed == "" {
		return "", errors.New("server URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		if parsed.Scheme == "" {
			return "", errors.New("server URL scheme is required")
		}
		return "", fmt.Errorf("unsupported server URL scheme: %s", parsed.Scheme)
	}

	parsed.Path = buildWSPath(parsed.Path)
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}

func buildWSPath(path string) string {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		return "/api/agent/ws"
	}
	if strings.HasSuffix(trimmed, "/api") {
		return trimmed + "/agent/ws"
	}
	return trimmed + "/api/agent/ws"
}
