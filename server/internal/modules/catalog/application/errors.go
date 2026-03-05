package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

var (
	ErrTargetNotFound       = catalogdomain.ErrTargetNotFound
	ErrTargetExists         = catalogdomain.ErrTargetExists
	ErrInvalidTarget        = catalogdomain.ErrInvalidTarget
	ErrTargetOrgNotFound    = catalogdomain.ErrTargetOrgNotFound
	ErrTargetOrgBindingFail = catalogdomain.ErrTargetOrgBindingFail

	ErrWordlistNotFound = catalogdomain.ErrWordlistNotFound
	ErrWordlistExists   = catalogdomain.ErrWordlistExists
	ErrEmptyName        = catalogdomain.ErrWordlistNameEmpty
	ErrNameTooLong      = catalogdomain.ErrWordlistNameTooLong
	ErrInvalidName      = catalogdomain.ErrWordlistNameInvalid
	ErrFileNotFound     = catalogdomain.ErrWordlistFileNotFound
	ErrInvalidFileType  = catalogdomain.ErrWordlistInvalidFileType
	ErrLineTooLong      = catalogdomain.ErrWordlistLineTooLong

	ErrWorkflowNotFound        = catalogdomain.ErrWorkflowNotFound
	ErrWorkflowProfileNotFound = catalogdomain.ErrWorkflowProfileNotFound
)
