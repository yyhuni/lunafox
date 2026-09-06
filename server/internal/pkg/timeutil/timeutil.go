package timeutil

import "time"

func ToUTC(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}

func ToUTCPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

func FormatRFC3339NanoUTC(value time.Time) string {
	return ToUTC(value).Format(time.RFC3339Nano)
}
