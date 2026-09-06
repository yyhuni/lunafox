import { Clock, Loader2, semanticIcons } from "@/components/icons"
export {
  SCAN_STATUS_STYLES,
  SCAN_STATUS_VARIANTS,
} from "@/lib/status-config"
import type { StageStatus } from "@/types/scan.types"

export const STAGE_STATUS_PRIORITY: Record<StageStatus, number> = {
  running: 0,
  pending: 1,
  succeeded: 2,
  skipped: 3,
  failed: 4,
  cancelled: 5,
}

export function formatStageDuration(seconds?: number): string | undefined {
  if (seconds === undefined || seconds === null) return undefined
  if (seconds < 1) return "<1s"
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = Math.floor(seconds / 60)
  const secs = Math.round(seconds % 60)
  return secs > 0 ? `${minutes}m ${secs}s` : `${minutes}m`
}

export function formatDate(dateString: string | undefined, locale: string): string {
  if (!dateString) return "-"
  const localeStr = locale === "zh" ? "zh-CN" : "en-US"
  return new Date(dateString).toLocaleString(localeStr, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function formatDuration(startedAt: string | undefined, completedAt: string | undefined): string {
  if (!startedAt) return "-"
  const start = new Date(startedAt)
  const end = completedAt ? new Date(completedAt) : new Date()
  const diffMs = end.getTime() - start.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const remainingMins = diffMins % 60

  if (diffHours > 0) {
    return `${diffHours}h ${remainingMins}m`
  }
  return `${diffMins}m`
}

export function getStatusIcon(status: string) {
  switch (status) {
    case "succeeded":
      return { icon: semanticIcons.status.success, animate: false }
    case "running":
      return { icon: Loader2, animate: true }
    case "failed":
      return { icon: semanticIcons.status.failed, animate: false }
    case "cancelled":
      return { icon: semanticIcons.status.cancelled, animate: false }
    case "skipped":
      return { icon: Clock, animate: false }
    case "pending":
      return { icon: Loader2, animate: true }
    default:
      return { icon: Clock, animate: false }
  }
}
