"use client"

import * as React from "react"

import { usePageRefreshTimestamp } from "@/hooks/_shared/use-page-refresh-timestamp"

interface UseAgentManagementRefreshOptions {
  refetchCurrent: () => Promise<unknown>
  refetchSummary: () => Promise<unknown>
  refetchFilterOptions: () => Promise<unknown>
}

export function useAgentManagementRefresh({
  refetchCurrent,
  refetchSummary,
  refetchFilterOptions,
}: UseAgentManagementRefreshOptions) {
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const { isMountedRef, lastRefreshedAt, markRefreshCompleted } = usePageRefreshTimestamp()
  const isRefreshingRef = React.useRef(false)

  const refresh = React.useCallback(async () => {
    if (isRefreshingRef.current) return

    isRefreshingRef.current = true
    setIsRefreshing(true)

    try {
      // The timestamp represents a complete visible-content refresh, not only the summary.
      await Promise.allSettled([refetchCurrent(), refetchSummary(), refetchFilterOptions()])
      markRefreshCompleted()
    } finally {
      isRefreshingRef.current = false
      if (isMountedRef.current) setIsRefreshing(false)
    }
  }, [isMountedRef, markRefreshCompleted, refetchCurrent, refetchFilterOptions, refetchSummary])

  return {
    isRefreshing,
    lastRefreshedAt,
    refresh,
  }
}
