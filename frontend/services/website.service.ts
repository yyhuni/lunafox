import { api } from "@/lib/api-client"
import type { FilterOptionsResponse, WebSite, WebsiteFilterOptionField, WebsiteListQueryParams, WebSiteListResponse, WebsiteScreenshotSummary } from '@/types/website.types'

// Bulk create websites response type
export interface BulkCreateWebsitesResponse {
  message: string
  createdCount: number
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

type AipWebsiteScreenshotSummary = Omit<WebsiteScreenshotSummary, "resourceName"> & {
  name: string
}

type AipWebsite = Omit<WebSite, "resourceName" | "screenshot"> & {
  name: string
  screenshot?: AipWebsiteScreenshotSummary
}

type AipWebsiteListResponse = {
  results: AipWebsite[]
  totalSize?: number
  nextPageToken?: string
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
}

function normalizeWebsite(website: AipWebsite): WebSite {
  const { name, screenshot, ...fields } = website
  return {
    ...fields,
    resourceName: name,
    ...(screenshot ? {
      screenshot: {
        id: screenshot.id,
        resourceName: screenshot.name,
        url: screenshot.url,
        statusCode: screenshot.statusCode,
        createdAt: screenshot.createdAt,
        updatedAt: screenshot.updatedAt,
      },
    } : {}),
  }
}

function buildWebsiteListParams(params?: WebsiteListQueryParams) {
  return {
    pageSize: params?.pageSize ?? 10,
    ...(params?.pageToken && { pageToken: params.pageToken }),
    ...(params?.filter && { filter: params.filter }),
    ...(params?.orderBy && { orderBy: params.orderBy }),
  }
}

function normalizeWebsiteListResponse(
  response: AipWebsiteListResponse,
  params?: WebsiteListQueryParams
): WebSiteListResponse {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? params?.pageSize ?? 10
  const total = response.total ?? response.totalSize ?? response.results.length

  return {
    results: response.results.map(normalizeWebsite),
    total,
    page,
    pageSize,
    totalPages: response.totalPages ?? (total === 0 ? 0 : Math.ceil(total / pageSize)),
    totalSize: response.totalSize,
    nextPageToken: response.nextPageToken,
  }
}

/**
 * Website related API service
 * All frontend website interface calls should be centralized here
 */
export class WebsiteService {
  /** Get one Website resource without loading an unbounded parent collection. */
  static async getWebsite(websiteId: number): Promise<WebSite> {
    const response = await api.get<AipWebsite>(`/websites/${websiteId}/`)
    return normalizeWebsite(response.data)
  }

  /**
   * Get target websites list
   * GET /v1/targets/{target_id}/websites/
   */
  static async getTargetWebSites(
    targetId: number,
    params?: WebsiteListQueryParams
  ): Promise<WebSiteListResponse> {
    const response = await api.get<AipWebsiteListResponse>(
      `/targets/${targetId}/websites/`,
      { params: buildWebsiteListParams(params) }
    )
    return normalizeWebsiteListResponse(response.data, params)
  }

  /**
   * Get scan websites list
   * GET /v1/scans/{scan_id}/websites/
   */
  static async getScanWebSites(
    scanId: number,
    params?: WebsiteListQueryParams
  ): Promise<WebSiteListResponse> {
    const response = await api.get<AipWebsiteListResponse>(
      `/scans/${scanId}/websites/`,
      { params: buildWebsiteListParams(params) }
    )
    return normalizeWebsiteListResponse(response.data, params)
  }

  static async getTargetWebsiteFilterOptions(
    targetId: number,
    field: WebsiteFilterOptionField
  ): Promise<FilterOptionsResponse> {
    const response = await api.get<FilterOptionsResponse>(
      `/targets/${targetId}/websites/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  static async getScanWebsiteFilterOptions(
    scanId: number,
    field: WebsiteFilterOptionField
  ): Promise<FilterOptionsResponse> {
    const response = await api.get<FilterOptionsResponse>(
      `/scans/${scanId}/websites/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  /**
   * Delete single website
   * DELETE /v1/websites/{website_id}/
   */
  static async deleteWebSite(websiteId: number): Promise<DeleteWebsiteResponse> {
    const response = await api.delete<DeleteWebsiteResponse>(`/websites/${websiteId}/`)
    return response.data
  }

  /**
   * Bulk delete websites
   * POST /v1/websites:batchDelete
   */
  static async bulkDeleteWebSites(targetId: number, ids: number[]): Promise<BulkDeleteWebsitesResponse> {
    const response = await api.post<BulkDeleteWebsitesResponse>(
      `/websites:batchDelete`,
      { names: ids.map((id) => `targets/${targetId}/websites/${id}`) }
    )
    return response.data
  }

  /**
   * Bulk create websites (bind to target)
   * POST /v1/targets/{target}/websites:batchCreate
   * Backend contract: max 5,000 websites per request.
   */
  static async bulkCreateWebsites(
    targetId: number,
    urls: string[]
  ): Promise<BulkCreateWebsitesResponse> {
    const response = await api.post<BulkCreateWebsitesResponse>(
      `/targets/${targetId}/websites:batchCreate`,
      { urls }
    )
    return response.data
  }

  /**
   * Export all website URLs by target (text file, one per line)
   * GET /v1/targets/{target}/websites/exportFiles/current
   */
  static async exportWebsitesByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/websites/exportFiles/current`, {
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
   * GET /v1/scans/{scan}/websites/exportFiles/current
   */
  static async exportWebsitesByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/websites/exportFiles/current`, {
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
