package infrastructure

import (
	"crypto/rand"
	"encoding/hex"

	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
)

type cryptoTokenGenerator struct{}

func (cryptoTokenGenerator) GenerateHex(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func NewCryptoTokenGenerator() agentapp.TokenGenerator {
	return cryptoTokenGenerator{}
}
