package dto

import "time"

type WebsiteListQuery struct {
	PaginationQuery
	Filter string `form:"filter"`
}

type WebsiteResponse struct {
	ID              int       `json:"id"`
	URL             string    `json:"url"`
	Host            string    `json:"host"`
	Location        string    `json:"location"`
	Title           string    `json:"title"`
	Webserver       string    `json:"webserver"`
	ContentType     string    `json:"contentType"`
	StatusCode      *int      `json:"statusCode"`
	ContentLength   *int      `json:"contentLength"`
	ResponseBody    string    `json:"responseBody"`
	Tech            []string  `json:"tech"`
	Vhost           *bool     `json:"vhost"`
	ResponseHeaders string    `json:"responseHeaders"`
	CreatedAt       time.Time `json:"createdAt"`
}

type BulkCreateWebsitesRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,max=5000,dive,url"`
}

type BulkCreateWebsitesResponse struct {
	CreatedCount int `json:"createdCount"`
}

type WebsiteUpsertItem struct {
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

type BulkUpsertWebsitesRequest struct {
	Websites []WebsiteUpsertItem `json:"websites" binding:"required,min=1,max=5000,dive"`
}

type BulkUpsertWebsitesResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}
