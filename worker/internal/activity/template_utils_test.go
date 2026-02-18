package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidIdentifier(t *testing.T) {
	assert.True(t, isValidIdentifier("Abc_123"))
	assert.False(t, isValidIdentifier("1abc"))
	assert.False(t, isValidIdentifier("bad-var"))
}

func TestIsValidSemanticID(t *testing.T) {
	assert.True(t, isValidSemanticID("timeout-cli"))
	assert.True(t, isValidSemanticID("TimeoutCli"))
	assert.False(t, isValidSemanticID("bad id"))
}

func TestIsValidConfigKey(t *testing.T) {
	assert.True(t, isValidConfigKey("timeout-cli"))
	assert.False(t, isValidConfigKey("BadKey"))
	assert.False(t, isValidConfigKey("bad_key"))
}

func TestIsValidDisplay(t *testing.T) {
	assert.True(t, isValidDisplay("value"))
	assert.True(t, isValidDisplay("comment"))
	assert.True(t, isValidDisplay("hidden"))
	assert.False(t, isValidDisplay("bad"))
}

func TestValidateType(t *testing.T) {
	assert.NoError(t, validateType("text", "string"))
	assert.NoError(t, validateType(1, "integer"))
	assert.NoError(t, validateType(true, "boolean"))

	assert.Error(t, validateType(1, "string"))
	assert.Error(t, validateType("text", "integer"))
	assert.Error(t, validateType("text", "boolean"))
}
