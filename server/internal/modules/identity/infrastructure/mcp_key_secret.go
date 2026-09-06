package infrastructure

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const mcpKeyRandomBytes = 32

// MCPKeySecretGenerator creates URL-safe, high-entropy static credentials.
type MCPKeySecretGenerator struct{}

// NewMCPKeySecretGenerator creates the production key generator.
func NewMCPKeySecretGenerator() MCPKeySecretGenerator {
	return MCPKeySecretGenerator{}
}

// Generate returns the one-time secret and its SHA-256 digest.
func (MCPKeySecretGenerator) Generate() (string, string, error) {
	randomBytes := make([]byte, mcpKeyRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", err
	}
	secret := "lf_mcp_" + base64.RawURLEncoding.EncodeToString(randomBytes)
	digest := sha256.Sum256([]byte(secret))
	return secret, hex.EncodeToString(digest[:]), nil
}

// Digest returns the persistence digest for a presented secret.
func (MCPKeySecretGenerator) Digest(secret string) string {
	digest := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(digest[:])
}
