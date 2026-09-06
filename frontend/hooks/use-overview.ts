import { useQuery } from '@tanstack/react-query'
import { getOverviewStats, getAssetStatistics, getStatisticsHistory, getServerRuntimeMetrics } from '@/services/overview.service'

// Query Keys
export const overviewKeys = {
  all: ['overview'] as const,
  stats: () => [...overviewKeys.all, 'stats'] as const,
  runtimeMetrics: () => [...overviewKeys.all, 'runtimeMetrics'] as const,
  asset: {
    all: () => ['asset'] as const,
    statistics: () => [...overviewKeys.asset.all(), 'statistics'] as const,
    history: (days: number) => [...overviewKeys.asset.statistics(), 'history', days] as const,
  },
}

export function useOverviewStats() {
  return useQuery({
    queryKey: overviewKeys.stats(),
    queryFn: () => getOverviewStats(),
  })
}

/**
 * Get asset statistics data (pre-aggregated)
 */
export function useAssetStatistics() {
  return useQuery({
    queryKey: overviewKeys.asset.statistics(),
    queryFn: getAssetStatistics,
  })
}

/**
 * Get statistics history data (for line charts)
 */
export function useStatisticsHistory(days: number) {
  return useQuery({
    queryKey: overviewKeys.asset.history(days),
    queryFn: () => getStatisticsHistory(days),
  })
}

export function useServerRuntimeMetrics() {
  return useQuery({
    queryKey: overviewKeys.runtimeMetrics(),
    queryFn: getServerRuntimeMetrics,
    refetchInterval: 2000,
  })
}
