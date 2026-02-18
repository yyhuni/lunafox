import { api } from "@/lib/api-client"
import { USE_MOCK, mockDelay, mockDirectories, getMockTargetById, getMockScanById } from '@/mock'
import type { DirectoryListResponse } from '@/types/directory.types'

// Bulk create directories response type
export interface BulkCreateDirectoriesResponse {
  message: string
  createdCount: number
}

// Bulk delete response type
export interface BulkDeleteResponse {
  deletedCount: number
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

// Mock helper function
const buildMockDirectoriesResponse = (params: {
  page: number
  pageSize: number
  filter?: string
  targetId?: number
  scanId?: number
}): DirectoryListResponse => {
  const page = params.page
  const pageSize = params.pageSize
  const filter = params.filter?.toLowerCase() || ""
  const target = params.targetId ? getMockTargetById(params.targetId) : undefined
  const scan = params.scanId ? getMockScanById(params.scanId) : undefined
  const domain = (target?.name || scan?.target?.name || "").toLowerCase()

  let filtered = mockDirectories

  if (domain) {
    filtered = filtered.filter((d) =>
      d.url.toLowerCase().includes(domain) ||
      d.websiteUrl.toLowerCase().includes(domain)
    )
  }

  if (filter) {
    filtered = filtered.filter((d) =>
      d.url.toLowerCase().includes(filter) ||
      d.contentType.toLowerCase().includes(filter)
    )
  }

  const total = filtered.length
  const totalPages = Math.ceil(total / pageSize)
  const start = (page - 1) * pageSize
  const results = filtered.slice(start, start + pageSize)

  return {
    results,
    total,
    page,
    pageSize,
    totalPages,
  }
}

/** Directory related API service */
export class DirectoryService {
  /**
   * Get target directories list
   * GET /api/targets/{target_id}/directories/
   */
  static async getTargetDirectories(
    targetId: number,
    params: { page: number; pageSize: number; filter?: string }
  ): Promise<DirectoryListResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return buildMockDirectoriesResponse({
        page: params.page,
        pageSize: params.pageSize,
        filter: params.filter,
        targetId,
      })
    }
    const response = await api.get<DirectoryListResponse>(
      `/targets/${targetId}/directories/`,
      { params }
    )
    return response.data
  }

  /**
   * Get scan directories list
   * GET /api/scans/{scan_id}/directories/
   */
  static async getScanDirectories(
    scanId: number,
    params: { page: number; pageSize: number; filter?: string }
  ): Promise<DirectoryListResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return buildMockDirectoriesResponse({
        page: params.page,
        pageSize: params.pageSize,
        filter: params.filter,
        scanId,
      })
    }
    const response = await api.get<DirectoryListResponse>(
      `/scans/${scanId}/directories/`,
      { params }
    )
    return response.data
  }

  /**
   * Delete single directory
   * DELETE /api/directories/{directory_id}/
   */
  static async deleteDirectory(directoryId: number): Promise<DeleteDirectoryResponse> {
    const response = await api.delete<DeleteDirectoryResponse>(`/directories/${directoryId}/`)
    return response.data
  }

  /**
   * Bulk delete directories
   * POST /api/directories/bulk-delete/
   */
  static async bulkDeleteDirectories(ids: number[]): Promise<BulkDeleteDirectoriesResponse> {
    const response = await api.post<BulkDeleteDirectoriesResponse>(
      `/directories/bulk-delete/`,
      { ids }
    )
    return response.data
  }

  /**
   * Bulk delete directories (legacy method)
   * POST /api/directories/bulk-delete/
   */
  static async bulkDelete(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await api.post<BulkDeleteResponse>(
      `/directories/bulk-delete/`,
      { ids }
    )
    return response.data
  }

  /**
   * Bulk create directories (bind to target)
   * POST /api/targets/{target_id}/directories/bulk-create/
   */
  static async bulkCreateDirectories(
    targetId: number,
    urls: string[]
  ): Promise<BulkCreateDirectoriesResponse> {
    const response = await api.post<BulkCreateDirectoriesResponse>(
      `/targets/${targetId}/directories/bulk-create/`,
      { urls }
    )
    return response.data
  }

  /**
   * Export all directory URLs by target (text file, one per line)
   * GET /api/targets/{target_id}/directories/export/
   */
  static async exportDirectoriesByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/directories/export/`, {
      responseType: "blob",
    })
    return response.data
  }

  /**
   * Export all directory URLs by scan task (text file, one per line)
   * GET /api/scans/{scan_id}/directories/export/
   */
  static async exportDirectoriesByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/directories/export/`, {
      responseType: "blob",
    })
    return response.data
  }
}
