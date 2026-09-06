"use client"

import { memo } from "react"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export type StatMetricTone = {
  label: string
  dot: string
  value?: string
  footer?: string
}

export type StatMetricRowItem = {
  key: string
  title: string
  value: string | number
  footer: string
  tone: StatMetricTone
  icon?: React.ReactNode
  loading?: boolean
}

const metricGridColumns: Record<number, string> = {
  1: "@xl/main:grid-cols-1",
  2: "@xl/main:grid-cols-2",
  3: "@xl/main:grid-cols-3",
  4: "@xl/main:grid-cols-4",
  5: "@xl/main:grid-cols-5",
  6: "@xl/main:grid-cols-6",
}

type StatMetricProps = Omit<StatMetricRowItem, "key"> & {
  featured: boolean
}

function MetricValueSkeleton({ featured = false }: { featured?: boolean }) {
  return (
    <span
      aria-hidden="true"
      data-featured={featured ? "true" : undefined}
      className={cn(
        textRole.metricValueDisplay,
        "relative inline-block select-none text-transparent",
        featured ? "w-6" : "w-4"
      )}
    >
      0
      <span
        data-slot="skeleton"
        className={cn(
          "loading-skeleton !absolute block left-0 top-1/2 -translate-y-1/2 rounded-md",
          featured ? "h-7 w-6" : "h-5 w-4"
        )}
      />
    </span>
  )
}

const StatMetric = memo(function StatMetric({
  title,
  value,
  icon,
  footer,
  tone,
  featured,
  loading,
}: StatMetricProps) {
  if (featured) {
    return (
      <div className="flex min-h-22 items-center gap-4 px-4 py-5 @xl/main:justify-center @xl/main:px-8">
        <span className={cn("size-3 rounded-full", tone.dot)} aria-hidden="true" />
        <div className="min-w-0">
          <div className="flex items-baseline gap-3">
            {loading ? (
              <MetricValueSkeleton featured />
            ) : (
              <span
                data-featured="true"
                className={cn(textRole.metricValueDisplay, tone.value ?? tone.label ?? "text-foreground")}
              >
                {typeof value === "number" ? value.toLocaleString() : value}
              </span>
            )}
            <span className={cn(textRole.sectionTitle, tone.label)}>{title}</span>
          </div>
          <div className={cn("mt-3 truncate", textRole.bodySubtle, tone.footer)}>{footer}</div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-22 flex-col justify-center px-4 py-4 @xl/main:px-6">
      <div className={cn("flex min-w-0 items-center gap-2", textRole.helperText, tone.label)}>
        {icon ? <span className="shrink-0 [&_svg]:size-4">{icon}</span> : null}
        <span className="truncate">{title}</span>
      </div>
      <div className="mt-3">
        {loading ? (
          <MetricValueSkeleton />
        ) : (
          <span className={cn(textRole.metricValueDisplay, tone.value ?? "text-foreground")}>
            {typeof value === "number" ? value.toLocaleString() : value}
          </span>
        )}
      </div>
      <div className={cn("mt-3 flex min-w-0 items-center gap-2", textRole.helperText, tone.footer)}>
        <span className={cn("size-1.5 rounded-full", tone.dot)} aria-hidden="true" />
        <span className="truncate">{footer}</span>
      </div>
    </div>
  )
})

export function StatMetricRow({
  items,
  featuredKey,
  className,
  dataSlot,
}: {
  items: StatMetricRowItem[]
  featuredKey?: string
  className?: string
  dataSlot?: string
}) {
  return (
    <div
      data-slot={dataSlot}
      className={cn(
        "grid grid-cols-1 divide-y divide-border border-y border-border text-foreground @xl/main:divide-x @xl/main:divide-y-0",
        metricGridColumns[items.length] ?? "@xl/main:grid-cols-6",
        className
      )}
    >
      {items.map(({ key, ...item }) => (
        <StatMetric key={key} {...item} featured={key === featuredKey} />
      ))}
    </div>
  )
}
