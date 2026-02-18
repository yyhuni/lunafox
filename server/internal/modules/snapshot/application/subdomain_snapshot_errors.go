package application

import "errors"

var ErrSubdomainSnapshotInvalidTargetType = errors.New("target type must be domain for subdomains")
