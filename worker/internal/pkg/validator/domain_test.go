package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDomain(t *testing.T) {
	assert.ErrorIs(t, ValidateDomain(""), ErrEmptyDomain)
	assert.ErrorIs(t, ValidateDomain("bad domain"), ErrInvalidDomain)
	assert.NoError(t, ValidateDomain("example.com"))
}

func TestNormalizeDomain(t *testing.T) {
	normalized, err := NormalizeDomain("Example.COM.")
	assert.NoError(t, err)
	assert.Equal(t, "example.com", normalized)

	_, err = NormalizeDomain("")
	assert.ErrorIs(t, err, ErrEmptyDomain)

	_, err = NormalizeDomain("bad domain")
	assert.ErrorIs(t, err, ErrInvalidDomain)
}

func TestIsValidSubdomainFormat(t *testing.T) {
	assert.False(t, IsValidSubdomainFormat(""))
	assert.False(t, IsValidSubdomainFormat("# comment"))
	assert.False(t, IsValidSubdomainFormat("bad domain"))
	assert.False(t, IsValidSubdomainFormat("127.0.0.1"))
	assert.True(t, IsValidSubdomainFormat("a.example.com"))
}
