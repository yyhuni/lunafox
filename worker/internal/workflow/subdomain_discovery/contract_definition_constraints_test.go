package subdomain_discovery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContractDefinitionIncludesParamConstraints(t *testing.T) {
	def := GetContractDefinition()

	var reconThreadsMinimum *int
	var bruteforceWordlistMinLength *int
	var bruteforceRateLimitMinimum *int

	for _, stage := range def.Stages {
		switch stage.ID {
		case stageRecon:
			for _, tool := range stage.Tools {
				if tool.ID != toolSubfinder {
					continue
				}
				for _, param := range tool.Params {
					if param.Key == "threads-cli" {
						reconThreadsMinimum = param.Minimum
					}
				}
			}
		case stageBruteforce:
			for _, tool := range stage.Tools {
				if tool.ID != toolSubdomainBruteforce {
					continue
				}
				for _, param := range tool.Params {
					if param.Key == "subdomain-wordlist-name-runtime" {
						bruteforceWordlistMinLength = param.MinLength
					}
					if param.Key == "rate-limit-cli" {
						bruteforceRateLimitMinimum = param.Minimum
					}
				}
			}
		}
	}

	require.NotNil(t, reconThreadsMinimum)
	require.Equal(t, 1, *reconThreadsMinimum)

	require.NotNil(t, bruteforceWordlistMinLength)
	require.Equal(t, 1, *bruteforceWordlistMinLength)

	require.NotNil(t, bruteforceRateLimitMinimum)
	require.Equal(t, 1, *bruteforceRateLimitMinimum)
}
