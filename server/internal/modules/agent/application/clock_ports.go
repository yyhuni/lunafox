package application

import "time"

// Clock provides deterministic time source for services.
type Clock interface {
	NowUTC() time.Time
}
