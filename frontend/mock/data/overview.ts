import type { AssetStatistics, StatisticsHistoryItem, OverviewStats, ServerRuntimeMetrics } from '@/types/overview.types'

export const mockOverviewStats: OverviewStats = {
  totalTargets: 156,
  totalSubdomains: 4823,
  totalEndpoints: 12456,
  totalVulnerabilities: 89,
}

export const mockAssetStatistics: AssetStatistics = {
  totalTargets: 156,
  totalSubdomains: 4823,
  totalIps: 892,
  totalEndpoints: 12456,
  totalWebsites: 3421,
  totalVulns: 89,
  totalAssets: 21638,
  runningScans: 5,
  updatedAt: new Date().toISOString(),
  // change value
  changeTargets: 12,
  changeSubdomains: 234,
  changeIps: 45,
  changeEndpoints: 567,
  changeWebsites: 89,
  changeVulns: 15,
  changeAssets: 942,
  // Vulnerability severity distribution
  vulnBySeverity: {
    critical: 3,
    high: 12,
    medium: 28,
    low: 34,
    info: 12,
  },
}

// Generate historical data for the past N days
function generateHistoryData(days: number): StatisticsHistoryItem[] {
  const data: StatisticsHistoryItem[] = []
  const now = new Date()
  
  for (let i = days - 1; i >= 0; i--) {
    const date = new Date(now)
    date.setDate(date.getDate() - i)
    
    // Simulate a gradual growth trend
    const factor = 1 + (days - i) * 0.02
    
    data.push({
      date: date.toISOString().split('T')[0],
      totalTargets: Math.floor(140 * factor),
      totalSubdomains: Math.floor(4200 * factor),
      totalIps: Math.floor(780 * factor),
      totalEndpoints: Math.floor(10800 * factor),
      totalWebsites: Math.floor(2980 * factor),
      totalVulns: Math.floor(75 * factor),
      totalAssets: Math.floor(18900 * factor),
    })
  }
  
  return data
}

export const mockStatisticsHistory7Days = generateHistoryData(7)
export const mockStatisticsHistory30Days = generateHistoryData(30)

export const mockServerRuntimeMetrics: ServerRuntimeMetrics = {
  scope: "runtime",
  latest: {
    cpu: 29,
    memory: 59,
    disk: 49,
    updatedAt: "2026-07-04T10:31:00Z",
  },
  series: [
    { time: "10:22", cpu: 28, memory: 58, disk: 47, sampledAt: "2026-07-04T10:22:00Z" },
    { time: "10:23", cpu: 34, memory: 60, disk: 48, sampledAt: "2026-07-04T10:23:00Z" },
    { time: "10:24", cpu: 41, memory: 62, disk: 48, sampledAt: "2026-07-04T10:24:00Z" },
    { time: "10:25", cpu: 33, memory: 64, disk: 49, sampledAt: "2026-07-04T10:25:00Z" },
    { time: "10:26", cpu: 25, memory: 61, disk: 48, sampledAt: "2026-07-04T10:26:00Z" },
    { time: "10:27", cpu: 30, memory: 63, disk: 49, sampledAt: "2026-07-04T10:27:00Z" },
    { time: "10:28", cpu: 47, memory: 66, disk: 50, sampledAt: "2026-07-04T10:28:00Z" },
    { time: "10:29", cpu: 58, memory: 65, disk: 49, sampledAt: "2026-07-04T10:29:00Z" },
    { time: "10:30", cpu: 36, memory: 62, disk: 48, sampledAt: "2026-07-04T10:30:00Z" },
    { time: "10:31", cpu: 29, memory: 59, disk: 49, sampledAt: "2026-07-04T10:31:00Z" },
  ],
  capacity: {
    cpuCores: 4,
    memoryTotalGb: 16,
    diskTotalGb: 480,
  },
  source: {
    hostname: "mock-server",
    diskPath: "/data/lunafox",
  },
  sampleIntervalSeconds: 2,
  retentionSeconds: 600,
}

export function getMockStatisticsHistory(days: number): StatisticsHistoryItem[] {
  if (days <= 7) return mockStatisticsHistory7Days
  return generateHistoryData(days)
}
