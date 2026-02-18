import { useQuery } from '@tanstack/react-query'
import { getDashboardStats, getAssetStatistics, getStatisticsHistory } from '@/services/dashboard.service'

// Query Keys
export const dashboardKeys = {
  all: ['dashboard'] as const,
  stats: () => [...dashboardKeys.all, 'stats'] as const,
  asset: {
    all: () => ['asset'] as const,
    statistics: () => [...dashboardKeys.asset.all(), 'statistics'] as const,
    history: (days: number) => [...dashboardKeys.asset.statistics(), 'history', days] as const,
  },
}

export function useDashboardStats() {
  return useQuery({
    queryKey: dashboardKeys.stats(),
    queryFn: () => getDashboardStats(),
  })
}

/**
 * Get asset statistics data (pre-aggregated)
 */
export function useAssetStatistics() {
  return useQuery({
    queryKey: dashboardKeys.asset.statistics(),
    queryFn: getAssetStatistics,
  })
}

/**
 * Get statistics history data (for line charts)
 */
export function useStatisticsHistory(days: number = 7) {
  return useQuery({
    queryKey: dashboardKeys.asset.history(days),
    queryFn: () => getStatisticsHistory(days),
  })
}
