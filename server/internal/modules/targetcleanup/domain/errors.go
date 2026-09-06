package domain

import "errors"

var ErrTargetTombstoneUnavailable = errors.New("target cleanup tombstone is unavailable")
