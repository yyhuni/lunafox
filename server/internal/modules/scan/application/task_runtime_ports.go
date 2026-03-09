package application

import (
	"errors"

	scandomain "github.com/yyhuni/lunafox/server/internal/modules/scan/domain"
)

var (
	ErrTaskNotFound          = errors.New("scan task not found")
	ErrTaskNotOwned          = errors.New("scan task not owned by agent")
	ErrTaskInvalidTransition = errors.New("invalid scan task transition")
	ErrTaskInvalidUpdate     = errors.New("invalid scan task update")
)

type TaskRecord = scandomain.TaskRecord

type TaskTargetRef = scandomain.TaskTargetRef

type TaskScanRecord = scandomain.TaskScanRecord

type FailureDetail = scandomain.FailureDetail
