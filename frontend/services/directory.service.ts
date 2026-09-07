import { api } from "@/lib/api-client"
import type { Directory, DirectoryFilterOptionField, DirectoryFilterOptionsResponse, DirectoryListQueryParams, DirectoryListResponse } from '@/types/directory.types'

// Bulk create directories response type
export interface BulkCreateDirectoriesResponse {
  message: string
  createdCount: number
}

// Delete single directory response type
export interface DeleteDirectoryResponse {
  message: string
  directoryId: number
  directoryUrl: string
  deletedCount: number
  deletedDirectories: string[]
  detail: {
    phase1: string
    phase2: string
  }
}

// Bulk delete directories response type
export interface BulkDeleteDirectoriesResponse {
  message: string
  deletedCount: number
  requestedIds: number[]
  cascadeDeleted: Record<string, number>
}

type AipDirectoryListResponse = {
  results: unknown[]
  totalSize?: number
  nextPageToken?: string
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
}

const DIRECTORY_INT64_MAX = "9223372036854775807"
const CANONICAL_NON_NEGATIVE_DECIMAL = /^(?:0|[1-9][0-9]*)$/

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function parseNullableDirectoryInt64(value: unknown, field: "contentLength" | "duration"): string | null {
  if (value === null) return null

  // Coercing through number would round valid observations above 2^53 - 1.
  if (
    typeof value !== "string" ||
    !CANONICAL_NON_NEGATIVE_DECIMAL.test(value) ||
    value.length > DIRECTORY_INT64_MAX.length ||
    (value.length === DIRECTORY_INT64_MAX.length && value > DIRECTORY_INT64_MAX)
  ) {
    throw new TypeError(`Invalid Directory ${field}: expected a canonical int64 decimal string or null`)
  }
  return value
}

function parseDirectoryReadModel(value: unknown): Directory {
  if (!isRecord(value)) {
    throw new TypeError("Invalid Directory response item: expected an object")
  }
  if (!Number.isInteger(value.id)) {
    throw new TypeError("Invalid Directory id")
  }
  if (typeof value.url !== "string") {
    throw new TypeError("Invalid Directory url")
  }
  if (value.status !== null && !Number.isInteger(value.status)) {
    throw new TypeError("Invalid Directory status")
  }
  if (typeof value.contentType !== "string") {
    throw new TypeError("Invalid Directory contentType")
  }
  if (typeof value.createdAt !== "string") {
    throw new TypeError("Invalid Directory createdAt")
  }

  return {
    id: value.id as number,
    url: value.url,
    status: value.status as number | null,
    contentLength: parseNullableDirectoryInt64(value.contentLength, "contentLength"),
    contentType: value.contentType,
    duration: parseNullableDirectoryInt64(value.duration, "duration"),
    createdAt: value.createdAt,
  }
}

function buildDirectoryListParams(params?: DirectoryListQueryParams) {
  return {
    pageSize: params?.pageSize ?? 10,
    ...(params?.pageToken && { pageToken: params.pageToken }),
    ...(params?.filter && { filter: params.filter }),
    ...(params?.orderBy && { orderBy: params.orderBy }),
  }
}

function normalizeDirectoryListResponse(
  response: AipDirectoryListResponse,
  params?: DirectoryListQueryParams
): DirectoryListResponse {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? params?.pageSize ?? 10
  const total = response.total ?? response.totalSize ?? response.results.length

  return {
    results: response.results.map(parseDirectoryReadModel),
    total,
    page,
    pageSize,
    totalPages: response.totalPages ?? (total === 0 ? 0 : Math.ceil(total / pageSize)),
    totalSize: response.totalSize,
    nextPageToken: response.nextPageToken,
  }
}

/** Directory related API service */
export class DirectoryService {
  /**
   * Get target directories list
   * GET /v1/targets/{target_id}/directories/
   */
  static async getTargetDirectories(
    targetId: number,
    params?: DirectoryListQueryParams
  ): Promise<DirectoryListResponse> {
    const response = await api.get<AipDirectoryListResponse>(
      `/targets/${targetId}/directories/`,
      { params: buildDirectoryListParams(params) }
    )
    return normalizeDirectoryListResponse(response.data, params)
  }

  /**
   * Get scan directories list
   * GET /v1/scans/{scan_id}/directories/
   */
  static async getScanDirectories(
    scanId: number,
    params?: DirectoryListQueryParams
  ): Promise<DirectoryListResponse> {
    const response = await api.get<AipDirectoryListResponse>(
      `/scans/${scanId}/directories/`,
      { params: buildDirectoryListParams(params) }
    )
    return normalizeDirectoryListResponse(response.data, params)
  }

  static async getTargetDirectoryFilterOptions(
    targetId: number,
    field: DirectoryFilterOptionField
  ): Promise<DirectoryFilterOptionsResponse> {
    const response = await api.get<DirectoryFilterOptionsResponse>(
      `/targets/${targetId}/directories/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  static async getScanDirectoryFilterOptions(
    scanId: number,
    field: DirectoryFilterOptionField
  ): Promise<DirectoryFilterOptionsResponse> {
    const response = await api.get<DirectoryFilterOptionsResponse>(
      `/scans/${scanId}/directories/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  /**
   * Delete single directory
   * DELETE /v1/directories/{directory_id}/
   */
  static async deleteDirectory(directoryId: number): Promise<DeleteDirectoryResponse> {
    const response = await api.delete<DeleteDirectoryResponse>(`/directories/${directoryId}/`)
    return response.data
  }

  /**
   * Bulk delete directories
   * POST /v1/directories:batchDelete
   */
  static async bulkDeleteDirectories(targetId: number, ids: number[]): Promise<BulkDeleteDirectoriesResponse> {
    const response = await api.post<BulkDeleteDirectoriesResponse>(
      `/directories:batchDelete`,
      { names: ids.map((id) => `targets/${targetId}/directories/${id}`) }
    )
    return response.data
  }

  /**
   * Bulk create directories (bind to target)
   * POST /v1/targets/{target}/directories:batchCreate
   * Backend contract: max 5,000 directories per request.
   */
  static async bulkCreateDirectories(
    targetId: number,
    urls: string[]
  ): Promise<BulkCreateDirectoriesResponse> {
    const response = await api.post<BulkCreateDirectoriesResponse>(
      `/targets/${targetId}/directories:batchCreate`,
      { urls }
    )
    return response.data
  }

  /**
   * Export all directory URLs by target (text file, one per line)
   * GET /v1/targets/{target}/directories/exportFiles/current
   */
  static async exportDirectoriesByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/directories/exportFiles/current`, {
      responseType: "blob",
    })
    return response.data
  }

  /**
   * Export all directory URLs by scan task (text file, one per line)
   * GET /v1/scans/{scan}/directories/exportFiles/current
   */
  static async exportDirectoriesByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/directories/exportFiles/current`, {
      responseType: "blob",
    })
    return response.data
  }
}
