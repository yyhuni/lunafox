package activity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllParams(t *testing.T) {
	tmpl := CommandTemplate{
		RuntimeParams: []Parameter{
			{Var: "RuntimeA"},
		},
		CLIParams: []Parameter{
			{Var: "CliB"},
		},
	}

	params := allParams(tmpl)
	assert.Len(t, params, 2)
	assert.Equal(t, "RuntimeA", params[0].Var)
	assert.Equal(t, "CliB", params[1].Var)
}
