package subdomaindiscoveryruntime

import (
	"strconv"

	subdomainspec "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

func buildPurednsResolveArgs(
	inputFile string,
	outputFile string,
	resolversPath string,
	threads int64,
	rateLimit int64,
	wildcardFilter bool,
) []string {
	args := []string{
		"resolve", inputFile,
		"-r", resolversPath,
		"--write", outputFile,
		"--quiet",
	}
	if !wildcardFilter {
		args = append(args, "--skip-wildcard-filter")
	}
	if threads > 0 {
		args = append(args, "-t", strconv.FormatInt(threads, 10))
	}
	if rateLimit > 0 {
		args = append(args, "--rate-limit", strconv.FormatInt(rateLimit, 10))
	}
	return args
}

func buildPurednsResolveInvocation(
	inputFile string,
	outputFile string,
	resolversPath string,
	config subdomainspec.ResolveConfig,
) (string, []string) {
	args := buildPurednsResolveArgs(inputFile, outputFile, resolversPath, config.Threads, config.RateLimit, config.WildcardFilter)
	return "puredns", args
}

func buildPurednsBruteforceInvocation(
	domain string,
	outputFile string,
	wordlistPath string,
	resolversPath string,
	toolConfig subdomainspec.BruteforceConfig,
) (string, []string) {
	args := []string{
		"bruteforce",
		wordlistPath,
		domain,
		"-r", resolversPath,
		"--write", outputFile,
		"--quiet",
		"-t", strconv.FormatInt(toolConfig.Threads, 10),
		"--rate-limit", strconv.FormatInt(toolConfig.RateLimit, 10),
	}
	if toolConfig.WildcardFilter {
		args = append(args,
			"--wildcard-tests", strconv.FormatInt(toolConfig.WildcardProbeCount, 10),
			"--wildcard-batch", strconv.FormatInt(toolConfig.WildcardBatch, 10),
		)
	} else {
		args = append(args, "--skip-wildcard-filter")
	}
	return "puredns", args
}
