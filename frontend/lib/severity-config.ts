/**
 * Global severity token configuration.
 * Theme stylesheets own the concrete colors; this file owns shared severity usage.
 */

import { textRole } from "@/lib/typography"

export type SeverityLevel = 'critical' | 'high' | 'medium' | 'low' | 'info'
export const SEVERITY_LEVELS: SeverityLevel[] = ["critical", "high", "medium", "low", "info"]

export type SeverityBadgeVariant =
  | "outline"
  | "severityCritical"
  | "severityHigh"
  | "severityMedium"
  | "severityLow"
  | "severityInfo"

export interface SeverityStyle {
  className: string
  color: string      // solid color for charts/icons
  bgColor: string    // background color
}

export const SEVERITY_COLORS: Record<SeverityLevel, string> = {
  critical: "var(--severity-critical)",
  high: "var(--severity-high)",
  medium: "var(--severity-medium)",
  low: "var(--severity-low)",
  info: "var(--severity-info)",
} as const

export const SEVERITY_BACKGROUND_COLORS: Record<SeverityLevel, string> = {
  critical: "var(--severity-critical-background)",
  high: "var(--severity-high-background)",
  medium: "var(--severity-medium-background)",
  low: "var(--severity-low-background)",
  info: "var(--severity-info-background)",
} as const

export const SEVERITY_TEXT_CLASSNAMES: Record<SeverityLevel, string> = {
  critical: "text-severity-critical",
  high: "text-severity-high",
  medium: "text-severity-medium",
  low: "text-severity-low",
  info: "text-severity-info",
}

// Badge/Tag styles with background, text, and border
export const SEVERITY_STYLES: Record<SeverityLevel, SeverityStyle> = {
  critical: {
    className: "border border-severity-critical-border bg-severity-critical-bg text-severity-critical",
    color: SEVERITY_COLORS.critical,
    bgColor: SEVERITY_BACKGROUND_COLORS.critical,
  },
  high: {
    className: "border border-severity-high-border bg-severity-high-bg text-severity-high",
    color: SEVERITY_COLORS.high,
    bgColor: SEVERITY_BACKGROUND_COLORS.high,
  },
  medium: {
    className: "border border-severity-medium-border bg-severity-medium-bg text-severity-medium",
    color: SEVERITY_COLORS.medium,
    bgColor: SEVERITY_BACKGROUND_COLORS.medium,
  },
  low: {
    className: "border border-severity-low-border bg-severity-low-bg text-severity-low",
    color: SEVERITY_COLORS.low,
    bgColor: SEVERITY_BACKGROUND_COLORS.low,
  },
  info: {
    className: "border border-severity-info-border bg-severity-info-bg text-severity-info",
    color: SEVERITY_COLORS.info,
    bgColor: SEVERITY_BACKGROUND_COLORS.info,
  },
}

export const SEVERITY_VARIANTS: Record<SeverityLevel, Exclude<SeverityBadgeVariant, "outline">> = {
  critical: "severityCritical",
  high: "severityHigh",
  medium: "severityMedium",
  low: "severityLow",
  info: "severityInfo",
}

export const VULNERABILITY_SEVERITY_BADGE_CLASS =
  `h-5 px-1.5 rounded-sm justify-center ${textRole.badge}`

// Card styles for notifications (with hover states)
export const SEVERITY_CARD_STYLES: Record<SeverityLevel, string> = {
  critical: "border-severity-critical-border bg-severity-critical-bg hover:bg-severity-critical-hover",
  high: "border-severity-high-border bg-severity-high-bg hover:bg-severity-high-hover",
  medium: "border-severity-medium-border bg-severity-medium-bg hover:bg-severity-medium-hover",
  low: "border-severity-low-border bg-severity-low-bg hover:bg-severity-low-hover",
  info: "border-severity-info-border bg-severity-info-bg hover:bg-severity-info-hover",
}

// Icon background styles
export const SEVERITY_ICON_BG: Record<SeverityLevel, string> = {
  critical: "bg-severity-critical-bg",
  high: "bg-severity-high-bg",
  medium: "bg-severity-medium-bg",
  low: "bg-severity-low-bg",
  info: "bg-severity-info-bg",
}

// Helper function to get severity style
export function normalizeSeverityLevel(severity: string): SeverityLevel {
  const normalized = severity?.toLowerCase() as SeverityLevel
  return normalized in SEVERITY_VARIANTS ? normalized : "info"
}
export function getSeverityVariant(severity: string): SeverityBadgeVariant {
  const normalized = normalizeSeverityLevel(severity)
  return SEVERITY_VARIANTS[normalized] || "outline"
}
export function getSeverityStyle(severity: string): SeverityStyle {
  const normalized = normalizeSeverityLevel(severity)
  return SEVERITY_STYLES[normalized] || SEVERITY_STYLES.info
}

// Helper function to get severity color
export function getSeverityColor(severity: string): string {
  const normalized = normalizeSeverityLevel(severity)
  return SEVERITY_COLORS[normalized] || SEVERITY_COLORS.info
}
