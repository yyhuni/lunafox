package domain

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrSourceNotFound        = errors.New("nuclei poc source not found")
	ErrPOCNotFound           = errors.New("nuclei poc not found")
	ErrSyncTaskNotFound      = errors.New("nuclei poc sync task not found")
	ErrRequestReplayConflict = errors.New("nuclei poc request replay conflicts")
	ErrRequestExpired        = errors.New("nuclei poc request expired")
	ErrActiveSyncConflict    = errors.New("nuclei poc sync already active")
	ErrDuplicateTemplateID   = errors.New("duplicate nuclei template id")
	ErrEmptyCandidate        = errors.New("nuclei candidate is empty")
	ErrInvalidPOC            = errors.New("invalid nuclei poc")
)

type ActiveSyncConflictError struct{ TaskID uuid.UUID }

func (err *ActiveSyncConflictError) Error() string {
	return fmt.Sprintf("%v: %s", ErrActiveSyncConflict, err.TaskID.String())
}

func (err *ActiveSyncConflictError) Unwrap() error { return ErrActiveSyncConflict }
