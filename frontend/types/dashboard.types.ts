export interface DashboardStats {
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
  updatedAt: string | null
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
