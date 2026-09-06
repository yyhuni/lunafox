package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	scanapp "github.com/yyhuni/lunafox/server/internal/modules/scan/application"
	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

const maxHandoffFailureMessageBytes = 2000

type NormalScanBatchCreator interface {
	CreateBatch(context.Context, *scanapp.CreateBatchRequest) (*scanapp.BatchScanResult, error)
}

type OccurrenceDispatcher struct {
	creator NormalScanBatchCreator
}

func NewOccurrenceDispatcher(creator NormalScanBatchCreator) *OccurrenceDispatcher {
	return &OccurrenceDispatcher{creator: creator}
}

func (dispatcher *OccurrenceDispatcher) Dispatch(ctx context.Context, input FrozenDispatchInput) (*scanapp.BatchScanResult, error) {
	if dispatcher == nil || dispatcher.creator == nil {
		return nil, ErrScheduledScanInvalidArgument
	}
	if input.TargetScoped && len(input.TargetIDs) != 1 {
		return nil, fmt.Errorf("%w: target-scoped attempt requires exactly one frozen Target", ErrScheduledScanInvalidArgument)
	}
	if !input.InputSource.Valid() {
		return nil, fmt.Errorf("%w: %w", ErrScheduledScanInvalidArgument, scandomain.ErrInvalidInputSource)
	}
	requests := make([]scanapp.CreateBatchItem, 0, len(input.TargetIDs))
	for _, targetID := range input.TargetIDs {
		if targetID <= 0 {
			return nil, fmt.Errorf("%w: frozen Target is required", ErrScheduledScanInvalidArgument)
		}
		requests = append(requests, scanapp.CreateBatchItem{TargetID: targetID})
	}
	request := &scanapp.CreateBatchRequest{
		ScanWorkflow:  resourcenames.ScanWorkflow(input.ScanWorkflowID),
		Configuration: cloneMap(input.Configuration),
		Requests:      requests,
		AgentID:       cloneIntPtr(input.AgentID),
		InputSource:   input.InputSource,
		TriggerType:   scanapp.ScanTriggerTypeScheduled,
	}
	return dispatcher.creator.CreateBatch(ctx, request)
}

func ClassifyHandoff(result *scanapp.BatchScanResult, err error, targetScoped bool) HandoffOutcome {
	created, failed, skipped := 0, 0, 0
	if result != nil {
		created = result.CreatedCount
		failed = len(result.Failed)
		skipped = len(result.Skipped)
	}
	if err == nil && result != nil && created > 0 && created == len(result.Scans) && failed == 0 && skipped == 0 && (!targetScoped || created == 1) {
		return HandoffOutcome{Kind: HandoffCompleted}
	}
	if created > 0 && (failed > 0 || skipped > 0 || err != nil || created != len(result.Scans) || (targetScoped && created != 1)) {
		return safeHandoffOutcome(HandoffPartial, fmt.Sprintf("Scan creation completed partially: %d created, %d failed, %d skipped.", created, failed, skipped))
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return safeHandoffOutcome(HandoffDeadlineExceeded, "Scan creation reached the five-minute handoff deadline.")
	case errors.Is(err, context.Canceled):
		return safeHandoffOutcome(HandoffCanceled, "Scan creation was canceled before handoff completed.")
	}
	return safeHandoffOutcome(HandoffScanCreateFailed, "Scan creation did not complete.")
}

func safeHandoffOutcome(kind HandoffOutcomeKind, message string) HandoffOutcome {
	message = strings.TrimSpace(strings.ToValidUTF8(message, ""))
	if len(message) > maxHandoffFailureMessageBytes {
		message = message[:maxHandoffFailureMessageBytes]
		for !utf8.ValidString(message) {
			message = message[:len(message)-1]
		}
	}
	return HandoffOutcome{Kind: kind, Message: message}
}
