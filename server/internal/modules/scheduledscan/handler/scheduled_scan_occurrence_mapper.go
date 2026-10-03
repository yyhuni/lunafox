package handler

import (
	scheduledapp "github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/application"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	"github.com/yyhuni/lunafox/server/internal/modules/scheduledscan/dto"
)

func toOccurrenceHistoryQuery(scheduledScanID int, query *dto.ScheduledScanOccurrenceListQuery) scheduledapp.OccurrenceHistoryQuery {
	return scheduledapp.OccurrenceHistoryQuery{
		ScheduledScanID: scheduledScanID,
		Page:            query.GetPage(),
		PageSize:        query.GetPageSize(),
	}
}

func toScheduledScanOccurrenceListOutput(scheduledScanID int, history *scheduledapp.OccurrenceHistory) ([]dto.ScheduledScanOccurrenceResponse, dto.ScheduledScanOccurrenceStatusCounts) {
	if history == nil {
		return []dto.ScheduledScanOccurrenceResponse{}, dto.ScheduledScanOccurrenceStatusCounts{}
	}
	out := make([]dto.ScheduledScanOccurrenceResponse, 0, len(history.Records))
	for index := range history.Records {
		out = append(out, toScheduledScanOccurrenceOutput(scheduledScanID, &history.Records[index]))
	}
	return out, dto.ScheduledScanOccurrenceStatusCounts{
		Pending:     history.Counts.Pending,
		Dispatching: history.Counts.Dispatching,
		Retrying:    history.Counts.Retrying,
		Succeeded:   history.Counts.Succeeded,
		Failed:      history.Counts.Failed,
	}
}

func toScheduledScanOccurrenceOutput(scheduledScanID int, record *scheduledapp.OccurrenceRecord) dto.ScheduledScanOccurrenceResponse {
	return dto.ScheduledScanOccurrenceResponse{
		Name:           httpdto.ScheduledScanOccurrenceName(scheduledScanID, record.ID),
		ID:             record.ID,
		ScheduledFor:   record.ScheduledFor,
		AttemptedAt:    record.AttemptedAt,
		DispatchedAt:   record.DispatchedAt,
		Status:         string(record.Status),
		FailureKind:    record.FailureKind,
		FailureCause:   handoffFailureCauseString(record.FailureCause),
		FailureMessage: record.FailureMessage,
		RetryCount:     record.RetryCount,
		NextRetryAt:    record.NextRetryAt,
		DurationMs:     record.DurationMs,
	}
}

func handoffFailureCauseString(cause *scheduledapp.HandoffFailureCause) *string {
	if cause == nil {
		return nil
	}
	value := string(*cause)
	return &value
}
