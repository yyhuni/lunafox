"use client"

import { useMemo } from "react"
import { cn } from "@/lib/utils"
import { Progress } from "@/components/ui/progress"
import { IconAlertTriangle } from "@/components/icons"

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
  const percentage = Math.min(100, Math.max(0, value))

  const status = useMemo(() => {
    if (!threshold) return "normal"
    if (percentage >= threshold) return "critical"
    if (percentage >= threshold * 0.8) return "warning"
    return "normal"
  }, [percentage, threshold])

  const progressColor = useMemo(() => {
    if (status === "critical") return "bg-[var(--error)]"
    if (status === "warning") return "bg-[var(--warning)]"
    return "bg-[var(--success)]"
  }, [status])

  const textColor = useMemo(() => {
    if (status === "critical") return "text-[var(--error)]"
    if (status === "warning") return "text-[var(--warning)]"
    return "text-foreground"
  }, [status])

  return (
    <div className={cn("space-y-1.5", className)}>
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground flex items-center gap-1">
          {label}
          {showWarning && status !== "normal" && (
            <IconAlertTriangle className={cn(
              "h-3 w-3",
              status === "critical" ? "text-[var(--error)]" : "text-[var(--warning)]"
            )} />
          )}
        </span>
        <span className={cn("font-medium tabular-nums", textColor)}>
          {percentage.toFixed(0)}{unit}
        </span>
      </div>
      <Progress
        value={percentage}
        className={cn(
          "h-1.5",
          status === "critical" && "bg-[var(--error)]/20",
          status === "warning" && "bg-[var(--warning)]/20"
        )}
      >
        <div
          className={cn("h-full transition-[width,background-color] duration-300", progressColor)}
          style={{ width: `${percentage}%` }}
        />
      </Progress>
      {threshold && (
        <div className="text-[10px] text-muted-foreground">
          阈值: {threshold}{unit}
        </div>
      )}
    </div>
  )
}
