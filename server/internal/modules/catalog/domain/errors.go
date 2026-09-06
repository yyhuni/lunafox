package domain

import "errors"

var (
	ErrTargetNotFound       = errors.New("target not found")
	ErrTargetExists         = errors.New("target name already exists")
	ErrInvalidTarget        = errors.New("invalid target format")
	ErrTargetOrgNotFound    = errors.New("organization not found")
	ErrTargetOrgBindingFail = errors.New("organization target binding failed")

	ErrWordlistNotFound          = errors.New("wordlist not found")
	ErrWordlistExists            = errors.New("wordlist fileName already exists")
	ErrWordlistFileNameEmpty     = errors.New("wordlist fileName cannot be empty")
	ErrWordlistFileNameTooLong   = errors.New("wordlist fileName too long (max 200 characters)")
	ErrWordlistFileNameInvalid   = errors.New("wordlist fileName contains invalid characters or path syntax")
	ErrWordlistFileNotFound    = errors.New("wordlist file not found")
	ErrWordlistInvalidFileType = errors.New("file appears to be binary, only text files are allowed")
	ErrWordlistLineTooLong     = errors.New("wordlist line exceeds maximum length (64KB)")

	ErrScanWorkflowNotFound = errors.New("scan workflow not found")
	ErrEngineNotFound       = errors.New("engine not found")
)
