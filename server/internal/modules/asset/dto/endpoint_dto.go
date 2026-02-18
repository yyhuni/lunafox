package dto

import "time"

type EndpointListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type EndpointResponse struct {
	ID                int       `json:"id"`
	TargetID          int       `json:"targetId"`
	URL               string    `json:"url"`
	Host              string    `json:"host"`
	Location          string    `json:"location"`
	Title             string    `json:"title"`
	Webserver         string    `json:"webserver"`
	ContentType       string    `json:"contentType"`
	StatusCode        *int      `json:"statusCode"`
	ContentLength     *int      `json:"contentLength"`
	ResponseBody      string    `json:"responseBody"`
	Tech              []string  `json:"tech"`
	Vhost             *bool     `json:"vhost"`
	ResponseHeaders   string    `json:"responseHeaders"`
	CreatedAt         time.Time `json:"createdAt"`
}

type BulkCreateEndpointsRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,max=5000"`
}

type BulkCreateEndpointsResponse struct {
	CreatedCount int `json:"createdCount"`
}

type EndpointUpsertItem struct {
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

type BulkUpsertEndpointsRequest struct {
	Endpoints []EndpointUpsertItem `json:"endpoints" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertEndpointsResponse struct {
	AffectedCount int64 `json:"affectedCount"`
}
