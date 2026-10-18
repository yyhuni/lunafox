package application

type EndpointSnapshotItem struct {
	URL                      string
	Host                     string
	Title                    string
	StatusCode               *int64
	ContentLength            *int64
	Location                 string
	Webserver                string
	ContentType              string
	Tech                     []string
	ResponseBody             string
	ResponseBodyTruncated    bool
	Vhost                    *bool
	ResponseHeaders          string
	ResponseHeadersTruncated bool
}

type EndpointAssetUpsertItem struct {
	URL                      string
	Host                     string
	Location                 string
	Title                    string
	Webserver                string
	ContentType              string
	StatusCode               *int64
	ContentLength            *int64
	ResponseBody             string
	ResponseBodyTruncated    bool
	Tech                     []string
	Vhost                    *bool
	ResponseHeaders          string
	ResponseHeadersTruncated bool
}
