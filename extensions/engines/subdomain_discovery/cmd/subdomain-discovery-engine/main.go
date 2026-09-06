package main

import (
	"context"
	"fmt"

	enginecontract "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
	subdomaindiscoveryruntime "github.com/yyhuni/lunafox/engines/subdomain_discovery/runtime"
)

func main() {
	Run(runSubdomainDiscovery)
}

func runSubdomainDiscovery(ctx context.Context, execution *enginecontract.Execution) error {
	if ctx == nil {
		return fmt.Errorf("execution context is required")
	}
	if !execution.Config.Resolve.Enabled {
		return fmt.Errorf("resolve.enabled must be true")
	}
	wordlistPath, bruteforceResolversPath, resolveResolversPath := enabledConfigResourcePaths(execution.Config)

	runtimeInstance := subdomaindiscoveryruntime.New()
	artifacts, err := runtimeInstance.Execute(
		ctx,
		execution.Target,
		execution.Config,
		execution.Workspace,
		execution.Progress,
		execution.PlatformResources.SubfinderProviderConfig,
		wordlistPath,
		bruteforceResolversPath,
		resolveResolversPath,
	)
	if err != nil {
		return err
	}
	return runtimeInstance.ReportResults(ctx, execution.Progress, execution.Results.Subdomains, artifacts)
}

func enabledConfigResourcePaths(config enginecontract.Config) (wordlistPath, bruteforceResolversPath, resolveResolversPath enginecontract.FilePath) {
	if config.Bruteforce.Enabled {
		wordlistPath = config.Bruteforce.Wordlist
		bruteforceResolversPath = config.Bruteforce.Resolvers
	}
	if config.Resolve.Enabled {
		resolveResolversPath = config.Resolve.Resolvers
	}
	return wordlistPath, bruteforceResolversPath, resolveResolversPath
}
