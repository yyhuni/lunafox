package application

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrInvalidArgument    = errors.New("invalid nuclei poc argument")
	ErrInvalidSourceURL   = errors.New("invalid nuclei poc source URL")
	ErrSyncConflict       = errors.New("nuclei poc sync conflict")
	ErrTaskNotFound       = errors.New("nuclei poc sync task not found")
	ErrSourceNotFound     = errors.New("nuclei poc source not found")
	ErrPOCNotFound        = errors.New("nuclei poc not found")
	ErrRequestExpired     = errors.New("nuclei poc request expired")
	ErrInvalidQuery       = errors.New("invalid nuclei poc query")
	ErrInvalidUpdateMask  = errors.New("invalid nuclei poc update mask")
	ErrSyncAlreadyRunning = errors.New("nuclei poc sync already running")
	ErrSyncFailure        = errors.New("nuclei poc sync failed")
	ErrQuotaExceeded      = errors.New("nuclei poc sync quota exceeded")
	ErrProcessInterrupted = errors.New("nuclei poc process interrupted")
)

type SyncAlreadyRunningError struct{ TaskID uuid.UUID }

func (err *SyncAlreadyRunningError) Error() string {
	return fmt.Sprintf("%v: %s", ErrSyncAlreadyRunning, err.TaskID.String())
}

func (err *SyncAlreadyRunningError) Unwrap() error { return ErrSyncAlreadyRunning }
