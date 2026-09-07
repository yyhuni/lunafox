package application

import "errors"

var (
	ErrScanNotFoundForSnapshot = errors.New("scan not found")
	ErrTargetMismatch          = errors.New("targetId does not match scan's target")
	ErrInvalidTargetType       = errors.New("target type must be domain for subdomains")
)
