"use client"

import { useState, type ReactNode } from "react"
import { useLocale, useTranslations } from "next-intl"
import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Skeleton } from "@/components/ui/skeleton"
import { HardDrive, IconClock, IconCpu, IconDatabase } from "@/components/icons"
import { SegmentedMetricProgress } from "@/components/shared/metrics/segmented-metric-progress"
import { useServerRuntimeMetrics } from "@/hooks/use-overview"
import { getDateLocale } from "@/lib/date-utils"
import { textRole } from "@/lib/typography"
import { shellOverlaySideOffsets } from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"
import type { ServerRuntimeMetricCapacity } from "@/types/overview.types"

type ServerResourceMetricKey = "cpu" | "memory" | "disk"

type ServerResourceMetric = {
  key: ServerResourceMetricKey
  label: string
  value: number
  detail: string
  icon: ReactNode
}

const serverResourceMetricDefinitions: Array<{
  key: ServerResourceMetricKey
  icon: ReactNode
}> = [
  { key: "cpu", icon: <IconCpu className="h-3 w-3" /> },
  { key: "memory", icon: <IconDatabase className="h-3 w-3" /> },
  { key: "disk", icon: <HardDrive className="h-3 w-3" /> },
]

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, Math.round(value)))
}

function getServerResourceMetricDetail(
  metricKey: ServerResourceMetricKey,
  value: number,
  capacity: ServerRuntimeMetricCapacity,
  t: ReturnType<typeof useTranslations<"overview.lazySections.serverResources">>
) {
  if (metricKey === "cpu") {
    return t("details.cpuCores", { count: capacity.cpuCores })
  }

  if (metricKey === "memory") {
    return t("details.memoryUsage", {
      used: ((value / 100) * capacity.memoryTotalGb).toFixed(1),
      total: capacity.memoryTotalGb,
    })
  }

  return t("details.diskUsage", {
    used: Math.round((value / 100) * capacity.diskTotalGb),
    total: Math.round(capacity.diskTotalGb),
  })
}

function formatRuntimeUpdatedAt(value: string | undefined, locale: string) {
  if (!value) return "-"

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"

  return date.toLocaleString(getDateLocale(locale), {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  })
}

function ServerResourcePopoverContent() {
  const locale = useLocale()
  const tNav = useTranslations("navigation")
  const tOverview = useTranslations("overview.lazySections.serverResources")
  const serverRuntimeMetrics = useServerRuntimeMetrics()
  const latestServerResourcePoint = serverRuntimeMetrics.data?.latest
  const capacity = serverRuntimeMetrics.data?.capacity ?? { cpuCores: 0, memoryTotalGb: 0, diskTotalGb: 0 }
  const metrics: ServerResourceMetric[] = serverResourceMetricDefinitions.map((metric) => {
    const value = clampPercent(latestServerResourcePoint?.[metric.key] ?? 0)

    return {
      ...metric,
      label: tOverview(`labels.${metric.key}`),
      value,
      detail: getServerResourceMetricDetail(metric.key, value, capacity, tOverview),
    }
  })

  const updatedAt = formatRuntimeUpdatedAt(latestServerResourcePoint?.updatedAt, locale)
  const isInitialLoading = serverRuntimeMetrics.isLoading && !serverRuntimeMetrics.data
  const isError = serverRuntimeMetrics.isError && !serverRuntimeMetrics.data

  return (
    <div className="space-y-4 p-4">
      <div className="flex items-center justify-between gap-4">
        <div className="flex min-w-0 items-center">
          <p className={cn("truncate", textRole.sectionTitle)}>{tNav("serverResources")}</p>
        </div>
        <div className="flex min-w-0 shrink-0 items-center gap-1.5 text-muted-foreground">
          <IconClock className="size-3.5 shrink-0" />
          <p className={cn("max-w-36 truncate text-right", textRole.caption)} title={updatedAt}>
            {tNav("serverResourcesUpdatedAt", { time: updatedAt })}
          </p>
        </div>
      </div>

      {isError ? (
        <p className={cn("rounded-md bg-destructive/10 px-3 py-2 text-destructive", textRole.helperText)}>
          {tNav("serverResourcesError")}
        </p>
      ) : (
        <div className="divide-y divide-border/60">
          {metrics.map((metric) => (
            <div key={metric.key} className="py-4 first:pt-0 last:pb-0">
              {isInitialLoading ? (
                <div className="flex items-center gap-3">
                  <Skeleton className="size-8 shrink-0" />
                  <div className="w-24 min-w-0 space-y-2">
                    <Skeleton className="h-5 w-12" />
                    <Skeleton className="h-4 w-20" />
                  </div>
                  <Skeleton className="h-2 min-w-0 flex-1" />
                  <Skeleton className="h-5 w-12 shrink-0" />
                </div>
              ) : (
                <SegmentedMetricProgress
                  label={metric.label}
                  value={metric.value}
                  icon={metric.icon}
                  detail={metric.detail}
                  variant="panel"
                  valueTone="neutral"
                />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function ServerResourcePopover() {
  const tNav = useTranslations("navigation")
  const [open, setOpen] = useState(false)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<Button type="button" variant="ghost" size="icon-sm" aria-label={tNav("serverResources")} />}>
        <IconCpu className="h-4 w-4" />
        <span className="sr-only">{tNav("serverResources")}</span>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={shellOverlaySideOffsets.header} collisionPadding={12} className="w-96 p-0 md:w-md">
        {open ? <ServerResourcePopoverContent /> : null}
      </PopoverContent>
    </Popover>
  )
}
