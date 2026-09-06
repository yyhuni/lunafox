package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

type WebsiteListQuery struct {
	httpdto.PaginationQuery
	Filter  string `form:"filter"`
	OrderBy string `form:"orderBy" binding:"omitempty"`
}

func (q *WebsiteListQuery) ValidatePageToken() error {
	return nil
}

type WebsiteResponse struct {
	ID              int                       `json:"id"`
	Name            string                    `json:"name"`
	URL             string                    `json:"url"`
	Host            string                    `json:"host"`
	Location        string                    `json:"location"`
	Title           string                    `json:"title"`
	Webserver       string                    `json:"webserver"`
	ContentType     string                    `json:"contentType"`
	StatusCode      *int                      `json:"statusCode"`
	ContentLength   *int                      `json:"contentLength"`
	ResponseBody    string                    `json:"responseBody"`
	Tech            []string                  `json:"tech"`
	Vhost           *bool                     `json:"vhost"`
	ResponseHeaders string                    `json:"responseHeaders"`
	CreatedAt       time.Time                 `json:"createdAt"`
	Screenshot      *WebsiteScreenshotSummary `json:"screenshot,omitempty"`
}

type WebsiteScreenshotSummary struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	StatusCode *int16    `json:"statusCode"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type BatchCreateWebsitesRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,max=5000"`
}

type BatchCreateWebsitesResponse struct {
	CreatedCount int `json:"createdCount"`
}

type WebsiteUpsertItem struct {
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

type BatchUpsertWebsitesRequest struct {
	Websites []WebsiteUpsertItem `json:"websites" binding:"required,min=1,max=5000,dive"`
}

type BatchUpsertWebsitesResponse struct {
	UpsertedCount int `json:"upsertedCount"`
}
