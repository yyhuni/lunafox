package dto

import "time"

type EndpointSnapshotItem struct {
	URL             string   `json:"url" binding:"required,url"`
	Host            string   `json:"host"`
	Title           string   `json:"title"`
	StatusCode      *int     `json:"statusCode"`
	ContentLength   *int     `json:"contentLength"`
	Location        string   `json:"location"`
	Webserver       string   `json:"webserver"`
	ContentType     string   `json:"contentType"`
	Tech            []string `json:"tech"`
	ResponseBody    string   `json:"responseBody"`
	Vhost           *bool    `json:"vhost"`
	ResponseHeaders string   `json:"responseHeaders"`
}

type BulkUpsertEndpointSnapshotsRequest struct {
	TargetID  int                    `json:"targetId" binding:"required"`
	Endpoints []EndpointSnapshotItem `json:"endpoints" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertEndpointSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type EndpointSnapshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type EndpointSnapshotResponse struct {
	ID              int       `json:"id"`
	ScanID          int       `json:"scanId"`
	URL             string    `json:"url"`
	Host            string    `json:"host"`
	Title           string    `json:"title"`
	StatusCode      *int      `json:"statusCode"`
	ContentLength   *int      `json:"contentLength"`
	Location        string    `json:"location"`
	Webserver       string    `json:"webserver"`
	ContentType     string    `json:"contentType"`
	Tech            []string  `json:"tech"`
	ResponseBody    string    `json:"responseBody"`
	Vhost           *bool     `json:"vhost"`
	ResponseHeaders string    `json:"responseHeaders"`
	CreatedAt       time.Time `json:"createdAt"`
}
