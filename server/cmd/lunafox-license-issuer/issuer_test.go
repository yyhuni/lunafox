package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentlicense"
)

func TestPrivateKeyPersistsWithRestrictedPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issuer", "issuer-key")
	first, err := loadOrCreatePrivateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private key permissions = %o", info.Mode().Perm())
	}
	second, err := loadOrCreatePrivateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("restarting issuer replaced publisher identity")
	}
	publicKey := first.Public().(ed25519.PublicKey)
	if len(publicKey) != ed25519.PublicKeySize {
		t.Fatal("invalid publisher identity")
	}
}

func TestInvalidPrivateKeyIsNeverReplaced(t *testing.T) {
	for _, sample := range []struct {
		name, content string
		permissions   os.FileMode
	}{
		{"corrupt", "not-a-private-key", 0o600},
		{"empty", "", 0o600},
		{"public-readable", privateKeyPrefix + strings.Repeat("A", 43) + "\n", 0o644},
		{"oversized", strings.Repeat("A", 257), 0o600},
	} {
		t.Run(sample.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "issuer-key")
			if err := os.WriteFile(path, []byte(sample.content), sample.permissions); err != nil {
				t.Fatal(err)
			}
			if _, err := loadOrCreatePrivateKey(path); err == nil {
				t.Fatal("invalid private file accepted")
			}
			remaining, _ := os.ReadFile(path)
			if string(remaining) != sample.content {
				t.Fatal("invalid private key silently overwritten")
			}
		})
	}
	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "original")
		if _, err := loadOrCreatePrivateKey(target); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if _, err := loadOrCreatePrivateKey(link); err == nil {
			t.Fatal("symlink private key accepted")
		}
	})
}

func TestNormalStartupRequiresTheExistingPublisherKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "issuer-key")
	if _, err := loadPrivateKey(path); err == nil || !strings.Contains(err.Error(), "未找到作者私钥") {
		t.Fatalf("missing key should fail closed: %v", err)
	}
	otherPath := filepath.Join(t.TempDir(), "issuer-key")
	otherKey, err := loadOrCreatePrivateKey(otherPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePublisherPrivateKey(otherKey); err == nil || !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("unrelated key should not be usable for normal startup: %v", err)
	}
}

func newTestIssuer(t *testing.T) *localIssuer {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &localIssuer{privateKey: privateKey, publicKey: base64.RawURLEncoding.EncodeToString(publicKey), keyPath: "/private/issuer-key", origin: "http://127.0.0.1:45678", session: "test-session-token", now: func() time.Time { return time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC) }}
}

func issuerRequest(issuer *localIssuer, path, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, issuer.origin+path, strings.NewReader(body))
	r.Header.Set("Origin", issuer.origin)
	r.Header.Set("Authorization", "Bearer "+issuer.session)
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestIssuerReturnsAuthenticPortableKeyAndNoPrivateMaterial(t *testing.T) {
	issuer := newTestIssuer(t)
	w := httptest.NewRecorder()
	issuer.handler().ServeHTTP(w, issuerRequest(issuer, "/api/issue", `{"agentLimit":10,"months":1}`))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var output struct {
		Key       string `json:"key"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	verifier, err := agentlicense.NewVerifier(issuer.publicKey)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := verifier.Verify(output.Key)
	if err != nil || claims.AgentLimit != 10 || claims.Months != 1 || output.ExpiresAt != "2026-02-28T12:00:00Z" {
		t.Fatalf("invalid signed output: %+v %v", claims, err)
	}
	seed := base64.RawURLEncoding.EncodeToString(issuer.privateKey.Seed())
	if strings.Contains(w.Body.String(), seed) {
		t.Fatal("private key leaked in issuance response")
	}
	info := httptest.NewRecorder()
	issuer.handler().ServeHTTP(info, issuerRequest(issuer, "/api/info", `{}`))
	if info.Code != http.StatusOK || strings.Contains(info.Body.String(), seed) {
		t.Fatalf("private material leaked via info: status %d", info.Code)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("issued key allowed in HTTP cache")
	}
}

func TestIssuerRejectsCrossSiteAndMalformedRequests(t *testing.T) {
	issuer := newTestIssuer(t)
	for _, sample := range []struct {
		name, path, body string
		alter            func(*http.Request)
		want             int
	}{
		{"foreign-host", "/api/issue", `{"agentLimit":10,"months":1}`, func(r *http.Request) { r.Host = "evil.example" }, http.StatusForbidden},
		{"foreign-origin", "/api/issue", `{"agentLimit":10,"months":1}`, func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, http.StatusForbidden},
		{"missing-origin", "/api/issue", `{"agentLimit":10,"months":1}`, func(r *http.Request) { r.Header.Del("Origin") }, http.StatusForbidden},
		{"missing-session", "/api/issue", `{"agentLimit":10,"months":1}`, func(r *http.Request) { r.Header.Del("Authorization") }, http.StatusForbidden},
		{"invalid-session", "/api/issue", `{"agentLimit":10,"months":1}`, func(r *http.Request) { r.Header.Set("Authorization", "Bearer wrong") }, http.StatusForbidden},
		{"form-post", "/api/issue", `{"agentLimit":10,"months":1}`, func(r *http.Request) { r.Header.Set("Content-Type", "application/x-www-form-urlencoded") }, http.StatusUnsupportedMediaType},
		{"get", "/api/issue", `{}`, func(r *http.Request) { r.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"unknown-field", "/api/issue", `{"agentLimit":10,"months":1,"privateKey":"other"}`, nil, http.StatusBadRequest},
		{"trailing-json", "/api/issue", `{"agentLimit":10,"months":1}{}`, nil, http.StatusBadRequest},
		{"missing-limit", "/api/issue", `{"months":1}`, nil, http.StatusBadRequest},
		{"invalid-limit", "/api/issue", `{"agentLimit":3,"months":1}`, nil, http.StatusBadRequest},
		{"missing-months", "/api/issue", `{"agentLimit":10}`, nil, http.StatusBadRequest},
		{"invalid-months", "/api/issue", `{"agentLimit":10,"months":121}`, nil, http.StatusBadRequest},
		{"oversized", "/api/issue", strings.Repeat("x", 1025), nil, http.StatusRequestEntityTooLarge},
	} {
		t.Run(sample.name, func(t *testing.T) {
			r := issuerRequest(issuer, sample.path, sample.body)
			if sample.alter != nil {
				sample.alter(r)
			}
			w := httptest.NewRecorder()
			issuer.handler().ServeHTTP(w, r)
			if w.Code != sample.want {
				t.Fatalf("status=%d want=%d body=%s", w.Code, sample.want, w.Body.String())
			}
		})
	}
	foreignPage := httptest.NewRequest(http.MethodGet, "http://evil.example/", nil)
	w := httptest.NewRecorder()
	issuer.handler().ServeHTTP(w, foreignPage)
	if w.Code != http.StatusForbidden {
		t.Fatal("page allowed through DNS rebinding host")
	}
}
