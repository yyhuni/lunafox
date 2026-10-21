package agentlicense

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testPublisher(t *testing.T) (ed25519.PrivateKey, *Verifier) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := NewVerifier(base64.RawURLEncoding.EncodeToString(publicKey))
	if err != nil {
		t.Fatal(err)
	}
	return privateKey, verifier
}

func TestKeyRoundTripAndFixedCalendarMonths(t *testing.T) {
	privateKey, verifier := testPublisher(t)
	for _, sample := range []struct {
		now, want string
		months    int
	}{
		{"2026-01-31T12:34:56Z", "2026-02-28T12:34:56Z", 1},
		{"2028-01-31T12:34:56Z", "2028-02-29T12:34:56Z", 1},
		{"2026-11-30T12:34:56Z", "2027-02-28T12:34:56Z", 3},
		{"2026-01-31T12:34:56+08:00", "2026-03-31T04:34:56Z", 2},
	} {
		t.Run(sample.now, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, sample.now)
			claims, err := NewClaims(10, sample.months, now)
			if err != nil {
				t.Fatal(err)
			}
			if got := time.Unix(claims.ExpiresAt, 0).UTC().Format(time.RFC3339); got != sample.want {
				t.Fatalf("expiresAt=%s want=%s", got, sample.want)
			}
			key, err := Sign(privateKey, claims)
			if err != nil {
				t.Fatal(err)
			}
			got, err := verifier.Verify(" \n" + key + "\n")
			if err != nil || !reflect.DeepEqual(got, claims) {
				t.Fatalf("round trip: %+v %v", got, err)
			}
			// Verification is independent of wall-clock time and machine identity.
			replayed, err := verifier.Verify(key)
			if err != nil || replayed.ExpiresAt != claims.ExpiresAt {
				t.Fatalf("replay changed fixed entitlement: %v", err)
			}
		})
	}
}

func TestVerifyRejectsTamperingAndUnsupportedClaims(t *testing.T) {
	privateKey, verifier := testPublisher(t)
	claims, _ := NewClaims(10, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	key, _ := Sign(privateKey, claims)
	parts := strings.Split(key, ".")
	_, wrongVerifier := testPublisher(t)
	if _, err := wrongVerifier.Verify(key); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("wrong signer accepted: %v", err)
	}
	for name, input := range map[string]string{"empty": "", "demo": "LF-DEMO-1M", "version": "LF2." + parts[1] + "." + parts[2], "signature": parts[0] + "." + parts[1] + "." + strings.Repeat("A", 86), "oversize": strings.Repeat("x", MaxKeyBytes+1), "encoding": parts[0] + "." + parts[1] + "=." + parts[2]} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Verify(input); !errors.Is(err, ErrInvalidKey) {
				t.Fatalf("accepted %s: %v", name, err)
			}
		})
	}
	mutations := map[string]func(*Claims){
		"version": func(c *Claims) { c.Version = 2 }, "product": func(c *Claims) { c.Product = "other" }, "id": func(c *Claims) { c.LicenseID = "" }, "zero-id": func(c *Claims) { c.LicenseID = "00000000-0000-0000-0000-000000000000" }, "limit": func(c *Claims) { c.AgentLimit = 3 }, "months": func(c *Claims) { c.Months = 0 }, "issuedAt": func(c *Claims) { c.IssuedAt = 0 }, "expiresAt": func(c *Claims) { c.ExpiresAt++ },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			invalid := claims
			mutate(&invalid)
			payload, _ := json.Marshal(invalid)
			signed := signTestPayload(privateKey, payload, signatureDomain)
			if _, err := verifier.Verify(signed); !errors.Is(err, ErrInvalidKey) {
				t.Fatalf("accepted invalid claims: %v", err)
			}
			if _, err := Sign(privateKey, invalid); err == nil {
				t.Fatal("signer accepted invalid claims")
			}
		})
	}
	payload, _ := json.Marshal(claims)
	for name, altered := range map[string][]byte{"duplicate": []byte(strings.Replace(string(payload), `"version":1`, `"version":1,"version":1`, 1)), "unknown": []byte(strings.Replace(string(payload), `"version":1`, `"version":1,"unknown":true`, 1)), "whitespace": append([]byte(" "), payload...), "trailing": append(append([]byte{}, payload...), []byte("{}")...)} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifier.Verify(signTestPayload(privateKey, altered, signatureDomain)); !errors.Is(err, ErrInvalidKey) {
				t.Fatalf("accepted noncanonical payload: %v", err)
			}
		})
	}
	if _, err := verifier.Verify(signTestPayload(privateKey, payload, "different.domain\n")); !errors.Is(err, ErrInvalidKey) {
		t.Fatal("accepted different signature domain")
	}
}

func TestInvalidInputsNeverPanic(t *testing.T) {
	if _, err := NewVerifier(""); err == nil {
		t.Fatal("missing public key accepted")
	}
	if _, err := NewVerifier(base64.RawURLEncoding.EncodeToString(make([]byte, 31))); err == nil {
		t.Fatal("short public key accepted")
	}
	var verifier *Verifier
	if _, err := verifier.Verify("LF1.a.b"); !errors.Is(err, ErrInvalidKey) {
		t.Fatal(err)
	}
	if _, err := Sign(nil, Claims{}); err == nil {
		t.Fatal("short private key accepted")
	}
	for _, months := range []int{0, -1, 121} {
		if _, err := NewClaims(10, months, time.Now()); err == nil {
			t.Fatalf("accepted months %d", months)
		}
	}
	if _, err := NewClaims(3, 1, time.Now()); err == nil {
		t.Fatal("accepted base quota as expansion")
	}
}

func TestEmbeddedPublisherPublicKeyIsValid(t *testing.T) {
	if _, err := NewDefaultVerifier(); err != nil {
		t.Fatal(err)
	}
}

func signTestPayload(privateKey ed25519.PrivateKey, payload []byte, domain string) string {
	signature := ed25519.Sign(privateKey, append([]byte(domain), payload...))
	return keyPrefix + "." + base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature)
}
