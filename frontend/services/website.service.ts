import { api } from "@/lib/api-client"
import { USE_MOCK, mockDelay, getMockWebsites, getMockScanById } from '@/mock'
import type { WebSiteListResponse } from '@/types/website.types'

// Bulk create websites response type
export interface BulkCreateWebsitesResponse {
  message: string
  createdCount: number
}

// Bulk delete response type
export interface BulkDeleteResponse {
  deletedCount: number
}

// Delete single website response type
export interface DeleteWebsiteResponse {
  message: string
  websiteId: number
  websiteUrl: string
  deletedCount: number
  deletedWebSites: string[]
  detail: {
    phase1: string
    phase2: string
  }
}

// Bulk delete websites response type
export interface BulkDeleteWebsitesResponse {
  message: string
  deletedCount: number
  requestedIds: number[]
  cascadeDeleted: Record<string, number>
}

/**
 * Website related API service
 * All frontend website interface calls should be centralized here
 */
export class WebsiteService {
  /**
   * Get target websites list
   * GET /api/targets/{target_id}/websites/
   */
  static async getTargetWebSites(
    targetId: number,
    params: { page: number; pageSize: number; filter?: string }
  ): Promise<WebSiteListResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockWebsites({
        page: params.page,
        pageSize: params.pageSize,
        search: params.filter,
        targetId,
      })
    }
    const response = await api.get<WebSiteListResponse>(
      `/targets/${targetId}/websites/`,
      { params }
    )
    return response.data
  }

  /**
   * Get scan websites list
   * GET /api/scans/{scan_id}/websites/
   */
  static async getScanWebSites(
    scanId: number,
    params: { page: number; pageSize: number; filter?: string }
  ): Promise<WebSiteListResponse> {
    if (USE_MOCK) {
      await mockDelay()
      const scan = getMockScanById(scanId)
      if (!scan?.targetId) {
        return {
          results: [],
          total: 0,
          page: params.page,
          pageSize: params.pageSize,
          totalPages: 0,
        }
      }
      return getMockWebsites({
        page: params.page,
        pageSize: params.pageSize,
        search: params.filter,
        targetId: scan.targetId,
      })
    }
    const response = await api.get<WebSiteListResponse>(
      `/scans/${scanId}/websites/`,
      { params }
    )
    return response.data
  }

  /**
   * Delete single website
   * DELETE /api/websites/{website_id}/
   */
  static async deleteWebSite(websiteId: number): Promise<DeleteWebsiteResponse> {
    const response = await api.delete<DeleteWebsiteResponse>(`/websites/${websiteId}/`)
    return response.data
  }

  /**
   * Bulk delete websites
   * POST /api/websites/bulk-delete/
   */
  static async bulkDeleteWebSites(ids: number[]): Promise<BulkDeleteWebsitesResponse> {
    const response = await api.post<BulkDeleteWebsitesResponse>(
      `/websites/bulk-delete/`,
      { ids }
    )
    return response.data
  }

  /**
   * Bulk delete websites (legacy method)
   * POST /api/websites/bulk-delete/
   */
  static async bulkDelete(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await api.post<BulkDeleteResponse>(
      `/websites/bulk-delete/`,
      { ids }
    )
    return response.data
  }

  /**
   * Bulk create websites (bind to target)
   * POST /api/targets/{target_id}/websites/bulk-create/
   */
  static async bulkCreateWebsites(
    targetId: number,
    urls: string[]
  ): Promise<BulkCreateWebsitesResponse> {
    const response = await api.post<BulkCreateWebsitesResponse>(
      `/targets/${targetId}/websites/bulk-create/`,
      { urls }
    )
    return response.data
  }

  /**
   * Export all website URLs by target (text file, one per line)
   * GET /api/targets/{target_id}/websites/export/
   */
  static async exportWebsitesByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/websites/export/`, {
      responseType: "blob",
    })
    // Check if response is actually an error (JSON instead of CSV)
    if (response.data.type === 'application/json') {
      const text = await response.data.text()
      const error = JSON.parse(text)
      throw new Error(error.error?.message || 'Export failed')
    }
    return response.data
  }

  /**
   * Export all website URLs by scan task (text file, one per line)
   * GET /api/scans/{scan_id}/websites/export/
   */
  static async exportWebsitesByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/websites/export/`, {
      responseType: "blob",
    })
    // Check if response is actually an error (JSON instead of CSV)
    if (response.data.type === 'application/json') {
      const text = await response.data.text()
      const error = JSON.parse(text)
      throw new Error(error.error?.message || 'Export failed')
    }
    return response.data
  }
}
