"use client"

import { useLocale, useTranslations } from "next-intl"

import { Info } from "@/components/icons"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useScanStatistics } from "@/hooks/use-scans"

function formatDurationUnit(locale: string, value: number, unit: Intl.NumberFormatOptions["unit"]): string {
  return new Intl.NumberFormat(locale, {
    style: "unit",
    unit,
    unitDisplay: "narrow",
    maximumFractionDigits: 0,
  }).format(value)
}

export function formatScanHistoryRetentionDuration(totalSeconds: number, locale: string): string {
  const seconds = Math.max(0, Math.floor(totalSeconds))
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) {
    return [formatDurationUnit(locale, days, "day"), hours > 0 ? formatDurationUnit(locale, hours, "hour") : null]
      .filter(Boolean)
      .join(" ")
  }
  if (hours > 0) {
    return [formatDurationUnit(locale, hours, "hour"), minutes > 0 ? formatDurationUnit(locale, minutes, "minute") : null]
      .filter(Boolean)
      .join(" ")
  }
  if (minutes > 0) {
    return formatDurationUnit(locale, minutes, "minute")
  }
  return formatDurationUnit(locale, seconds, "second")
}

export function ScanHistoryRetentionSummary() {
  const locale = useLocale()
  const t = useTranslations("scan.history")
  const { data } = useScanStatistics()
  const policy = data?.retentionPolicy

  if (!policy) {
    return null
  }

  const duration = formatScanHistoryRetentionDuration(policy.minimumRetentionSeconds, locale)
  const description = policy.automaticCleanupEnabled
    ? t("retention.automaticCleanupDescription", { duration })
    : t("retention.inactiveCleanupDescription", { duration })

  return (
    <TooltipProvider delay={100}>
      <Tooltip>
        <TooltipTrigger
          render={
            <span
              className="inline-flex shrink-0 cursor-help text-muted-foreground"
              role="img"
              tabIndex={0}
              aria-label={t("retention.title")}
            />
          }
        >
          <Info className="size-4" aria-hidden="true" />
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={8} align="center" className="max-w-sm whitespace-normal text-left">
          {description}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
