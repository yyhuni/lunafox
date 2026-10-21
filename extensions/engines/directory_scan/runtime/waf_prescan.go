package directoryscanruntime

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

const (
	wafProbePathPrefix = "/__lunafox_waf_probe__"
	wafProbeUserAgent  = "Fuzz Faster U Fool v2.2.1"
	wafProbeBodyLimit  = 64 * 1024
)

// strongWAFBodySignatures are challenge/block page fragments that rarely appear
// on a healthy control response. Generic words such as "access denied" or
// "captcha" are deliberately excluded to avoid skipping legitimate sites.
var strongWAFBodySignatures = []string{
	"just a moment",
	"cf-chl",
	"attention required",
	"challenge-platform",
	"cloudflare ray id",
	"incapsula incident",
	"sucuri website firewall",
	"modsecurity",
	"request blocked",
}

var wafBlockHeaderNames = []string{
	"cf-mitigated",
	"x-sucuri-id",
	"x-sucuri-block",
	"x-iinfo",
}

// wafPrescanner decides whether a Website origin is already blocked by a WAF.
type wafPrescanner interface {
	Blocked(context.Context, string) (bool, string, error)
}

type wafProbeResult struct {
	blocked bool
	reason  string
}

type wafProbeEntry struct {
	done   chan struct{}
	result wafProbeResult
}

type wafPrescan struct {
	client  *http.Client
	header  http.Header
	mu      sync.Mutex
	entries map[string]*wafProbeEntry
}

func newWAFPrescan(config enginecontract.FfufConfig) (*wafPrescan, error) {
	header := make(http.Header)
	for _, line := range config.Headers {
		name, value, found := strings.Cut(line, ":")
		if !found {
			return nil, errors.New("configured header is invalid")
		}
		header.Add(strings.TrimSpace(name), strings.TrimSpace(value))
	}
	if header.Get("User-Agent") == "" {
		header.Set("User-Agent", wafProbeUserAgent)
	}
	client := &http.Client{
		Timeout: time.Duration(config.RequestTimeout) * time.Second,
		Transport: &http.Transport{
			// Target traffic parity with FFUF: target certificates are not verified.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &wafPrescan{client: client, header: header, entries: make(map[string]*wafProbeEntry)}, nil
}

// Blocked probes an origin at most once per execution and caches the decision.
// Probe failures are best-effort: callers should scan normally on error.
func (prescan *wafPrescan) Blocked(ctx context.Context, candidate string) (bool, string, error) {
	origin, err := wafProbeOrigin(candidate)
	if err != nil {
		return false, "", err
	}
	prescan.mu.Lock()
	entry, running := prescan.entries[origin]
	if !running {
		entry = &wafProbeEntry{done: make(chan struct{})}
		prescan.entries[origin] = entry
	}
	prescan.mu.Unlock()

	if running {
		select {
		case <-entry.done:
			return entry.result.blocked, entry.result.reason, nil
		case <-ctx.Done():
			return false, "", ctx.Err()
		}
	}

	result := prescan.probe(ctx, origin)
	prescan.mu.Lock()
	entry.result = result
	close(entry.done)
	prescan.mu.Unlock()
	return result.blocked, result.reason, nil
}

func (prescan *wafPrescan) probe(ctx context.Context, origin string) wafProbeResult {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return wafProbeResult{}
	}
	randomURL := origin + wafProbePathPrefix + hex.EncodeToString(suffix)
	randomStatus, randomHeader, randomBody, err := prescan.get(ctx, randomURL)
	if err != nil {
		return wafProbeResult{}
	}
	randomBlocked, reason := looksWAFBlocked(randomStatus, randomHeader, randomBody)
	if !randomBlocked {
		return wafProbeResult{}
	}

	controlStatus, controlHeader, controlBody, err := prescan.get(ctx, origin+"/")
	if err != nil {
		return wafProbeResult{}
	}
	if controlBlocked, _ := looksWAFBlocked(controlStatus, controlHeader, controlBody); controlBlocked {
		return wafProbeResult{}
	}
	if controlStatus < 200 || controlStatus >= 400 {
		return wafProbeResult{}
	}
	return wafProbeResult{blocked: true, reason: reason}
}

func (prescan *wafPrescan) get(ctx context.Context, target string) (int, http.Header, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, nil, err
	}
	request.Header = prescan.header.Clone()
	response, err := prescan.client.Do(request)
	if err != nil {
		return 0, nil, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, wafProbeBodyLimit))
	if err != nil {
		return 0, nil, nil, err
	}
	return response.StatusCode, response.Header.Clone(), body, nil
}

func wafProbeOrigin(candidate string) (string, error) {
	parsed, err := url.Parse(candidate)
	if err != nil {
		return "", err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if (scheme != "http" && scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("candidate %q is not an absolute HTTP URL", candidate)
	}
	return scheme + "://" + parsed.Host, nil
}

func looksWAFBlocked(status int, header http.Header, body []byte) (bool, string) {
	if status == http.StatusForbidden {
		return true, "status_403"
	}
	if status == http.StatusTooManyRequests {
		return true, "status_429"
	}
	for _, name := range wafBlockHeaderNames {
		if header.Get(name) != "" {
			return true, "header_" + name
		}
	}
	lowerBody := strings.ToLower(string(body))
	for _, signature := range strongWAFBodySignatures {
		if strings.Contains(lowerBody, signature) {
			return true, "body_signature"
		}
	}
	return false, ""
}
