package application

import contractresults "github.com/yyhuni/lunafox/contracts/results"

// HTTPX host and numeric metadata are observations; URL remains the identity
// and scope source even when a probe deliberately changes the Host header.
func validateWebsiteSnapshotItem(item WebsiteSnapshotItem) error {
	if _, err := contractresults.DeriveObservedAssetURLHost(item.URL); err != nil {
		return err
	}
	return contractresults.Validate(contractresults.ResultKindAssetWebsiteV2, contractresults.WebsiteV2{
		URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode,
		ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver,
		ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody,
		ResponseBodyTruncated: item.ResponseBodyTruncated, Vhost: item.Vhost,
		ResponseHeaders: item.ResponseHeaders, ResponseHeadersTruncated: item.ResponseHeadersTruncated,
	})
}

func validateEndpointSnapshotItem(item EndpointSnapshotItem) error {
	if _, err := contractresults.DeriveObservedAssetURLHost(item.URL); err != nil {
		return err
	}
	return contractresults.Validate(contractresults.ResultKindAssetEndpointV2, contractresults.EndpointV2{
		URL: item.URL, Host: item.Host, Title: item.Title, StatusCode: item.StatusCode,
		ContentLength: item.ContentLength, Location: item.Location, Webserver: item.Webserver,
		ContentType: item.ContentType, Tech: item.Tech, ResponseBody: item.ResponseBody,
		ResponseBodyTruncated: item.ResponseBodyTruncated, Vhost: item.Vhost,
		ResponseHeaders: item.ResponseHeaders, ResponseHeadersTruncated: item.ResponseHeadersTruncated,
	})
}
