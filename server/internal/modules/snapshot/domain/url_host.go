package domain

import contractresults "github.com/yyhuni/lunafox/contracts/results"

func ExtractHostFromURL(rawURL string) string {
	host, err := contractresults.DeriveObservedAssetURLHost(rawURL)
	if err != nil {
		return ""
	}
	return host
}
