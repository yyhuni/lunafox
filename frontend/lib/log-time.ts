const LOG_TIMESTAMP_PLACEHOLDER = "---- -- -- --:--:--.---"
const SECOND_PRECISION_TIMESTAMP_LENGTH = 19

function pad(value: number, width: number) {
  return String(value).padStart(width, "0")
}

/** Formats log timestamps with calendar date and millisecond precision in local time. */
export function formatLogTimestamp(raw: string): string {
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) {
    return LOG_TIMESTAMP_PLACEHOLDER
  }

  return `${pad(date.getFullYear(), 4)}-${pad(date.getMonth() + 1, 2)}-${pad(date.getDate(), 2)} ${pad(date.getHours(), 2)}:${pad(date.getMinutes(), 2)}:${pad(date.getSeconds(), 2)}.${pad(date.getMilliseconds(), 3)}`
}

/** Formats timestamps in local time with calendar date and second precision. */
export function formatLocalTimestampSeconds(raw: string): string {
  return formatLogTimestamp(raw).slice(0, SECOND_PRECISION_TIMESTAMP_LENGTH)
}
