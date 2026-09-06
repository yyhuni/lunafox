import { cache } from 'react'
import { api } from '@/lib/api-client'
import type { OverviewStats, AssetStatistics, StatisticsHistoryItem, ServerRuntimeMetrics } from '@/types/overview.types'

export const getOverviewStats = cache(async (): Promise<OverviewStats> => {
  const res = await api.get<OverviewStats>('/dashboard/stats/')
  return res.data
})

/**
 * Get asset statistics data (pre-aggregated)
 */
export const getAssetStatistics = cache(async (): Promise<AssetStatistics> => {
  const res = await api.get<AssetStatistics>('/assetStatistics')
  return res.data
})

/**
 * Get statistics history data (for line charts)
 * Note: React.cache() uses Object.is for equality, so pass primitive values
 */
export const getStatisticsHistory = cache(async (days: number): Promise<StatisticsHistoryItem[]> => {
  const res = await api.get<StatisticsHistoryItem[]>('/assetStatistics/history', {
    params: { days }
  })
  return res.data
})

export const getServerRuntimeMetrics = cache(async (): Promise<ServerRuntimeMetrics> => {
  const res = await api.get<ServerRuntimeMetrics>('/admin/system/runtimeMetrics/current')
  return res.data
})
