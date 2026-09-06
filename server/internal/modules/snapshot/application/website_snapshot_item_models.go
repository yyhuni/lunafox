package application

type WebsiteSnapshotItem struct {
	URL             string
	Host            string
	Title           string
	StatusCode      *int
	ContentLength   *int
	Location        string
	Webserver       string
	ContentType     string
	Tech            []string
	ResponseBody    string
	Vhost           *bool
	ResponseHeaders string
}

type WebsiteAssetUpsertItem struct {
	URL             string
	Host            string
	Location        string
	Title           string
	Webserver       string
	ContentType     string
	StatusCode      *int
	ContentLength   *int
	ResponseBody    string
	Tech            []string
	Vhost           *bool
	ResponseHeaders string
}
