"use client"

import type { ReactNode } from "react"

import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { getStatusToneBgClass, getStatusToneTextClass, type StatusTone } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

interface SegmentedMetricProgressSharedProps {
  variant?: "compact" | "inline" | "panel" | "runtime-card"
  barTone?: "status" | "neutral" | "threshold"
  valueTone?: "status" | "neutral"
  className?: string
  valueClassName?: string
}

interface SegmentedMetricProgressResolvedProps extends SegmentedMetricProgressSharedProps {
  label: string
  value: number
  threshold?: number
  icon?: ReactNode
  detail?: string
  showProgress?: boolean
  loading?: false
}

interface SegmentedMetricProgressLoadingProps extends SegmentedMetricProgressSharedProps {
  loading: true
  variant?: "compact"
}

export type SegmentedMetricProgressProps =
  | SegmentedMetricProgressResolvedProps
  | SegmentedMetricProgressLoadingProps

const SEGMENT_COUNT = 18

const COMPACT_METRIC_ROW_CLASS = "flex min-w-0 items-center gap-1.5 text-[10px]"
const COMPACT_METRIC_LABEL_TRACK_CLASS = "flex w-10 shrink-0 items-center gap-1.5 text-muted-foreground"
const COMPACT_METRIC_BAR_TRACK_CLASS = "flex h-1.5 min-w-0 flex-1 gap-0.5"
const COMPACT_METRIC_VALUE_TRACK_CLASS = "flex w-12 shrink-0 items-center justify-end gap-1 whitespace-nowrap tabular-nums"

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

function getMetricTone(value: number, threshold?: number): StatusTone {
  if (!threshold) return "success"
  if (value >= threshold) return "error"
  if (value >= threshold * 0.8) return "warning"
  return "success"
}

function getMetricTextClass(value: number, threshold?: number) {
  const tone = getMetricTone(value, threshold)
  return tone === "success" ? "text-foreground" : getStatusToneTextClass(tone)
}

function getMetricBarClass(
  barTone: NonNullable<SegmentedMetricProgressSharedProps["barTone"]>,
  value: number,
  threshold?: number,
) {
  if (barTone === "neutral") return "bg-muted-foreground"
  if (barTone === "threshold" && (threshold === undefined || value < threshold)) return "bg-muted-foreground"
  return getStatusToneBgClass(getMetricTone(value, threshold))
}

function SegmentedMetricProgressLoadingState({ className }: Pick<SegmentedMetricProgressLoadingProps, "className">) {
  return (
    <div data-slot="segmented-metric-progress-skeleton" aria-hidden="true" className={cn(COMPACT_METRIC_ROW_CLASS, className)}>
      <div className={COMPACT_METRIC_LABEL_TRACK_CLASS}>
        <span className="truncate">
          <Skeleton className="inline-block h-2.5 w-7 align-middle rounded-full" />
        </span>
      </div>

      <div className={COMPACT_METRIC_BAR_TRACK_CLASS}>
        <Skeleton className="h-full min-w-0 flex-1 rounded-full" />
      </div>

      <div className={COMPACT_METRIC_VALUE_TRACK_CLASS}>
        <span>
          <Skeleton className="inline-block h-2.5 w-9 align-middle rounded-full" />
        </span>
      </div>
    </div>
  )
}

export function SegmentedMetricProgress(props: SegmentedMetricProgressProps) {
  if (props.loading) {
    return <SegmentedMetricProgressLoadingState className={props.className} />
  }

  const {
    label,
    value,
    threshold,
    icon,
    detail,
    showProgress = true,
    variant = "compact",
    barTone = "status",
    valueTone = "status",
    className,
    valueClassName: valueClassNameOverride,
  } = props
  const percentage = clampPercent(value)
  const activeCount = Math.ceil((percentage / 100) * SEGMENT_COUNT)
  const thresholdValue = threshold === undefined ? undefined : clampPercent(threshold)
  const thresholdIndex = thresholdValue === undefined
    ? undefined
    : Math.floor((thresholdValue / 100) * SEGMENT_COUNT)
  const activeColorClass = getMetricBarClass(barTone, percentage, thresholdValue)
  const valueClassName = valueTone === "neutral"
    ? textRole.metadataValueStrong
    : cn(textRole.metadataValueStrong, getStatusToneTextClass(getMetricTone(percentage, thresholdValue)))
  const segments = Array.from({ length: SEGMENT_COUNT }).map((_, index) => {
    const isActive = index < activeCount
    const isThresholdMarker = thresholdIndex !== undefined && index === thresholdIndex

    return (
      <span
        key={index}
        aria-hidden="true"
        className={cn(
          "min-w-0 flex-1 rounded-none",
          isActive
            ? activeColorClass
            : isThresholdMarker
              ? "bg-foreground/30"
              : "bg-muted"
        )}
      />
    )
  })

  if (variant === "panel") {
    return (
      <div className={cn("flex items-center gap-3", className)}>
        <div className="flex size-8 shrink-0 items-center justify-center text-muted-foreground [&_svg]:size-5">
          {icon}
        </div>

        <div className="w-24 min-w-0 space-y-0.5">
          <p className={cn("truncate", textRole.bodyStrong)}>{label}</p>
          {detail ? <p className={cn("truncate", textRole.caption)} title={detail}>{detail}</p> : null}
        </div>

        <div className="flex h-2 min-w-0 flex-1 gap-0.5">
          {segments}
        </div>

        <div className="w-14 shrink-0 text-right tabular-nums">
          <span className={valueClassName}>{percentage.toFixed(0)}%</span>
          {thresholdValue !== undefined ? <span className="block text-muted-foreground/70">/ {thresholdValue}%</span> : null}
        </div>
      </div>
    )
  }

  if (variant === "runtime-card") {
    return (
      <div className={cn("min-w-0 space-y-1", className)}>
        <div className="flex min-w-0 items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            {icon ? (
              <span className="flex size-3.5 shrink-0 items-center justify-center text-muted-foreground [&_svg]:size-3.5">
                {icon}
              </span>
            ) : null}
            <p className={cn("truncate", textRole.caption)}>{label}</p>
          </div>
          {detail ? <p className={cn("shrink-0 tabular-nums text-muted-foreground", textRole.caption)} title={detail}>{detail}</p> : null}
        </div>

        <div className="flex min-w-0 items-center gap-2">
          <div className="flex h-2 min-w-0 flex-1 gap-0.5">
            {segments}
          </div>
          <div className="shrink-0 text-right tabular-nums">
            <span className={valueClassName}>{percentage.toFixed(0)}%</span>
          </div>
        </div>

      </div>
    )
  }

  if (variant === "inline") {
    return (
      <span className={cn("flex min-w-0 flex-1 flex-col gap-1.5", className)}>
        <span className={cn("flex min-w-0 items-center gap-1.5", textRole.caption)}>
          {icon ? <span className="shrink-0 text-muted-foreground [&_svg]:size-3.5">{icon}</span> : null}
          <span className="truncate">{label}</span>
        </span>
        <span className={cn("tabular-nums", valueClassName, valueClassNameOverride)}>{detail ?? `${percentage.toFixed(0)}%`}</span>
        {showProgress ? <Progress value={percentage} aria-label={label} className="h-1 bg-muted" indicatorClassName={activeColorClass} /> : null}
      </span>
    )
  }

  return (
    <div className={cn(COMPACT_METRIC_ROW_CLASS, className)}>
      <div className={COMPACT_METRIC_LABEL_TRACK_CLASS}>
        {icon ? <span className="shrink-0 text-muted-foreground/80 [&_svg]:size-3">{icon}</span> : null}
        <span className="truncate">{label}</span>
      </div>

      <div className={COMPACT_METRIC_BAR_TRACK_CLASS}>
        {segments}
      </div>

      <div className={COMPACT_METRIC_VALUE_TRACK_CLASS}>
        <span className={cn(getMetricTextClass(percentage, thresholdValue))}>{percentage.toFixed(0)}%</span>
        {thresholdValue !== undefined ? <span className="text-muted-foreground/70">/ {thresholdValue}%</span> : null}
      </div>
    </div>
  )
}
