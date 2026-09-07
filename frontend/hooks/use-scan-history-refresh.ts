"use client"

import * as React from "react"
import { useQueryClient } from "@tanstack/react-query"

import { usePageRefreshTimestamp } from "@/hooks/_shared/use-page-refresh-timestamp"
import { scanKeys, useScanStatistics } from "@/hooks/use-scans"

export const SCAN_HISTORY_AUTO_REFRESH_MS = 3000

export function useScanHistoryRefresh() {
  const queryClient = useQueryClient()
  const { data: scanStatistics } = useScanStatistics()
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const { isMountedRef, lastRefreshedAt, markRefreshCompleted } = usePageRefreshTimestamp()
  const [isPageVisible, setIsPageVisible] = React.useState(true)
  const refreshPromiseRef = React.useRef<Promise<boolean> | null>(null)
  const wasPageHiddenRef = React.useRef(false)

  const hasActiveScans = (scanStatistics?.pending ?? 0) > 0 || (scanStatistics?.running ?? 0) > 0

  React.useEffect(() => {
    const updateVisibility = () => {
      const visible = document.visibilityState === "visible"
      if (!visible) wasPageHiddenRef.current = true
      setIsPageVisible(visible)
    }

    updateVisibility()
    document.addEventListener("visibilitychange", updateVisibility)

    return () => document.removeEventListener("visibilitychange", updateVisibility)
  }, [])

  const refetchActiveQueries = React.useCallback(() => {
    if (refreshPromiseRef.current) return refreshPromiseRef.current

    const refreshPromise = Promise.allSettled([
      queryClient.refetchQueries({
        exact: true,
        queryKey: scanKeys.statistics(),
        type: "active",
      }),
      queryClient.refetchQueries({
        queryKey: scanKeys.lists(),
        type: "active",
      }),
    ]).then(() => true)

    refreshPromiseRef.current = refreshPromise
    void refreshPromise.finally(() => {
      if (refreshPromiseRef.current === refreshPromise) {
        refreshPromiseRef.current = null
      }
    })

    return refreshPromise
  }, [queryClient])

  const refresh = React.useCallback(async () => {
    if (refreshPromiseRef.current) return

    setIsRefreshing(true)

    try {
      await refetchActiveQueries()
      markRefreshCompleted()
    } finally {
      if (isMountedRef.current) setIsRefreshing(false)
    }
  }, [isMountedRef, markRefreshCompleted, refetchActiveQueries])

  React.useEffect(() => {
    if (!isPageVisible || !hasActiveScans) return

    const runAutomaticRefresh = () => {
      void refetchActiveQueries().then(() => markRefreshCompleted())
    }

    if (wasPageHiddenRef.current) {
      wasPageHiddenRef.current = false
      runAutomaticRefresh()
    }

    const intervalId = window.setInterval(runAutomaticRefresh, SCAN_HISTORY_AUTO_REFRESH_MS)
    return () => window.clearInterval(intervalId)
  }, [hasActiveScans, isPageVisible, markRefreshCompleted, refetchActiveQueries])

  return {
    isRefreshing,
    lastRefreshedAt,
    refresh,
  }
}
