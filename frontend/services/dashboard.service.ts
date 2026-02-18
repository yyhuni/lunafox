import { cache } from 'react'
import { api } from '@/lib/api-client'
import type { DashboardStats, AssetStatistics, StatisticsHistoryItem } from '@/types/dashboard.types'
import { USE_MOCK, mockDelay, mockDashboardStats, mockAssetStatistics, getMockStatisticsHistory } from '@/mock'

export const getDashboardStats = cache(async (): Promise<DashboardStats> => {
  if (USE_MOCK) {
    await mockDelay()
    return mockDashboardStats
  }
  const res = await api.get<DashboardStats>('/dashboard/stats/')
  return res.data
})

/**
 * Get asset statistics data (pre-aggregated)
 */
export const getAssetStatistics = cache(async (): Promise<AssetStatistics> => {
  if (USE_MOCK) {
    await mockDelay()
    return mockAssetStatistics
  }
  const res = await api.get<AssetStatistics>('/assets/statistics/')
  return res.data
})

/**
 * Get statistics history data (for line charts)
 * Note: React.cache() uses Object.is for equality, so pass primitive values
 */
export const getStatisticsHistory = cache(async (days: number = 7): Promise<StatisticsHistoryItem[]> => {
  if (USE_MOCK) {
    await mockDelay()
    return getMockStatisticsHistory(days)
  }
  const res = await api.get<StatisticsHistoryItem[]>('/assets/statistics/history/', {
    params: { days }
  })
  return res.data
})
