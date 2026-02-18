package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

var (
	ErrEngineNotFound = catalogdomain.ErrEngineNotFound
	ErrEngineExists   = catalogdomain.ErrEngineExists
	ErrInvalidEngine  = catalogdomain.ErrInvalidEngine

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
)
