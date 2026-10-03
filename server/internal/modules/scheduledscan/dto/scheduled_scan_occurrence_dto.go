package dto

import (
	"time"

	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
)

// ScheduledScanOccurrenceListQuery reuses the Scheduled Scan list pagination
// conventions for one Schedule's occurrence history page.
type ScheduledScanOccurrenceListQuery struct {
	httpdto.PaginationQuery
}

// ScheduledScanOccurrenceResponse is the read-only occurrence projection.
type ScheduledScanOccurrenceResponse struct {
	Name           string     `json:"name"`
	ID             int64      `json:"id"`
	ScheduledFor   time.Time  `json:"scheduledFor"`
	AttemptedAt    *time.Time `json:"attemptedAt"`
	DispatchedAt   *time.Time `json:"dispatchedAt"`
	Status         string     `json:"status"`
	FailureKind    *string    `json:"failureKind,omitempty"`
	FailureCause   *string    `json:"failureCause,omitempty"`
	FailureMessage *string    `json:"failureMessage,omitempty"`
	RetryCount     int        `json:"retryCount"`
	NextRetryAt    *time.Time `json:"nextRetryAt,omitempty"`
	// DurationMs is exposed only for succeeded rows; failed rows carry no
	// settlement timestamp, so the field stays absent instead of synthesized.
	DurationMs *int64 `json:"durationMs,omitempty"`
}

// ScheduledScanOccurrenceStatusCounts aggregates every retained occurrence of
// the Schedule by projected status, independent of the returned page.
type ScheduledScanOccurrenceStatusCounts struct {
	Pending     int64 `json:"pending"`
	Dispatching int64 `json:"dispatching"`
	Retrying    int64 `json:"retrying"`
	Succeeded   int64 `json:"succeeded"`
	Failed      int64 `json:"failed"`
}

type ScheduledScanOccurrenceListResponse struct {
	Occurrences  []ScheduledScanOccurrenceResponse     `json:"occurrences"`
	StatusCounts ScheduledScanOccurrenceStatusCounts   `json:"statusCounts"`
	TotalSize    int64                                 `json:"totalSize,omitempty"`
	NextPageToken string                              `json:"nextPageToken,omitempty"`
}

func NewScheduledScanOccurrenceListResponse(
	data []ScheduledScanOccurrenceResponse,
	counts ScheduledScanOccurrenceStatusCounts,
	total int64,
	page, pageSize int,
) ScheduledScanOccurrenceListResponse {
	paginated := httpdto.NewPaginatedResponse(data, total, page, pageSize)
	return ScheduledScanOccurrenceListResponse{
		Occurrences:  paginated.Results,
		StatusCounts: counts,
		TotalSize:    paginated.TotalSize,
		NextPageToken: paginated.NextPageToken,
	}
}
