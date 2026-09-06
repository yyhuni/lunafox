export interface OverviewStats {
  totalTargets: number
  totalSubdomains: number
  totalEndpoints: number
  totalVulnerabilities: number
}

/**
 * Asset statistics data (pre-aggregated)
 */
export interface VulnBySeverity {
  critical: number
  high: number
  medium: number
  low: number
  info: number
}

export interface AssetStatistics {
  totalTargets: number
  totalSubdomains: number
  totalIps: number
  totalEndpoints: number
  totalWebsites: number
  totalVulns: number
  totalAssets: number
  runningScans: number
  updatedAt: string
  // Change values
  changeTargets: number
  changeSubdomains: number
  changeIps: number
  changeEndpoints: number
  changeWebsites: number
  changeVulns: number
  changeAssets: number
  // Vulnerability severity distribution
  vulnBySeverity: VulnBySeverity
}

/**
 * Statistics history data (for line charts)
 */
export interface StatisticsHistoryItem {
  date: string
  totalTargets: number
  totalSubdomains: number
  totalIps: number
  totalEndpoints: number
  totalWebsites: number
  totalVulns: number
  totalAssets: number
}

export type ServerRuntimeMetricScope = "host" | "container" | "process" | "runtime"

export interface ServerRuntimeMetricLatest {
  cpu: number
  memory: number
  disk: number
  updatedAt: string
}

export interface ServerRuntimeMetricSeriesPoint extends Record<string, string | number> {
  time: string
  cpu: number
  memory: number
  disk: number
  sampledAt: string
}

export interface ServerRuntimeMetricCapacity {
  cpuCores: number
  memoryTotalGb: number
  diskTotalGb: number
}

export interface ServerRuntimeMetricSource {
  hostname?: string
  diskPath?: string
}

export interface ServerRuntimeMetrics {
  scope: ServerRuntimeMetricScope
  latest: ServerRuntimeMetricLatest
  series: ServerRuntimeMetricSeriesPoint[]
  capacity: ServerRuntimeMetricCapacity
  source: ServerRuntimeMetricSource
  sampleIntervalSeconds: number
  retentionSeconds: number
}
