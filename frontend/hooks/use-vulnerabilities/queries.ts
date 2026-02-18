import { useQuery, keepPreviousData } from "@tanstack/react-query"
import { normalizePagination } from "@/hooks/_shared/pagination"
import { VulnerabilityService } from "@/services/vulnerability.service"
import type {
  Vulnerability,
  VulnerabilitySeverity,
  GetVulnerabilitiesParams,
} from "@/types/vulnerability.types"
import type { PaginationInfo } from "@/types/common.types"
import { vulnerabilityKeys } from "./keys"

type VulnerabilityResponseItem = Partial<Vulnerability> & {
  severity?: VulnerabilitySeverity | "unknown"
  cvssScore?: number | string | null
  createdAt?: string
  reviewedAt?: string | null
  isReviewed?: boolean
  rawOutput?: Record<string, unknown>
  source?: string
  vulnType?: string
  url?: string
  description?: string
  target?: number
}

type VulnerabilityListResponse = {
  results?: VulnerabilityResponseItem[]
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
  page_size?: number
  total_pages?: number
}

const DEFAULT_VULNERABILITY_PARAMS = {
  page: 1,
  pageSize: 10,
} as const

function buildVulnerabilityParams(params?: GetVulnerabilitiesParams): GetVulnerabilitiesParams {
  return {
    ...DEFAULT_VULNERABILITY_PARAMS,
    ...params,
  }
}

function resolveEnabledFlag(enabled: boolean | undefined, fallback: boolean): boolean {
  return enabled !== undefined ? enabled : fallback
}

function transformVulnerabilityResponse(
  response: VulnerabilityListResponse,
  defaultParams: GetVulnerabilitiesParams,
  targetId?: number
) {
  const items = response?.results ?? []

  const vulnerabilities: Vulnerability[] = items.flatMap((item) => {
    if (typeof item.id !== "number") {
      return []
    }
    let severity = (item.severity || "info") as
      | VulnerabilitySeverity
      | "unknown"
    if (severity === "unknown") {
      severity = "info"
    }

    let cvssScore: number | undefined
    if (typeof item.cvssScore === "number") {
      cvssScore = item.cvssScore
    } else if (item.cvssScore != null) {
      const num = Number(item.cvssScore)
      cvssScore = Number.isNaN(num) ? undefined : num
    }

    const createdAt = item.createdAt ?? new Date().toISOString()

    const vulnerability: Vulnerability = {
      id: item.id,
      vulnType: item.vulnType || "unknown",
      url: item.url || "",
      description: item.description || "",
      severity: severity as VulnerabilitySeverity,
      source: item.source || "scan",
      cvssScore,
      rawOutput: item.rawOutput || {},
      isReviewed: item.isReviewed ?? false,
      reviewedAt: item.reviewedAt ?? null,
      createdAt,
    }

    if (targetId !== undefined) {
      vulnerability.target = item.target ?? targetId
    }

    return [vulnerability]
  })

  const pagination: PaginationInfo = {
    ...normalizePagination(response, defaultParams.page ?? 1, defaultParams.pageSize ?? 10),
  }

  return { vulnerabilities, pagination }
}

/** Get all vulnerabilities */
export function useAllVulnerabilities(
  params?: GetVulnerabilitiesParams,
  options?: { enabled?: boolean },
  filter?: string,
) {
  const defaultParams = buildVulnerabilityParams(params)

  return useQuery({
    queryKey: vulnerabilityKeys.list(defaultParams, filter),
    queryFn: () => VulnerabilityService.getAllVulnerabilities(defaultParams, filter),
    enabled: resolveEnabledFlag(options?.enabled, true),
    select: (response: VulnerabilityListResponse) =>
      transformVulnerabilityResponse(response, defaultParams),
    placeholderData: keepPreviousData,
  })
}

export function useScanVulnerabilities(
  scanId: number,
  params?: GetVulnerabilitiesParams,
  options?: { enabled?: boolean },
  filter?: string,
) {
  const defaultParams = buildVulnerabilityParams(params)

  return useQuery({
    queryKey: vulnerabilityKeys.byScan(scanId, defaultParams, filter),
    queryFn: () =>
      VulnerabilityService.getVulnerabilitiesByScanId(scanId, defaultParams, filter),
    enabled: resolveEnabledFlag(options?.enabled, !!scanId),
    select: (response: VulnerabilityListResponse) =>
      transformVulnerabilityResponse(response, defaultParams),
    placeholderData: keepPreviousData,
  })
}

export function useTargetVulnerabilities(
  targetId: number,
  params?: GetVulnerabilitiesParams,
  options?: { enabled?: boolean },
  filter?: string,
) {
  const defaultParams = buildVulnerabilityParams(params)

  return useQuery({
    queryKey: vulnerabilityKeys.byTarget(targetId, defaultParams, filter),
    queryFn: () =>
      VulnerabilityService.getVulnerabilitiesByTargetId(targetId, defaultParams, filter),
    enabled: resolveEnabledFlag(options?.enabled, !!targetId),
    select: (response: VulnerabilityListResponse) =>
      transformVulnerabilityResponse(response, defaultParams, targetId),
    placeholderData: keepPreviousData,
  })
}

/** Get global vulnerability stats */
export function useVulnerabilityStats(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: vulnerabilityKeys.stats(),
    queryFn: () => VulnerabilityService.getStats(),
    enabled: options?.enabled ?? true,
  })
}

/** Get vulnerability stats by target ID */
export function useTargetVulnerabilityStats(targetId: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: vulnerabilityKeys.statsByTarget(targetId),
    queryFn: () => VulnerabilityService.getStatsByTargetId(targetId),
    enabled: options?.enabled !== undefined ? options.enabled : !!targetId,
  })
}
