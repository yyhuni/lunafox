package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentlicense"
)

//go:embed issuer.html
var issuerPage []byte

type localIssuer struct {
	privateKey ed25519.PrivateKey
	publicKey  string
	keyPath    string
	origin     string
	session    string
	now        func() time.Time
}

func (issuer *localIssuer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		// The exact numeric loopback Host rejects DNS rebinding; session tokens
		// stay in the URL fragment until JavaScript sends an explicit header.
		if r.Host != strings.TrimPrefix(issuer.origin, "http://") {
			http.Error(w, "拒绝非本机请求", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(issuerPage)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+issuer.session)) != 1 {
			http.Error(w, "会话无效，请使用终端中的完整地址重新打开", http.StatusForbidden)
			return
		}
		if r.Header.Get("Origin") != issuer.origin {
			http.Error(w, "请求来源无效", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "仅允许POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/api/info" {
			writeJSON(w, map[string]string{"publicKey": issuer.publicKey, "keyPath": issuer.keyPath})
			return
		}
		if r.URL.Path != "/api/issue" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "请求必须为JSON", http.StatusUnsupportedMediaType)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
		if err != nil {
			http.Error(w, "请求内容过大", http.StatusRequestEntityTooLarge)
			return
		}
		var input struct {
			AgentLimit int `json:"agentLimit"`
			Months     int `json:"months"`
		}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "请输入有效额度和期限", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			http.Error(w, "请求包含多余内容", http.StatusBadRequest)
			return
		}
		claims, err := agentlicense.NewClaims(input.AgentLimit, input.Months, issuer.now())
		if err != nil {
			http.Error(w, "节点额度须大于3，期限须为1至120个月", http.StatusBadRequest)
			return
		}
		key, err := agentlicense.Sign(issuer.privateKey, claims)
		if err != nil {
			http.Error(w, "签发失败，请检查本地私钥", http.StatusInternalServerError)
			return
		}
		writeJSON(w, struct {
			Key        string `json:"key"`
			LicenseID  string `json:"licenseId"`
			AgentLimit int    `json:"agentLimit"`
			Months     int    `json:"months"`
			IssuedAt   string `json:"issuedAt"`
			ExpiresAt  string `json:"expiresAt"`
		}{key, claims.LicenseID, claims.AgentLimit, claims.Months, time.Unix(claims.IssuedAt, 0).UTC().Format(time.RFC3339), time.Unix(claims.ExpiresAt, 0).UTC().Format(time.RFC3339)})
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(value)
}
