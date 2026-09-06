package application

import "errors"

// ErrAdminNotFound reports that the canonical administrator is absent.
var ErrAdminNotFound = errors.New("administrator account not found")
