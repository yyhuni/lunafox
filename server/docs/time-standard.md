# Time Standard

## Scope

This project standardizes all backend time semantics to UTC.

## Rules

- Persist time as `TIMESTAMPTZ` in PostgreSQL.
- Always set DB connection timezone to UTC (`TimeZone=UTC`).
- Use UTC when writing server-side timestamps.
- Return API/WebSocket timestamps in RFC3339Nano UTC.
- Export CSV timestamps in RFC3339Nano UTC.

## Utility

Use `server/internal/pkg/timeutil` for consistent conversions and formatting:

- `ToUTC(time.Time) time.Time`
- `ToUTCPtr(*time.Time) *time.Time`
- `FormatRFC3339NanoUTC(time.Time) string`
