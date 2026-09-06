import {
  getStatusToneBadgeClass,
  getStatusToneTextClass,
  type StatusBadgeVariant,
  type StatusTone,
} from "@/lib/status-config"

function isHttpStatusInRange(statusCode: number, min: number, max: number): boolean {
  return statusCode >= min && statusCode < max
}

export function getHttpStatusTone(statusCode: number | null | undefined): StatusTone {
  if (statusCode === null || statusCode === undefined) return "muted"
  if (isHttpStatusInRange(statusCode, 200, 300)) return "success"
  if (isHttpStatusInRange(statusCode, 300, 400)) return "info"
  if (isHttpStatusInRange(statusCode, 400, 500)) return "warning"
  if (statusCode >= 500) return "error"
  return "muted"
}

export function getHttpStatusBadgeVariant(statusCode: number | null | undefined): StatusBadgeVariant {
  const tone = getHttpStatusTone(statusCode)

  if (tone === "muted") return "outline"
  if (tone === "success") return "success"
  if (tone === "info") return "info"
  if (tone === "warning") return "warning"
  return "error"
}

export function getHttpStatusBadgeClassName(statusCode: number | null | undefined): string {
  return getStatusToneBadgeClass(getHttpStatusTone(statusCode))
}

export function getHttpStatusTextClassName(statusCode: number | null | undefined): string {
  return getStatusToneTextClass(getHttpStatusTone(statusCode))
}
