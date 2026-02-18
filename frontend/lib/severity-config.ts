/**
 * Global severity color configuration
 * Color progression: cool (info) → warm (low/medium) → hot (high/critical)
 * 
 * Used for: vulnerabilities, notifications, fingerprints, etc.
 */

export type SeverityLevel = 'critical' | 'high' | 'medium' | 'low' | 'info'

export interface SeverityStyle {
  className: string
  color: string      // solid color for charts/icons
  bgColor: string    // background color
}

// Core color values (for charts, icons, etc.)
export const SEVERITY_COLORS = {
  critical: '#9b1c31',
  high: '#dc2626',
  medium: '#f97316',
  low: '#eab308',
  info: '#6b7280',
} as const

// Dark mode text colors
export const SEVERITY_COLORS_DARK = {
  critical: '#f87171',
  high: '#f87171',
  medium: '#fb923c',
  low: '#facc15',
  info: '#9ca3af',
} as const

// Badge/Tag styles with background, text, and border
export const SEVERITY_STYLES: Record<SeverityLevel, SeverityStyle> = {
  critical: {
    className: 'bg-[#9b1c31]/10 text-[#9b1c31] border border-[#9b1c31]/20 dark:bg-[#9b1c31]/20 dark:text-[#f87171]',
    color: SEVERITY_COLORS.critical,
    bgColor: 'rgba(155, 28, 49, 0.1)',
  },
  high: {
    className: 'bg-[#dc2626]/10 text-[#dc2626] border border-[#dc2626]/20 dark:text-[#f87171]',
    color: SEVERITY_COLORS.high,
    bgColor: 'rgba(220, 38, 38, 0.1)',
  },
  medium: {
    className: 'bg-[#f97316]/10 text-[#ea580c] border border-[#f97316]/20 dark:text-[#fb923c]',
    color: SEVERITY_COLORS.medium,
    bgColor: 'rgba(249, 115, 22, 0.1)',
  },
  low: {
    className: 'bg-[#eab308]/10 text-[#ca8a04] border border-[#eab308]/20 dark:text-[#facc15]',
    color: SEVERITY_COLORS.low,
    bgColor: 'rgba(234, 179, 8, 0.1)',
  },
  info: {
    className: 'bg-[#6b7280]/10 text-[#6b7280] border border-[#6b7280]/20 dark:text-[#9ca3af]',
    color: SEVERITY_COLORS.info,
    bgColor: 'rgba(107, 114, 128, 0.1)',
  },
}

// Card styles for notifications (with hover states)
export const SEVERITY_CARD_STYLES: Record<SeverityLevel, string> = {
  critical: 'border-[#9b1c31]/30 bg-[#9b1c31]/5 hover:bg-[#9b1c31]/10 dark:border-[#f87171]/30 dark:bg-[#f87171]/5 dark:hover:bg-[#f87171]/10',
  high: 'border-[#dc2626]/30 bg-[#dc2626]/5 hover:bg-[#dc2626]/10 dark:border-[#f87171]/30 dark:bg-[#f87171]/5 dark:hover:bg-[#f87171]/10',
  medium: 'border-[#f97316]/30 bg-[#f97316]/5 hover:bg-[#f97316]/10 dark:border-[#fb923c]/30 dark:bg-[#fb923c]/5 dark:hover:bg-[#fb923c]/10',
  low: 'border-[#eab308]/30 bg-[#eab308]/5 hover:bg-[#eab308]/10 dark:border-[#facc15]/30 dark:bg-[#facc15]/5 dark:hover:bg-[#facc15]/10',
  info: 'border-[#6b7280]/30 bg-[#6b7280]/5 hover:bg-[#6b7280]/10 dark:border-[#9ca3af]/30 dark:bg-[#9ca3af]/5 dark:hover:bg-[#9ca3af]/10',
}

// Icon background styles
export const SEVERITY_ICON_BG: Record<SeverityLevel, string> = {
  critical: 'bg-[#9b1c31]/10 dark:bg-[#f87171]/10',
  high: 'bg-[#dc2626]/10 dark:bg-[#f87171]/10',
  medium: 'bg-[#f97316]/10 dark:bg-[#fb923c]/10',
  low: 'bg-[#eab308]/10 dark:bg-[#facc15]/10',
  info: 'bg-muted',
}

// Helper function to get severity style
export function getSeverityStyle(severity: string): SeverityStyle {
  const normalized = severity?.toLowerCase() as SeverityLevel
  return SEVERITY_STYLES[normalized] || SEVERITY_STYLES.info
}

// Helper function to get severity color
export function getSeverityColor(severity: string): string {
  const normalized = severity?.toLowerCase() as SeverityLevel
  return SEVERITY_COLORS[normalized] || SEVERITY_COLORS.info
}
