import type { ScanRecord, StageStatus } from "@/types/scan.types"
import type { ScanProgressData, StageDetail } from "@/components/scan/scan-progress-dialog-types"

export function formatDuration(seconds?: number): string | undefined {
  if (seconds === undefined || seconds === null) return undefined
  if (seconds < 1) return "<1s"
  if (seconds < 60) return `${Math.round(seconds)}s`
  const minutes = Math.floor(seconds / 60)
  const secs = Math.round(seconds % 60)
  return secs > 0 ? `${minutes}m ${secs}s` : `${minutes}m`
}

export function formatDateTime(isoString?: string, locale: string = "zh"): string {
  if (!isoString) return ""
  try {
    const date = new Date(isoString)
    return date.toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    })
  } catch {
    return isoString
  }
}

function getStageResultCount(stageName: string, stats: ScanRecord["cachedStats"]): number | undefined {
  if (!stats) return undefined
  switch (stageName) {
    case "subdomain_discovery":
    case "subdomainDiscovery":
      return stats.subdomainsCount
    case "site_scan":
    case "siteScan":
      return stats.websitesCount
    case "directory_scan":
    case "directoryScan":
      return stats.directoriesCount
    case "url_fetch":
    case "urlFetch":
      return stats.endpointsCount
    case "vuln_scan":
    case "vulnScan":
      return stats.vulnsTotal
    default:
      return undefined
  }
}

const STATUS_PRIORITY: Record<StageStatus, number> = {
  running: 0,
  pending: 1,
  completed: 2,
  failed: 3,
  cancelled: 4,
}

export function buildScanProgressData(scan: ScanRecord): ScanProgressData {
  const stages: StageDetail[] = []

  if (scan.stageProgress) {
    const sortedEntries = Object.entries(scan.stageProgress)
      .toSorted(([, a], [, b]) => {
        const priorityA = STATUS_PRIORITY[a.status] ?? 99
        const priorityB = STATUS_PRIORITY[b.status] ?? 99
        if (priorityA !== priorityB) {
          return priorityA - priorityB
        }
        return (a.order ?? 0) - (b.order ?? 0)
      })

    for (const [stageName, progress] of sortedEntries) {
      const resultCount = progress.status === "completed"
        ? getStageResultCount(stageName, scan.cachedStats)
        : undefined

      stages.push({
        stage: stageName,
        status: progress.status,
        duration: formatDuration(progress.duration),
        detail: progress.detail || progress.error || progress.reason,
        resultCount,
      })
    }
  }

  return {
    id: scan.id,
    target: scan.target,
    engineNames: scan.engineNames || [],
    status: scan.status,
    progress: scan.progress,
    currentStage: scan.currentStage,
    startedAt: scan.createdAt,
    errorMessage: scan.errorMessage,
    stages,
  }
}
