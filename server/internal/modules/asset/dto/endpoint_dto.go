package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type EndpointListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *EndpointListQuery) ValidatePageToken() error {
	return nil
}

type EndpointResponse struct {
	ID                       int       `json:"id"`
	TargetID                 int       `json:"targetId"`
	Name                     string    `json:"name"`
	URL                      string    `json:"url"`
	Host                     string    `json:"host"`
	Location                 string    `json:"location"`
	Title                    string    `json:"title"`
	Webserver                string    `json:"webserver"`
	ContentType              string    `json:"contentType"`
	StatusCode               *int      `json:"statusCode"`
	ContentLength            *int      `json:"contentLength"`
	ResponseBody             string    `json:"responseBody"`
	ResponseBodyTruncated    bool      `json:"responseBodyTruncated"`
	Tech                     []string  `json:"tech"`
	Vhost                    *bool     `json:"vhost"`
	ResponseHeaders          string    `json:"responseHeaders"`
	ResponseHeadersTruncated bool      `json:"responseHeadersTruncated"`
	CreatedAt                time.Time `json:"createdAt"`
}

type BatchCreateEndpointsRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,max=5000"`
}

type BatchCreateEndpointsResponse struct {
	CreatedCount int `json:"createdCount"`
}

type EndpointUpsertItem struct {
	URL             string   `json:"url" binding:"required"`
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

type BatchUpsertEndpointsRequest struct {
	Endpoints []EndpointUpsertItem `json:"endpoints" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertEndpointsResponse struct {
	AffectedCount int64 `json:"affectedCount"`
}
