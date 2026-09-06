"use client"

import { useMemo } from "react"
import { useTranslations } from "next-intl"
import { cn } from "@/lib/utils"
import { Progress } from "@/components/ui/progress"
import { IconAlertTriangle } from "@/components/icons"
import {
  getAgentMetricAlertTextClass,
  getAgentMetricBarClass,
  getAgentMetricTextClass,
  getAgentMetricTrackClass,
} from "./agent-status"

interface MetricProgressProps {
  label: string
  value: number
  threshold?: number
  unit?: string
  className?: string
  showWarning?: boolean
}

export function MetricProgress({
  label,
  value,
  threshold,
  unit = "%",
  className,
  showWarning = true,
}: MetricProgressProps) {
  const t = useTranslations("settings.agents")
  const percentage = Math.min(100, Math.max(0, value))
  const progressColor = useMemo(() => getAgentMetricBarClass(percentage, threshold), [percentage, threshold])
  const textColor = useMemo(() => getAgentMetricTextClass(percentage, threshold), [percentage, threshold])
  const alertTextColor = useMemo(() => getAgentMetricAlertTextClass(percentage, threshold), [percentage, threshold])
  const trackClass = useMemo(() => getAgentMetricTrackClass(percentage, threshold), [percentage, threshold])

  return (
    <div className={cn("space-y-1.5", className)}>
      <div className="flex items-center justify-between text-xs">
        <span className="flex gap-1 items-center text-muted-foreground">
          {label}
          {showWarning && trackClass && (
            <IconAlertTriangle className={cn("h-3 w-3", alertTextColor)} />
          )}
        </span>
        <span className={cn("font-medium tabular-nums", textColor)}>
          {percentage.toFixed(0)}{unit}
        </span>
      </div>
      <Progress
        value={percentage}
        className={cn("h-1.5", trackClass)}
        indicatorClassName={cn("duration-300", progressColor)}
      />
      {threshold && (
        <div className="text-[10px] text-muted-foreground">
          {t("metrics.threshold", { value: `${threshold}${unit}` })}
        </div>
      )}
    </div>
  )
}
