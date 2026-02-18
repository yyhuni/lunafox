package dto

import "time"

type WebsiteSnapshotItem struct {
	URL             string   `json:"url" binding:"required,url"`
	Host            string   `json:"host"`
	Location        string   `json:"location"`
	Title           string   `json:"title"`
	Webserver       string   `json:"webserver"`
	ContentType     string   `json:"contentType"`
	StatusCode      *int     `json:"statusCode"`
	ContentLength   *int     `json:"contentLength"`
	ResponseBody    string   `json:"responseBody"`
	Tech            []string `json:"tech"`
	Vhost           *bool    `json:"vhost"`
	ResponseHeaders string   `json:"responseHeaders"`
}

type BulkUpsertWebsiteSnapshotsRequest struct {
	TargetID int                   `json:"targetId" binding:"required"`
	Websites []WebsiteSnapshotItem `json:"websites" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertWebsiteSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type WebsiteSnapshotListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type WebsiteSnapshotResponse struct {
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
