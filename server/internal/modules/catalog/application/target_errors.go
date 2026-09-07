package application

import (
	"errors"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

var (
	ErrTargetNotFound              = catalogdomain.ErrTargetNotFound
	ErrTargetExists                = catalogdomain.ErrTargetExists
	ErrInvalidTarget               = catalogdomain.ErrInvalidTarget
	ErrTargetOrgNotFound           = catalogdomain.ErrTargetOrgNotFound
	ErrTargetOrgBindingFail        = catalogdomain.ErrTargetOrgBindingFail
	ErrBatchTransactionUnavailable = errors.New("target batch transaction is not configured")

	ErrUnsupportedTargetFilter  = errors.New("unsupported target filter")
	ErrUnsupportedTargetOrderBy = errors.New("unsupported target orderBy")
	ErrInvalidTargetPageToken   = errors.New("invalid target pageToken")
	// ErrTargetSummaryUnavailable indicates an inconsistent query dependency
	// that returned no aggregate row without an error. It is intentionally
	// distinct from ErrTargetNotFound so callers do not misreport a storage
	// anomaly as a missing target.
	ErrTargetSummaryUnavailable = errors.New("target summary unavailable")
)
