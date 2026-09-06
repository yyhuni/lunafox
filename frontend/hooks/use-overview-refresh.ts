"use client"

import * as React from "react"
import { useQueryClient } from "@tanstack/react-query"

import { agentKeys } from "@/hooks/use-agents"
import { databaseHealthKeys } from "@/hooks/use-database-health"
import { usePageRefreshTimestamp } from "@/hooks/_shared/use-page-refresh-timestamp"
import { overviewKeys } from "@/hooks/use-overview"
import { scanKeys } from "@/hooks/use-scans"
import { vulnerabilityKeys } from "@/hooks/use-vulnerabilities"

export const overviewRefreshQueryKeys = [
  overviewKeys.asset.statistics(),
  overviewKeys.asset.history(7),
  overviewKeys.runtimeMetrics(),
  scanKeys.statistics(),
  agentKeys.clusterSummary(),
  agentKeys.locationMap(),
  databaseHealthKeys.current(),
  vulnerabilityKeys.stats(),
] as const

export function useOverviewRefresh() {
  const queryClient = useQueryClient()
  const [isRefreshing, setIsRefreshing] = React.useState(false)
  const { isMountedRef, lastRefreshedAt, markRefreshCompleted } = usePageRefreshTimestamp()
  const isRefreshingRef = React.useRef(false)

  const refresh = React.useCallback(async () => {
    if (isRefreshingRef.current) return

    isRefreshingRef.current = true
    setIsRefreshing(true)

    try {
      await Promise.allSettled(
        overviewRefreshQueryKeys.map((queryKey) =>
          queryClient.refetchQueries({
            exact: true,
            queryKey,
            type: "active",
          })
        )
      )

      markRefreshCompleted()
    } finally {
      isRefreshingRef.current = false
      if (isMountedRef.current) setIsRefreshing(false)
    }
  }, [isMountedRef, markRefreshCompleted, queryClient])

  return {
    isRefreshing,
    lastRefreshedAt,
    refresh,
  }
}
