import { useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"

export function usePrefetchOverviewData() {
  const queryClient = useQueryClient()

  return useCallback(async () => {
    const scansParams = { page: 1, pageSize: 10 }
    const vulnerabilitiesParams = { page: 1, pageSize: 10 }
    const [
      { overviewKeys },
      { agentKeys },
      { scanKeys },
      { vulnerabilityKeys },
      { getAssetStatistics, getStatisticsHistory },
      { agentService },
      { getScans },
      { VulnerabilityService },
    ] = await Promise.all([
      import("@/hooks/use-overview"),
      import("@/hooks/use-agents"),
      import("@/hooks/use-scans/keys"),
      import("@/hooks/use-vulnerabilities/keys"),
      import("@/services/overview.service"),
      import("@/services/agent.service"),
      import("@/services/scan.service"),
      import("@/services/vulnerability.service"),
    ])

    return Promise.allSettled([
      queryClient.prefetchQuery({
        queryKey: overviewKeys.asset.statistics(),
        queryFn: getAssetStatistics,
      }),
      queryClient.prefetchQuery({
        queryKey: overviewKeys.asset.history(7),
        queryFn: () => getStatisticsHistory(7),
      }),
      queryClient.prefetchQuery({
        queryKey: agentKeys.clusterSummary(),
        queryFn: ({ signal }) => agentService.getAgentClusterSummary(signal),
      }),
      queryClient.prefetchQuery({
        queryKey: agentKeys.locationMap(),
        queryFn: ({ signal }) => agentService.getAgentLocationMap(signal),
      }),
      queryClient.prefetchQuery({
        queryKey: scanKeys.list(scansParams),
        queryFn: () => getScans(scansParams),
      }),
      queryClient.prefetchQuery({
        queryKey: vulnerabilityKeys.list(vulnerabilitiesParams),
        queryFn: () => VulnerabilityService.getAllVulnerabilities(vulnerabilitiesParams),
      }),
    ])
  }, [queryClient])
}
