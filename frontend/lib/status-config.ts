import type { ScanStatus } from "@/types/scan.types"

export type StatusTone = "success" | "warning" | "error" | "info" | "muted"
export type TrendTone = "positive" | "negative" | "neutral"
export type StatusBadgeVariant = "outline" | "warning" | "success" | "error" | "info"
export type RuntimeStatus = "online" | "offline" | "maintenance" | "degraded"
export type SharedScanStatus = ScanStatus | "initiated"
export type StatusMetricToneClassNames = {
  label: string
  dot: string
  value?: string
  footer?: string
}

export const STATUS_TONE_TEXT_CLASSNAMES: Record<StatusTone, string> = {
  success: "text-success",
  warning: "text-warning",
  error: "text-error",
  info: "text-info",
  muted: "text-muted-foreground",
}

export const STATUS_TONE_BG_CLASSNAMES: Record<StatusTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  error: "bg-error",
  info: "bg-info",
  muted: "bg-muted-foreground",
}

export const STATUS_TONE_BADGE_CLASSNAMES: Record<StatusTone, string> = {
  success: "border-success/20 bg-success/10 text-success",
  warning: "border-warning/20 bg-warning/10 text-warning",
  error: "border-error/20 bg-error/10 text-error",
  info: "border-info/20 bg-info/10 text-info",
  muted: "border-border bg-muted/10 text-muted-foreground",
}

export const STATUS_TONE_SURFACE_CLASSNAMES: Record<StatusTone, string> = {
  success: "border border-success/20 bg-success/10",
  warning: "border border-warning/20 bg-warning/10",
  error: "border border-error/20 bg-error/10",
  info: "border border-info/20 bg-info/10",
  muted: "border border-muted/20 bg-muted/10",
}

export const STATUS_TONE_INTERACTIVE_OUTLINE_CLASSNAMES: Record<StatusTone, string> = {
  success: "bg-transparent border-success/40 text-foreground hover:border-success/60 hover:bg-transparent hover:text-foreground active:border-success/70 active:bg-transparent dark:bg-transparent dark:border-success/35 dark:hover:border-success/55 dark:hover:bg-transparent dark:active:border-success/65 dark:active:bg-transparent",
  warning: "bg-transparent border-warning/40 text-foreground hover:border-warning/60 hover:bg-transparent hover:text-foreground active:border-warning/70 active:bg-transparent dark:bg-transparent dark:border-warning/35 dark:hover:border-warning/55 dark:hover:bg-transparent dark:active:border-warning/65 dark:active:bg-transparent",
  error: "bg-transparent border-error/40 text-foreground hover:border-error/60 hover:bg-transparent hover:text-foreground active:border-error/70 active:bg-transparent dark:bg-transparent dark:border-error/35 dark:hover:border-error/55 dark:hover:bg-transparent dark:active:border-error/65 dark:active:bg-transparent",
  info: "bg-transparent border-info/40 text-foreground hover:border-info/60 hover:bg-transparent hover:text-foreground active:border-info/70 active:bg-transparent dark:bg-transparent dark:border-info/35 dark:hover:border-info/55 dark:hover:bg-transparent dark:active:border-info/65 dark:active:bg-transparent",
  muted: "bg-transparent border-border text-foreground hover:border-foreground/25 hover:bg-transparent hover:text-foreground active:border-foreground/35 active:bg-transparent dark:bg-transparent dark:border-input dark:hover:border-foreground/35 dark:hover:bg-transparent dark:active:border-foreground/45 dark:active:bg-transparent",
}

export const STATUS_TONE_COLOR_VARS: Record<StatusTone, string> = {
  success: "var(--success)",
  warning: "var(--warning)",
  error: "var(--error)",
  info: "var(--info)",
  muted: "var(--muted-foreground)",
}

export const TREND_TONE_TEXT_CLASSNAMES: Record<TrendTone, string> = {
  positive: "text-trend-positive",
  negative: "text-trend-negative",
  neutral: "text-trend-neutral",
}

export const TREND_TONE_BADGE_CLASSNAMES: Record<TrendTone, string> = {
  positive: "border-trend-positive/20 bg-trend-positive/10 text-trend-positive",
  negative: "border-trend-negative/20 bg-trend-negative/10 text-trend-negative",
  neutral: "border-muted/20 bg-muted/10 text-trend-neutral",
}

export const TREND_TONE_COLOR_VARS: Record<TrendTone, string> = {
  positive: "var(--trend-positive)",
  negative: "var(--trend-negative)",
  neutral: "var(--trend-neutral)",
}

export const RUNTIME_STATUS_CONFIG: Record<RuntimeStatus, { tone: StatusTone }> = {
  online: { tone: "success" },
  offline: { tone: "error" },
  maintenance: { tone: "muted" },
  degraded: { tone: "warning" },
}

export const SCAN_STATUS_CONFIG: Record<SharedScanStatus, { tone: StatusTone; badgeVariant: StatusBadgeVariant }> = {
  running: { tone: "warning", badgeVariant: "warning" },
  pending: { tone: "info", badgeVariant: "info" },
  succeeded: { tone: "success", badgeVariant: "success" },
  failed: { tone: "error", badgeVariant: "error" },
  cancelled: { tone: "muted", badgeVariant: "outline" },
  initiated: { tone: "warning", badgeVariant: "warning" },
}

export const SCAN_STATUS_STYLES: Record<string, string> = {
  running: getStatusToneBadgeClass(SCAN_STATUS_CONFIG.running.tone),
  cancelled: getStatusToneBadgeClass(SCAN_STATUS_CONFIG.cancelled.tone),
  succeeded: getStatusToneBadgeClass(SCAN_STATUS_CONFIG.succeeded.tone),
  failed: getStatusToneBadgeClass(SCAN_STATUS_CONFIG.failed.tone),
  pending: getStatusToneBadgeClass(SCAN_STATUS_CONFIG.pending.tone),
  initiated: getStatusToneBadgeClass(SCAN_STATUS_CONFIG.initiated.tone),
}

export const SCAN_STATUS_VARIANTS: Record<string, StatusBadgeVariant> = {
  running: SCAN_STATUS_CONFIG.running.badgeVariant,
  cancelled: SCAN_STATUS_CONFIG.cancelled.badgeVariant,
  succeeded: SCAN_STATUS_CONFIG.succeeded.badgeVariant,
  failed: SCAN_STATUS_CONFIG.failed.badgeVariant,
  pending: SCAN_STATUS_CONFIG.pending.badgeVariant,
  initiated: SCAN_STATUS_CONFIG.initiated.badgeVariant,
}

export function getStatusToneTextClass(tone: StatusTone): string {
  return STATUS_TONE_TEXT_CLASSNAMES[tone]
}

export function getStatusToneBgClass(tone: StatusTone): string {
  return STATUS_TONE_BG_CLASSNAMES[tone]
}

export function getStatusToneBadgeClass(tone: StatusTone): string {
  return STATUS_TONE_BADGE_CLASSNAMES[tone]
}

export function getStatusToneSurfaceClass(tone: StatusTone): string {
  return STATUS_TONE_SURFACE_CLASSNAMES[tone]
}

export function getStatusToneInteractiveOutlineClass(tone: StatusTone): string {
  return STATUS_TONE_INTERACTIVE_OUTLINE_CLASSNAMES[tone]
}

export function getStatusToneColorVar(tone: StatusTone): string {
  return STATUS_TONE_COLOR_VARS[tone]
}

export function getTrendToneTextClass(tone: TrendTone): string {
  return TREND_TONE_TEXT_CLASSNAMES[tone]
}

export function getTrendToneBadgeClass(tone: TrendTone): string {
  return TREND_TONE_BADGE_CLASSNAMES[tone]
}

export function getTrendToneColorVar(tone: TrendTone): string {
  return TREND_TONE_COLOR_VARS[tone]
}

export function getRuntimeStatusTone(status: RuntimeStatus): StatusTone {
  return RUNTIME_STATUS_CONFIG[status].tone
}

export function getRuntimeStatusClasses(status: RuntimeStatus): string {
  return getStatusToneBadgeClass(getRuntimeStatusTone(status))
}

export function getScanStatusConfig(status: string): { tone: StatusTone; badgeVariant: StatusBadgeVariant } {
  return SCAN_STATUS_CONFIG[status as SharedScanStatus] ?? { tone: "muted", badgeVariant: "outline" }
}

export function isSharedScanStatus(status: string): status is SharedScanStatus {
  return status in SCAN_STATUS_CONFIG
}

export function getScanStatusClasses(status: string): string {
  return getStatusToneBadgeClass(getScanStatusConfig(status).tone)
}

export function getScanStatusBadgeVariant(status: string): StatusBadgeVariant {
  return getScanStatusConfig(status).badgeVariant
}

export function getScanStatusTextClass(status: string): string {
  return getStatusToneTextClass(getScanStatusConfig(status).tone)
}

export function getScanStatusMetricTone(status: string): StatusMetricToneClassNames {
  const config = getScanStatusConfig(status)
  const textClassName = getStatusToneTextClass(config.tone)

  return { label: textClassName, dot: getStatusToneBgClass(config.tone), footer: textClassName }
}

export function getScanStatusSurfaceClass(status: string): string {
  return getStatusToneSurfaceClass(getScanStatusConfig(status).tone)
}

export function getScanStatusColorVar(status: string): string {
  return getStatusToneColorVar(getScanStatusConfig(status).tone)
}
