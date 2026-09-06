package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type EndpointSnapshotItem struct {
	URL             string   `json:"url" binding:"required"`
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

type BatchUpsertEndpointSnapshotsRequest struct {
	Target    string                 `json:"target" binding:"required"`
	Endpoints []EndpointSnapshotItem `json:"endpoints" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertEndpointSnapshotsResponse struct {
	SnapshotCount int `json:"snapshotCount"`
	AssetCount    int `json:"assetCount"`
}

type EndpointSnapshotListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *EndpointSnapshotListQuery) ValidatePageToken() error {
	return nil
}

type EndpointSnapshotResponse struct {
	ID              int       `json:"id"`
	ScanID          int       `json:"scanId"`
	Name            string    `json:"name"`
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
