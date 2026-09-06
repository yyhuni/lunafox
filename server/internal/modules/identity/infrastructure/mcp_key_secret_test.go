package infrastructure

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestMCPKeySecretGeneratorProducesRandomURLSafeDigestableKeys(t *testing.T) {
	generator := NewMCPKeySecretGenerator()
	first, firstDigest, err := generator.Generate()
	if err != nil {
		t.Fatalf("first key generation: %v", err)
	}
	second, secondDigest, err := generator.Generate()
	if err != nil {
		t.Fatalf("second key generation: %v", err)
	}
	if first == second || firstDigest == secondDigest {
		t.Fatal("two generated keys unexpectedly matched")
	}
	for _, key := range []string{first, second} {
		if !strings.HasPrefix(key, "lf_mcp_") {
			t.Fatalf("key %q has unexpected prefix", key)
		}
		encoded := strings.TrimPrefix(key, "lf_mcp_")
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil || len(decoded) != mcpKeyRandomBytes {
			t.Fatalf("key payload is not 32-byte raw URL-safe base64: len=%d err=%v", len(decoded), err)
		}
		if strings.ContainsAny(key, "=+/\n\r") {
			t.Fatalf("key contains non URL-safe characters: %q", key)
		}
	}
	if generator.Digest(first) != firstDigest || generator.Digest(second) != secondDigest {
		t.Fatal("Digest does not reproduce Generate digest")
	}
	if len(firstDigest) != 64 || len(secondDigest) != 64 {
		t.Fatalf("unexpected SHA-256 digest length: %d, %d", len(firstDigest), len(secondDigest))
	}
}
