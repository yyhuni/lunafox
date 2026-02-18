import { api } from "@/lib/api-client"

// Screenshot type
export interface Screenshot {
  id: number
  url: string
  statusCode: number | null
  createdAt: string
  updatedAt: string
}

// Screenshot snapshot type (for scan results)
export interface ScreenshotSnapshot {
  id: number
  url: string
  statusCode: number | null
  createdAt: string
}

// Paginated response
export interface PaginatedResponse<T> {
  results: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// Bulk delete response
export interface BulkDeleteResponse {
  deletedCount: number
}

/**
 * Screenshot related API service
 */
export class ScreenshotService {
  /**
   * Get screenshots by target
   * GET /api/targets/{target_id}/screenshots/
   */
  static async getByTarget(
    targetId: number,
    params?: { page?: number; pageSize?: number; filter?: string }
  ): Promise<PaginatedResponse<Screenshot>> {
    const response = await api.get<PaginatedResponse<Screenshot>>(
      `/targets/${targetId}/screenshots/`,
      { params }
    )
    return response.data
  }

  /**
   * Get screenshot image URL
   * Returns the URL to fetch the image binary
   */
  static getImageUrl(screenshotId: number): string {
    return `/api/assets/screenshots/${screenshotId}/image/`
  }

  /**
   * Get screenshot snapshots by scan
   * GET /api/scans/{scan_id}/screenshots/
   */
  static async getByScan(
    scanId: number,
    params?: { page?: number; pageSize?: number; filter?: string }
  ): Promise<PaginatedResponse<ScreenshotSnapshot>> {
    const response = await api.get<PaginatedResponse<ScreenshotSnapshot>>(
      `/scans/${scanId}/screenshots/`,
      { params }
    )
    return response.data
  }

  /**
   * Get screenshot snapshot image URL
   */
  static getSnapshotImageUrl(scanId: number, snapshotId: number): string {
    return `/api/scans/${scanId}/screenshots/${snapshotId}/image/`
  }

  /**
   * Bulk delete screenshots
   * POST /api/screenshots/bulk-delete/
   */
  static async bulkDelete(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await api.post<BulkDeleteResponse>(
      `/screenshots/bulk-delete/`,
      { ids }
    )
    return response.data
  }
}
