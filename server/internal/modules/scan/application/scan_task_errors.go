package application

import "errors"

var (
	ErrScanTaskNotFound          = errors.New("scan task not found")
	ErrScanTaskNotOwned          = errors.New("scan task not owned by agent")
	ErrScanTaskInvalidTransition = errors.New("invalid scan task transition")
	ErrScanTaskInvalidUpdate     = errors.New("invalid scan task update")
)
