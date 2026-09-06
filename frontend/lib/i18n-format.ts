// Internationalization formatting utility functions
// Provides localized formatting for dates, numbers, and relative time

import { useFormatter, useNow, useLocale } from 'next-intl'

/**
 * Date formatting Hook
 * Formats dates according to current locale
 */
export function useFormatDate() {
  const format = useFormatter()
  const locale = useLocale()

  return {
    // Format date and time (full format)
    formatDateTime: (date: Date | string | number | null | undefined) => {
      if (!date) return '-'
      const d = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date
      if (isNaN(d.getTime())) return '-'
      return format.dateTime(d, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      })
    },

    // Format date (date only)
    formatDate: (date: Date | string | number | null | undefined) => {
      if (!date) return '-'
      const d = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date
      if (isNaN(d.getTime())) return '-'
      return format.dateTime(d, {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      })
    },

    // Format time (time only)
    formatTime: (date: Date | string | number | null | undefined) => {
      if (!date) return '-'
      const d = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date
      if (isNaN(d.getTime())) return '-'
      return format.dateTime(d, {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
    },

    // Current locale
    locale,
  }
}

/**
 * Relative time formatting Hook
 * Example: "2 hours ago", "3 days ago"
 */
export function useFormatRelativeTime() {
  const format = useFormatter()
  const now = useNow({ updateInterval: 5000 }) // Update every 5 seconds

  return (date: Date | string | number | null | undefined) => {
    if (!date) return '-'
    const d = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date
    if (isNaN(d.getTime())) return '-'
    return format.relativeTime(d, now)
  }
}

/**
 * Heartbeat time display Hook
 *
 * Returns an object with two fields:
 * - `display`: relative time for normal proximity, switches to absolute
 *   date-time once the heartbeat exceeds `thresholdDays` (default 7).
 * - `title`: full absolute date-time (with seconds) for tooltip / title attr.
 *
 * This mirrors the convention used by GitHub's relative-time-element and
 * Red Hat's rh-timestamp: relative for intuition, absolute for precision.
 */
type HeartbeatTimeDensity = 'default' | 'compact'

/**
 * Formats heartbeat time for full and space-constrained status surfaces.
 * Compact mode omits only the current year, never the seconds. A cross-year
 * heartbeat retains its year so stale data is not ambiguous; the title remains
 * precise in every mode.
 */
export function useFormatHeartbeatTime(
  thresholdDays = 7,
  density: HeartbeatTimeDensity = 'default',
) {
  const format = useFormatter()
  const now = useNow({ updateInterval: 5000 })

  return (date: Date | string | number | null | undefined) => {
    if (!date) return { display: '-', title: '-' }
    const d = typeof date === 'string' || typeof date === 'number' ? new Date(date) : date
    if (isNaN(d.getTime())) return { display: '-', title: '-' }

    const diffMs = now.getTime() - d.getTime()
    const thresholdMs = thresholdDays * 24 * 60 * 60 * 1000

    // Precise absolute time for tooltip (includes seconds)
    const absolute = format.dateTime(d, {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })

    // Far-distance: show absolute date-time as main display instead of
    // a low-information relative string like "45 天前"
    if (diffMs > thresholdMs) {
      if (density === 'compact') {
        return {
          display: format.dateTime(d, {
            ...(d.getFullYear() === now.getFullYear() ? {} : { year: 'numeric' }),
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
          }),
          title: absolute,
        }
      }

      return { display: absolute, title: absolute }
    }

    return { display: format.relativeTime(d, now), title: absolute }
  }
}

/**
 * Number formatting Hook
 * Formats numbers according to current locale
 */
export function useFormatNumber() {
  const format = useFormatter()

  return {
    // Format integer (with thousands separator)
    formatInteger: (num: number | null | undefined) => {
      if (num === null || num === undefined) return '-'
      return format.number(num, { maximumFractionDigits: 0 })
    },

    // Format decimal
    formatDecimal: (num: number | null | undefined, decimals = 2) => {
      if (num === null || num === undefined) return '-'
      return format.number(num, { 
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      })
    },

    // Format percentage
    formatPercent: (num: number | null | undefined, decimals = 1) => {
      if (num === null || num === undefined) return '-'
      return format.number(num / 100, { 
        style: 'percent',
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      })
    },

    // Format file size
    formatFileSize: (bytes: number | null | undefined) => {
      if (bytes === null || bytes === undefined) return '-'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      let unitIndex = 0
      let size = bytes
      while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024
        unitIndex++
      }
      return `${format.number(size, { maximumFractionDigits: 2 })} ${units[unitIndex]}`
    },
  }
}
