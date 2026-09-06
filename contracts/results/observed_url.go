package results

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ObservedAssetURLMaxBytes keeps complete URL natural keys safely below the
// PostgreSQL B-tree tuple limit. URLs are rejected at admission, never cut.
const ObservedAssetURLMaxBytes = 2000

// ValidateObservedAssetURL applies the shared transport-safety contract for
// observed assets. It intentionally returns the input unchanged: URL parsing,
// percent decoding, trimming, and reconstruction would destroy scan evidence.
func ValidateObservedAssetURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("observed asset url is required")
	}
	if !utf8.ValidString(raw) {
		return "", fmt.Errorf("observed asset url must be valid UTF-8")
	}
	if strings.ContainsAny(raw, "\x00\r\n") {
		return "", fmt.Errorf("observed asset url contains an unsafe control character")
	}
	if len(raw) > ObservedAssetURLMaxBytes {
		return "", fmt.Errorf("observed asset url exceeds %d UTF-8 bytes", ObservedAssetURLMaxBytes)
	}
	if observedAssetURLAuthorityOffset(raw) == 0 {
		return "", fmt.Errorf("observed asset url must begin with http:// or https://")
	}
	return raw, nil
}

// DeriveObservedAssetURLHost derives only the authority Host needed for
// ownership checks. It never parses or rebuilds the path, query, or fragment,
// and callers must never write the derived value back into the URL field.
func DeriveObservedAssetURLHost(raw string) (string, error) {
	if _, err := ValidateObservedAssetURL(raw); err != nil {
		return "", err
	}
	offset := observedAssetURLAuthorityOffset(raw)
	authorityEnd := len(raw)
	for index := offset; index < len(raw); index++ {
		switch raw[index] {
		case '/', '?', '#':
			authorityEnd = index
			index = len(raw)
		}
	}
	authority := raw[offset:authorityEnd]
	if authority == "" {
		return "", fmt.Errorf("observed asset url authority is required")
	}

	// The final @ has the only unambiguous RFC authority meaning. Anything
	// before it is userinfo and is intentionally irrelevant to Host ownership.
	hostPort := authority
	if at := strings.LastIndexByte(hostPort, '@'); at >= 0 {
		hostPort = hostPort[at+1:]
	}
	if hostPort == "" || strings.TrimSpace(hostPort) != hostPort {
		return "", fmt.Errorf("observed asset url authority host is invalid")
	}

	// LunaFox's established Target/Host model supports DNS names and IPv4.
	// Bracketed or unbracketed IPv6 remains outside that model rather than being
	// guessed from a malformed authority.
	if strings.HasPrefix(hostPort, "[") || strings.Count(hostPort, ":") > 1 {
		return "", fmt.Errorf("observed asset url authority host is unsupported")
	}
	host := hostPort
	if colon := strings.IndexByte(hostPort, ':'); colon >= 0 {
		// Port text belongs to the preserved URL identity. Host ownership only
		// needs the single separator to be unambiguous; port syntax is not part
		// of the minimum admission contract.
		host = hostPort[:colon]
	}
	if host == "" || strings.TrimSpace(host) != host {
		return "", fmt.Errorf("observed asset url authority host is invalid")
	}
	normalizedHost, ok := normalizeHostPortHost(host)
	if !ok {
		return "", fmt.Errorf("observed asset url authority host is invalid")
	}
	return normalizedHost, nil
}

// ReadObservedAssetURLLines reads the strict LF-delimited websiteURLs product.
// It deliberately preserves every record byte until admission so a CRLF or an
// unterminated line is rejected rather than silently repaired by a line scanner.
func ReadObservedAssetURLLines(reader io.Reader, visit func(string) error) error {
	if reader == nil {
		return fmt.Errorf("observed asset URL line reader is required")
	}
	if visit == nil {
		return fmt.Errorf("observed asset URL line visitor is required")
	}

	buffered := bufio.NewReaderSize(reader, ObservedAssetURLMaxBytes+2)
	for {
		line, err := buffered.ReadSlice('\n')
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return nil
			}
			return fmt.Errorf("observed asset URL line must be LF-terminated")
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			return fmt.Errorf("observed asset URL line exceeds %d UTF-8 bytes", ObservedAssetURLMaxBytes)
		}
		if err != nil {
			return fmt.Errorf("read observed asset URL line: %w", err)
		}

		raw := string(line[:len(line)-1])
		if _, err := ValidateObservedAssetURL(raw); err != nil {
			return err
		}
		if err := visit(raw); err != nil {
			return err
		}
	}
}

func observedAssetURLAuthorityOffset(raw string) int {
	if len(raw) >= len("http://") && strings.EqualFold(raw[:len("http://")], "http://") {
		return len("http://")
	}
	if len(raw) >= len("https://") && strings.EqualFold(raw[:len("https://")], "https://") {
		return len("https://")
	}
	return 0
}
