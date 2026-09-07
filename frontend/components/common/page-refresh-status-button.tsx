"use client"

import * as React from "react"

import { semanticIcons } from "@/components/icons"
import { RefreshSpinner } from "@/components/shared/loading/spinner"
import { useRefreshFeedback } from "@/components/shared/loading/use-refresh-feedback"
import { Button } from "@/components/ui/button"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

const RefreshIcon = semanticIcons.action.refresh

function padDatePart(value: number) {
  return String(value).padStart(2, "0")
}

export function formatPageRefreshTimestamp(date: Date) {
  return [
    date.getFullYear(),
    padDatePart(date.getMonth() + 1),
    padDatePart(date.getDate()),
  ].join("/") + " " + [
    padDatePart(date.getHours()),
    padDatePart(date.getMinutes()),
    padDatePart(date.getSeconds()),
  ].join(":")
}

interface PageRefreshStatusButtonProps {
  isRefreshing: boolean
  lastRefreshedAt: Date | null
  onRefresh: () => void | Promise<void>
  refreshLabel: string
  updatedAtLabel: string
  updatedAtUnavailable: string
  display?: "full" | "icon-only"
  className?: string
}

export function PageRefreshStatusButton({
  isRefreshing,
  lastRefreshedAt,
  onRefresh,
  refreshLabel,
  updatedAtLabel,
  updatedAtUnavailable,
  display = "full",
  className,
}: PageRefreshStatusButtonProps) {
  const { isVisible: isRefreshFeedbackVisible, beginManualRefresh } = useRefreshFeedback(isRefreshing)
  const timestamp = lastRefreshedAt
    ? formatPageRefreshTimestamp(lastRefreshedAt)
    : updatedAtUnavailable

  const handleRefresh = React.useCallback(() => {
    if (isRefreshFeedbackVisible) return

    beginManualRefresh()
    void onRefresh()
  }, [beginManualRefresh, isRefreshFeedbackVisible, onRefresh])

  const indicator = isRefreshFeedbackVisible
    ? <RefreshSpinner className="size-4" />
    : <RefreshIcon className="size-4" aria-hidden="true" />

  return (
    <Button
      type="button"
      variant="ghost"
      size={display === "icon-only" ? "icon" : "sm"}
      onClick={handleRefresh}
      disabled={isRefreshFeedbackVisible}
      aria-label={refreshLabel}
      aria-busy={isRefreshFeedbackVisible}
      className={cn("text-muted-foreground hover:text-foreground", className)}
    >
      {display === "full" ? (
        <>
          <span className={cn("whitespace-nowrap", textRole.monoLabel)}>{updatedAtLabel}</span>
          <span className={cn("whitespace-nowrap tabular-nums text-foreground", textRole.monoLabel)}>{timestamp}</span>
          {indicator}
        </>
      ) : (
        <>
          {indicator}
          <span className="sr-only">{refreshLabel}</span>
        </>
      )}
    </Button>
  )
}
