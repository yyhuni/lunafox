"use client"

import * as React from "react"
import { loadOverviewLazySections } from "@/components/overview/overview-sections-dynamic"
import { useAgentClusterSummary, useAgentLocationMap } from "@/hooks/use-agents"
import { useDatabaseHealth } from "@/hooks/use-database-health"
import { useAssetStatistics, useServerRuntimeMetrics, useStatisticsHistory } from "@/hooks/use-overview"
import { useScanStatistics } from "@/hooks/use-scans"
import { useVulnerabilityStats } from "@/hooks/use-vulnerabilities"

export interface OverviewSectionsReadyProbeProps {
  onReady: () => void
}

export function OverviewSectionsReadyProbe({ onReady }: OverviewSectionsReadyProbeProps) {
  const assetStatistics = useAssetStatistics()
  const assetHistory = useStatisticsHistory(7)
  const serverRuntimeMetrics = useServerRuntimeMetrics()
  const scanStatistics = useScanStatistics()
  const agentClusterSummary = useAgentClusterSummary()
  const agentLocationMap = useAgentLocationMap()
  const databaseHealth = useDatabaseHealth()
  const vulnerabilityStatistics = useVulnerabilityStats()
  const [isLazyChunkLoaded, setIsLazyChunkLoaded] = React.useState(false)
  const framesRef = React.useRef<number[]>([])

  React.useEffect(() => {
    let cancelled = false

    loadOverviewLazySections().then(() => {
      if (!cancelled) setIsLazyChunkLoaded(true)
    })

    return () => {
      cancelled = true
    }
  }, [])

  const isResolved = isLazyChunkLoaded && [
    assetStatistics,
    assetHistory,
    serverRuntimeMetrics,
    scanStatistics,
    agentClusterSummary,
    agentLocationMap,
    databaseHealth,
    vulnerabilityStatistics,
  ].every((query) => query.isSuccess || query.isError)

  React.useEffect(() => {
    const pendingFrames = framesRef.current

    if (!isResolved) {
      while (pendingFrames.length > 0) {
        const frame = pendingFrames.pop()
        if (frame !== undefined) window.cancelAnimationFrame(frame)
      }
      return
    }

    let cancelled = false
    const first = window.requestAnimationFrame(() => {
      const second = window.requestAnimationFrame(() => {
        if (!cancelled) onReady()
      })

      pendingFrames.push(second)
    })

    pendingFrames.push(first)

    return () => {
      cancelled = true
      while (pendingFrames.length > 0) {
        const frame = pendingFrames.pop()
        if (frame !== undefined) window.cancelAnimationFrame(frame)
      }
    }
  }, [isResolved, onReady])

  return null
}
