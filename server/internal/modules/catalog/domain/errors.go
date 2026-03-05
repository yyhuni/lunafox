package domain

import "errors"

var (
	ErrTargetNotFound       = errors.New("target not found")
	ErrTargetExists         = errors.New("target name already exists")
	ErrInvalidTarget        = errors.New("invalid target format")
	ErrTargetOrgNotFound    = errors.New("organization not found")
	ErrTargetOrgBindingFail = errors.New("organization target binding failed")

	ErrWordlistNotFound        = errors.New("wordlist not found")
	ErrWordlistExists          = errors.New("wordlist name already exists")
	ErrWordlistNameEmpty       = errors.New("wordlist name cannot be empty")
	ErrWordlistNameTooLong     = errors.New("wordlist name too long (max 200 characters)")
	ErrWordlistNameInvalid     = errors.New("wordlist name contains invalid characters")
	ErrWordlistFileNotFound    = errors.New("wordlist file not found")
	ErrWordlistInvalidFileType = errors.New("file appears to be binary, only text files are allowed")
	ErrWordlistLineTooLong     = errors.New("wordlist line exceeds maximum length (64KB)")

	ErrWorkerScanNotFound = errors.New("scan not found")
	ErrWorkerToolRequired = errors.New("tool parameter required for provider_config")

	ErrWorkflowNotFound        = errors.New("workflow not found")
	ErrWorkflowProfileNotFound = errors.New("workflow profile not found")
)
