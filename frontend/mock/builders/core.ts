import type { AssetStatistics, OverviewStats, StatisticsHistoryItem } from "@/types/overview.types"
import type { Organization, OrganizationsResponse } from "@/types/organization.types"
import type { Target, TargetDetail, TargetsResponse } from "@/types/target.types"
import type { GetScansResponse, ScanRecord } from "@/types/scan.types"
import type { GetVulnerabilitiesResponse, Vulnerability, VulnerabilityStatsResponse } from "@/types/vulnerability.types"
import { mockOverviewStats, mockAssetStatistics, getMockStatisticsHistory } from "../data/overview"
import { filterMockOrganizations, mockOrganizations, getMockOrganizations } from "../data/organizations"
import { mockTargets, getMockTargetById, getMockTargets } from "../data/targets"
import { mockScanStatistics, getMockScanById, getMockScans } from "../data/scans"
import { getMockVulnerabilities, getMockVulnerabilityById, getMockVulnerabilityStats, mockVulnerabilities } from "../data/vulnerabilities"
import { getMockScenario, type MockScenarioId } from "../scenarios"

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function mapScenario<T>(happyValue: T, emptyValue: T, stressValue: T, edgeValue: T): T {
  const scenario = getMockScenario()
  if (scenario === "empty") return emptyValue
  if (scenario === "stress") return stressValue
  if (scenario === "edge") return edgeValue
  return happyValue
}

export function buildOverviewStats(): OverviewStats {
  return mapScenario(
    clone(mockOverviewStats),
    {
      totalTargets: 0,
      totalSubdomains: 0,
      totalEndpoints: 0,
      totalVulnerabilities: 0,
    },
    {
      totalTargets: 4821,
      totalSubdomains: 182344,
      totalEndpoints: 915204,
      totalVulnerabilities: 1762,
    },
    {
      totalTargets: 1,
      totalSubdomains: 0,
      totalEndpoints: 99999,
      totalVulnerabilities: 0,
    }
  )
}

export function buildAssetStatistics(): AssetStatistics {
  const happy = clone(mockAssetStatistics)
  const empty: AssetStatistics = {
    totalTargets: 0,
    totalSubdomains: 0,
    totalIps: 0,
    totalEndpoints: 0,
    totalWebsites: 0,
    totalVulns: 0,
    totalAssets: 0,
    runningScans: 0,
    updatedAt: happy.updatedAt,
    changeTargets: 0,
    changeSubdomains: 0,
    changeIps: 0,
    changeEndpoints: 0,
    changeWebsites: 0,
    changeVulns: 0,
    changeAssets: 0,
    vulnBySeverity: {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
      info: 0,
    },
  }
  const stress: AssetStatistics = {
    ...happy,
    totalTargets: 4821,
    totalSubdomains: 182344,
    totalIps: 20483,
    totalEndpoints: 915204,
    totalWebsites: 288771,
    totalVulns: 1762,
    totalAssets: 1419383,
    runningScans: 47,
    changeTargets: 212,
    changeSubdomains: 8234,
    changeIps: 744,
    changeEndpoints: 18121,
    changeWebsites: 9721,
    changeVulns: 134,
    changeAssets: 37312,
    vulnBySeverity: {
      critical: 48,
      high: 229,
      medium: 641,
      low: 612,
      info: 232,
    },
  }
  const edge: AssetStatistics = {
    ...happy,
    totalTargets: 1,
    totalSubdomains: 0,
    totalIps: 0,
    totalEndpoints: 99999,
    totalWebsites: 3,
    totalVulns: 1,
    totalAssets: 100003,
    runningScans: 0,
    changeTargets: 0,
    changeSubdomains: -17,
    changeIps: 0,
    changeEndpoints: 99999,
    changeWebsites: -2,
    changeVulns: 1,
    changeAssets: 99981,
    vulnBySeverity: {
      critical: 0,
      high: 0,
      medium: 0,
      low: 1,
      info: 0,
    },
  }

  return mapScenario(happy, empty, stress, edge)
}

export function buildStatisticsHistory(days: number): StatisticsHistoryItem[] {
  const happy = clone(getMockStatisticsHistory(days))
  if (getMockScenario() === "empty") {
    return happy.map((item) => ({
      ...item,
      totalTargets: 0,
      totalSubdomains: 0,
      totalIps: 0,
      totalEndpoints: 0,
      totalWebsites: 0,
      totalVulns: 0,
      totalAssets: 0,
    }))
  }
  if (getMockScenario() === "stress") {
    return happy.map((item, index) => ({
      ...item,
      totalTargets: item.totalTargets + index * 25,
      totalSubdomains: item.totalSubdomains + index * 1300,
      totalIps: item.totalIps + index * 180,
      totalEndpoints: item.totalEndpoints + index * 6100,
      totalWebsites: item.totalWebsites + index * 2200,
      totalVulns: item.totalVulns + index * 16,
      totalAssets: item.totalAssets + index * 9800,
    }))
  }
  if (getMockScenario() === "edge") {
    return happy.map((item, index) => ({
      ...item,
      totalTargets: Math.max(0, item.totalTargets - index * 2),
      totalSubdomains: index % 2 === 0 ? item.totalSubdomains : item.totalSubdomains - 200,
      totalIps: index % 3 === 0 ? 0 : item.totalIps,
      totalEndpoints: item.totalEndpoints + index * 400,
      totalWebsites: Math.max(0, item.totalWebsites - index * 5),
      totalVulns: index === happy.length - 1 ? 1 : item.totalVulns,
      totalAssets: item.totalAssets + index * 100,
    }))
  }
  return happy
}

export function buildOrganizations(params?: {
  page?: number
  pageSize?: number
  search?: string
  filter?: string
}): OrganizationsResponse<Organization> {
  if (getMockScenario() === "empty") {
    return {
      results: [],
      total: 0,
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      totalPages: 0,
    }
  }

  const base = getMockOrganizations(params)
  if (getMockScenario() === "happy") {
    return clone(base)
  }

  const expanded = clone(mockOrganizations)
  if (getMockScenario() === "stress") {
    expanded.unshift({
      ...expanded[0],
      id: 9991,
      name: "North America Enterprise Shared Security Platform and Acquisition Integration Office",
      description: "Long cross-regional M&A integration and shared security operations description used to validate table density and truncation behavior",
      targetCount: 126,
      domainCount: 4821,
      endpointCount: 195221,
    })
  }

  if (getMockScenario() === "edge") {
    expanded.unshift({
      ...expanded[0],
      id: 9992,
      name: "Edge Org",
      description: "",
      targetCount: 0,
      domainCount: 0,
      endpointCount: 0,
      targets: [],
    })
  }

  const filtered = filterMockOrganizations(expanded, params?.filter ?? params?.search)
  const page = params?.page || 1
  const pageSize = params?.pageSize || 10
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)

  return {
    results,
    total: filtered.length,
    page,
    pageSize,
    totalPages: Math.ceil(filtered.length / pageSize),
  }
}

export function buildTargets(params?: {
  page?: number
  pageSize?: number
  search?: string
  orderBy?: string | null
}): TargetsResponse {
  if (getMockScenario() === "empty") {
    return {
      results: [],
      total: 0,
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      totalPages: 0,
    }
  }

  const base = clone(getMockTargets(params))
  if (getMockScenario() === "happy") {
    return base
  }

  const results = [...base.results]
  if (getMockScenario() === "stress") {
    results.unshift({
      ...clone(mockTargets[0]),
      id: 9911,
      name: "north-america-enterprise-shared-security-platform-and-acquisition-integration.example.com",
      description: "Extremely long target name and description used to validate first-screen list truncation, wrapping, and tooltip handoff.",
      organizations: [
        { id: 41, name: "Global Shared Security Operations and Regional Acquisition Integration Group" },
        { id: 42, name: "Platform Engineering" },
        { id: 43, name: "North America Customer Success and Partner Enablement" },
      ],
    })
  }

  if (getMockScenario() === "edge") {
    results.unshift({
      ...clone(mockTargets[0]),
      id: 9912,
      name: "203.0.113.15",
      type: "ip",
      description: "",
      lastScannedAt: undefined,
      organizations: [],
    } as Target)
  }

  return {
    ...base,
    results,
    total: getMockScenario() === "happy" ? base.total : base.total + 1,
    totalPages: Math.ceil((getMockScenario() === "happy" ? base.total : base.total + 1) / base.pageSize),
  }
}

export function buildTargetDetail(id: number): TargetDetail | undefined {
  const target = getMockTargetById(id)
  if (!target) return undefined
  if (getMockScenario() !== "edge") {
    return clone(target)
  }
  return {
    ...clone(target),
    lastScannedAt: undefined,
    summary: {
      ...target.summary,
      subdomains: 0,
      websites: 0,
      endpoints: target.summary.endpoints,
      ips: 0,
      screenshots: 0,
      vulnerabilities: {
        total: 0,
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
      },
    },
  }
}

export function buildScans(params?: {
  page?: number
  pageSize?: number
  pageToken?: string
  target?: number
  status?: ScanRecord["status"]
  search?: string
  filter?: string
  orderBy?: string
}): GetScansResponse {
  if (getMockScenario() === "empty") {
    return {
      results: [],
      total: 0,
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      totalPages: 0,
    }
  }
  const base = clone(getMockScans(params))
  if (getMockScenario() === "happy") {
    return base
  }
  const results = [...base.results]
  if (getMockScenario() === "stress") {
    results.unshift({
      ...clone(results[0]),
      id: 9913,
      target: {
        id: 9913,
        name: "customer-api-shared-gateway.platform.example.com",
        displayName: "customer-api-shared-gateway.platform.example.com",
        type: "domain",
      },
      plannedEngineIds: [
        "engine.lunafox.subdomain_discovery",
        "engine.lunafox.web_crawling",
        "engine.lunafox.nuclei_vulnerability",
        "engine.lunafox.directory_scan",
        "engine.lunafox.screenshot",
      ],
      status: "running",
      progress: 83,
    } as ScanRecord)
  }
  if (getMockScenario() === "edge") {
    results.unshift({
      ...clone(results[0]),
      id: 9914,
      status: "failed",
      progress: 1,
      errorMessage: "Partial bootstrap failed after queue admission.",
    } as ScanRecord)
  }
  return {
    ...base,
    results,
    total: getMockScenario() === "happy" ? base.total : base.total + 1,
    totalPages: Math.ceil((getMockScenario() === "happy" ? base.total : base.total + 1) / base.pageSize),
  }
}

export function buildScanById(id: number): ScanRecord | undefined {
  return clone(getMockScanById(id))
}

export function buildScanStatistics(): typeof mockScanStatistics {
  const happy = clone(mockScanStatistics)
  const empty: typeof mockScanStatistics = {
    total: 0,
    pending: 0,
    running: 0,
    succeeded: 0,
    failed: 0,
    cancelled: 0,
    totalVulns: 0,
    totalSubdomains: 0,
    totalEndpoints: 0,
    totalWebsites: 0,
    totalAssets: 0,
    retentionPolicy: { minimumRetentionSeconds: 30 * 24 * 60 * 60, automaticCleanupEnabled: true },
  }
  const stress: typeof mockScanStatistics = {
    total: 128,
    pending: 17,
    running: 34,
    succeeded: 48,
    failed: 8,
    cancelled: 21,
    totalVulns: 641,
    totalSubdomains: 88234,
    totalEndpoints: 351902,
    totalWebsites: 67120,
    totalAssets: 507256,
    retentionPolicy: { minimumRetentionSeconds: 30 * 24 * 60 * 60, automaticCleanupEnabled: true },
  }
  const edge: typeof mockScanStatistics = {
    total: 3,
    pending: 0,
    running: 0,
    succeeded: 1,
    failed: 2,
    cancelled: 0,
    totalVulns: 1,
    totalSubdomains: 0,
    totalEndpoints: 99999,
    totalWebsites: 3,
    totalAssets: 100003,
    retentionPolicy: { minimumRetentionSeconds: 30 * 24 * 60 * 60, automaticCleanupEnabled: true },
  }
  return mapScenario(happy, empty, stress, edge)
}

export function buildVulnerabilities(params?: Parameters<typeof getMockVulnerabilities>[0]): GetVulnerabilitiesResponse {
  if (getMockScenario() === "empty") {
    return {
      results: [],
      total: 0,
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
      totalPages: 0,
    }
  }
  const base = clone(getMockVulnerabilities(params))
  if (getMockScenario() === "happy") {
    return base
  }
  const results = [...base.results]
  if (getMockScenario() === "stress") {
    results.unshift({
      ...clone(mockVulnerabilities[0]),
      id: 9915,
      url: "https://north-america-enterprise-shared-security-platform-and-acquisition-integration.example.com/auth/callback?redirect=https%3A%2F%2Fpartner.example.com%2Fdeep%2Fpath",
      vulnType: "open-redirect-chain",
      severity: "high",
      description: "Long URL and high-risk vulnerability used to validate list truncation, detail drawer, and copy interactions.",
      isReviewed: false,
    })
  }
  if (getMockScenario() === "edge") {
    results.unshift({
      ...clone(mockVulnerabilities[0]),
      id: 9916,
      severity: "info",
      cvssScore: undefined,
      description: "",
      rawOutput: {},
      isReviewed: true,
    } as Vulnerability)
  }
  return {
    ...base,
    results,
    total: getMockScenario() === "happy" ? base.total : base.total + 1,
    totalPages: Math.ceil((getMockScenario() === "happy" ? base.total : base.total + 1) / base.pageSize),
  }
}

export function buildVulnerabilityStats(targetId?: number): VulnerabilityStatsResponse {
  const happy = clone(getMockVulnerabilityStats(targetId))
  const empty: VulnerabilityStatsResponse = {
    total: 0,
    pendingCount: 0,
    reviewedCount: 0,
    criticalCount: 0,
    highCount: 0,
    mediumCount: 0,
    lowCount: 0,
    infoCount: 0,
  }
  const stress: VulnerabilityStatsResponse = {
    total: happy.total + 37,
    pendingCount: happy.pendingCount + 23,
    reviewedCount: happy.reviewedCount + 14,
    criticalCount: (happy.criticalCount ?? 0) + 4,
    highCount: (happy.highCount ?? 0) + 11,
    mediumCount: (happy.mediumCount ?? 0) + 10,
    lowCount: (happy.lowCount ?? 0) + 8,
    infoCount: (happy.infoCount ?? 0) + 4,
  }
  const edge: VulnerabilityStatsResponse = {
    total: 1,
    pendingCount: 0,
    reviewedCount: 1,
    criticalCount: 0,
    highCount: 0,
    mediumCount: 0,
    lowCount: 0,
    infoCount: 1,
  }
  return mapScenario(happy, empty, stress, edge)
}

export function buildVulnerabilityById(id: number): Vulnerability | undefined {
  const vulnerability = getMockVulnerabilityById(id)
  return vulnerability ? clone(vulnerability) : undefined
}

export function scenarioReturnsError(scenario: MockScenarioId): boolean {
  return scenario === "error"
}
