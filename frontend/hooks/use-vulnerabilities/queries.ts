import { useCallback } from "react"
import { useQuery, useQueryClient, keepPreviousData } from "@tanstack/react-query"
import { normalizePagination } from "@/hooks/_shared/pagination"
import { VulnerabilityService } from "@/services/vulnerability.service"
import type {
  Vulnerability,
  VulnerabilitySeverity,
  GetVulnerabilitiesParams,
  VulnerabilityFilterOptionField,
} from "@/types/vulnerability.types"
import type { PaginationInfo } from "@/types/common.types"
import type { WebsiteAssetScope } from "@/types/website.types"
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
  totalSize?: number
  page?: number
  pageSize?: number
  totalPages?: number
  nextPageToken?: string
}

function buildVulnerabilityParams(params: GetVulnerabilitiesParams = {}): GetVulnerabilitiesParams {
  return {
    pageSize: params.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
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
    ...normalizePagination(
      { ...response, total: response.total ?? response.totalSize },
      1,
      defaultParams.pageSize ?? 10,
    ),
  }

  return {
    vulnerabilities,
    pagination,
    results: vulnerabilities,
    total: response.total ?? response.totalSize ?? vulnerabilities.length,
    page: response.page ?? 1,
    pageSize: response.pageSize ?? defaultParams.pageSize ?? 10,
    totalPages: response.totalPages,
    totalSize: response.totalSize,
    nextPageToken: response.nextPageToken,
  }
}

/** Get all vulnerabilities */
export function useAllVulnerabilities(
  params?: GetVulnerabilitiesParams,
  options?: { enabled?: boolean },
) {
  const defaultParams = buildVulnerabilityParams(params)

  return useQuery({
    queryKey: vulnerabilityKeys.list(defaultParams),
    queryFn: () => VulnerabilityService.getAllVulnerabilities(defaultParams),
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
) {
  const defaultParams = buildVulnerabilityParams(params)

  return useQuery({
    queryKey: vulnerabilityKeys.byScan(scanId, defaultParams),
    queryFn: () =>
      VulnerabilityService.getVulnerabilitiesByScanId(scanId, defaultParams),
    enabled: resolveEnabledFlag(options?.enabled, !!scanId),
    select: (response: VulnerabilityListResponse) =>
      transformVulnerabilityResponse(response, defaultParams),
    placeholderData: keepPreviousData,
  })
}

export function useTargetVulnerabilities(
  targetId: number,
  params?: GetVulnerabilitiesParams,
  options?: { enabled?: boolean; websiteScope?: WebsiteAssetScope },
) {
  const defaultParams = buildVulnerabilityParams(params)

  return useQuery({
    queryKey: vulnerabilityKeys.byTarget(targetId, defaultParams, options?.websiteScope),
    queryFn: () =>
      VulnerabilityService.getVulnerabilitiesByTargetId(targetId, defaultParams),
    enabled: resolveEnabledFlag(options?.enabled, !!targetId),
    select: (response: VulnerabilityListResponse) =>
      transformVulnerabilityResponse(response, defaultParams, targetId),
    placeholderData: keepPreviousData,
  })
}

export function useGlobalVulnerabilityFilterOptions(
  field: VulnerabilityFilterOptionField,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: vulnerabilityKeys.filterOptions("global", undefined, field),
    queryFn: () => VulnerabilityService.getGlobalFilterOptions(field),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

export function useTargetVulnerabilityFilterOptions(
  targetId: number,
  field: VulnerabilityFilterOptionField,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: vulnerabilityKeys.filterOptions("target", targetId, field),
    queryFn: () => VulnerabilityService.getTargetFilterOptions(targetId, field),
    enabled: options?.enabled ?? !!targetId,
    placeholderData: keepPreviousData,
  })
}

export function useScanVulnerabilityFilterOptions(
  scanId: number,
  field: VulnerabilityFilterOptionField,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: vulnerabilityKeys.filterOptions("scan", scanId, field),
    queryFn: () => VulnerabilityService.getScanFilterOptions(scanId, field),
    enabled: options?.enabled ?? !!scanId,
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

export function useRecentVulnerabilities(limit = 5, options?: { enabled?: boolean }) {
  return useAllVulnerabilities(
    {
      pageSize: limit,
      orderBy: "createdAt desc",
    },
    options,
  )
}

export function useLoadVulnerabilityDetail() {
  const queryClient = useQueryClient()

  return useCallback((id: number) => {
    return queryClient.fetchQuery({
      queryKey: vulnerabilityKeys.detail(id),
      queryFn: () => VulnerabilityService.getVulnerabilityById(id),
    })
  }, [queryClient])
}
