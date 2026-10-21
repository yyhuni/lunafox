// Package agentlicense authenticates portable, fixed-term Agent entitlements.
package agentlicense

import (
	"bytes"
	"crypto/ed25519"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	keyPrefix       = "LF1"
	signatureDomain = "lunafox.agent-license.v1\n"
	product         = "lunafox.agent-expansion"
	// MaxKeyBytes bounds parsing of customer-supplied authorization material.
	MaxKeyBytes = 4096
)

// ErrInvalidKey identifies malformed, unauthentic, or unsupported license keys.
var ErrInvalidKey = errors.New("invalid agent license key")

//go:embed trusted-public-key.txt
var trustedPublicKey string

// Claims contains the signed, machine-independent entitlement. Times are Unix
// seconds in UTC; importing a key never changes either endpoint.
type Claims struct {
	Version    int    `json:"version"`
	Product    string `json:"product"`
	LicenseID  string `json:"licenseId"`
	AgentLimit int    `json:"agentLimit"`
	Months     int    `json:"months"`
	IssuedAt   int64  `json:"issuedAt"`
	ExpiresAt  int64  `json:"expiresAt"`
}

// Verifier authenticates keys against one publisher-owned Ed25519 public key.
type Verifier struct{ publicKey ed25519.PublicKey }

// NewDefaultVerifier uses the publisher public key embedded in this release.
func NewDefaultVerifier() (*Verifier, error) { return NewVerifier(trustedPublicKey) }

// NewVerifier constructs an isolated verifier from a base64url public key.
func NewVerifier(encodedPublicKey string) (*Verifier, error) {
	encodedPublicKey = strings.TrimSpace(encodedPublicKey)
	publicKey, err := base64.RawURLEncoding.Strict().DecodeString(encodedPublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || base64.RawURLEncoding.EncodeToString(publicKey) != encodedPublicKey {
		return nil, fmt.Errorf("agent license public key must be base64url-encoded Ed25519 public key")
	}
	return &Verifier{publicKey: ed25519.PublicKey(publicKey)}, nil
}

// Verify authenticates structure and signature, leaving temporal state to the
// caller so that an expired stored key can still produce its display metadata.
func (verifier *Verifier) Verify(key string) (Claims, error) {
	var claims Claims
	if verifier == nil || len(verifier.publicKey) != ed25519.PublicKeySize || len(key) > MaxKeyBytes {
		return claims, ErrInvalidKey
	}
	parts := strings.Split(strings.TrimSpace(key), ".")
	if len(parts) != 3 || parts[0] != keyPrefix {
		return claims, ErrInvalidKey
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil || base64.RawURLEncoding.EncodeToString(payload) != parts[1] {
		return claims, ErrInvalidKey
	}
	signature, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != parts[2] {
		return claims, ErrInvalidKey
	}
	if !ed25519.Verify(verifier.publicKey, append([]byte(signatureDomain), payload...), signature) {
		return claims, ErrInvalidKey
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&claims); err != nil {
		return Claims{}, ErrInvalidKey
	}
	canonical, err := json.Marshal(claims)
	// Canonical bytes also reject duplicate fields, case-insensitive aliases and
	// trailing JSON accepted by Go's otherwise deliberately permissive decoder.
	if err != nil || !bytes.Equal(payload, canonical) || validateClaims(claims) != nil {
		return Claims{}, ErrInvalidKey
	}
	return claims, nil
}

// NewClaims creates an entitlement that begins now and ends after calendar
// months, clamping a missing target day to that month's final day.
func NewClaims(agentLimit, months int, now time.Time) (Claims, error) {
	issuedAt := now.UTC().Truncate(time.Second)
	if months < 1 || months > 120 {
		return Claims{}, fmt.Errorf("months must be between 1 and 120")
	}
	claims := Claims{Version: 1, Product: product, LicenseID: uuid.NewString(), AgentLimit: agentLimit, Months: months, IssuedAt: issuedAt.Unix(), ExpiresAt: addCalendarMonths(issuedAt, months).Unix()}
	if err := validateClaims(claims); err != nil {
		return Claims{}, err
	}
	return claims, nil
}

// Sign produces a portable key using the publisher-only private key.
func Sign(privateKey ed25519.PrivateKey, claims Claims) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid Ed25519 private key")
	}
	if err := validateClaims(claims); err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signature := ed25519.Sign(privateKey, append([]byte(signatureDomain), payload...))
	return keyPrefix + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func validateClaims(claims Claims) error {
	if claims.Version != 1 || claims.Product != product {
		return ErrInvalidKey
	}
	id, err := uuid.Parse(claims.LicenseID)
	if err != nil || id == uuid.Nil || id.String() != claims.LicenseID {
		return ErrInvalidKey
	}
	if claims.AgentLimit <= 3 || int64(claims.AgentLimit) > 9007199254740991 || claims.Months < 1 || claims.Months > 120 {
		return ErrInvalidKey
	}
	issuedAt := time.Unix(claims.IssuedAt, 0).UTC()
	if claims.IssuedAt <= 0 || issuedAt.Year() > 9999 || claims.ExpiresAt <= claims.IssuedAt {
		return ErrInvalidKey
	}
	expiresAt := addCalendarMonths(issuedAt, claims.Months)
	if expiresAt.Year() > 9999 || claims.ExpiresAt != expiresAt.Unix() {
		return ErrInvalidKey
	}
	return nil
}

func addCalendarMonths(instant time.Time, months int) time.Time {
	first := time.Date(instant.Year(), instant.Month()+time.Month(months), 1, instant.Hour(), instant.Minute(), instant.Second(), 0, time.UTC)
	lastDay := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	day := instant.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(first.Year(), first.Month(), day, first.Hour(), first.Minute(), first.Second(), 0, time.UTC)
}
