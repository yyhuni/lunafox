import type { ScheduledScan } from "@/types/scheduled-scan.types"

export interface ScheduledScanSummary {
  todayRuns: number
  next24hRuns: number
  enabledCount: number
  pausedCount: number
}

export interface ScheduledScanInsights {
  summary: ScheduledScanSummary
  timeline: ScheduledScan[]
}

const DAY_MS = 24 * 60 * 60 * 1000

const isSameLocalDay = (date: Date, reference: Date) =>
  date.getFullYear() === reference.getFullYear() &&
  date.getMonth() === reference.getMonth() &&
  date.getDate() === reference.getDate()

const parseNextRun = (scan: ScheduledScan) => {
  if (!scan.nextRunTime) return null
  const date = new Date(scan.nextRunTime)
  return Number.isNaN(date.getTime()) ? null : date
}

export function buildScheduledScanInsights(
  scans: ScheduledScan[],
  referenceDate: Date = new Date()
): ScheduledScanInsights {
  const timeline = scans
    .filter((scan) => parseNextRun(scan) !== null)
    .sort((a, b) => {
      const left = parseNextRun(a)?.getTime() ?? 0
      const right = parseNextRun(b)?.getTime() ?? 0
      return left - right
    })
    .slice(0, 5)

  const referenceTime = referenceDate.getTime()
  const summary = scans.reduce<ScheduledScanSummary>(
    (acc, scan) => {
      if (scan.isEnabled) {
        acc.enabledCount += 1
      } else {
        acc.pausedCount += 1
      }

      const nextRun = parseNextRun(scan)
      if (!nextRun) return acc

      if (isSameLocalDay(nextRun, referenceDate)) {
        acc.todayRuns += 1
      }

      const nextRunTime = nextRun.getTime()
      if (nextRunTime >= referenceTime && nextRunTime < referenceTime + DAY_MS) {
        acc.next24hRuns += 1
      }

      return acc
    },
    {
      todayRuns: 0,
      next24hRuns: 0,
      enabledCount: 0,
      pausedCount: 0,
    }
  )

  return {
    summary,
    timeline,
  }
}
