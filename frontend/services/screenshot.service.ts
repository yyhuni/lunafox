import { api } from "@/lib/api-client"
import { screenshotName } from "@/lib/resource-name"
import type {
  BulkDeleteScreenshotResponse,
  Screenshot,
  ScreenshotFilterOptionField,
  ScreenshotFilterOptionsResponse,
  ScreenshotListQueryParams,
  ScreenshotListResponse,
  ScreenshotSnapshot,
} from "@/types/screenshot.types"

type AipScreenshotListResponse<T> = {
  results: T[]
  totalSize: number
  nextPageToken?: string
}

function buildScreenshotListParams(params?: ScreenshotListQueryParams) {
  return {
    pageSize: params?.pageSize ?? 12,
    ...(params?.pageToken && { pageToken: params.pageToken }),
    ...(params?.filter && { filter: params.filter }),
    ...(params?.orderBy && { orderBy: params.orderBy }),
  }
}

function normalizeScreenshotListResponse<T>(
  response: AipScreenshotListResponse<T>,
  params?: ScreenshotListQueryParams
): ScreenshotListResponse<T> {
  const pageSize = params?.pageSize ?? 12
  const total = response.totalSize

  return {
    results: response.results,
    total,
    page: 1,
    pageSize,
    totalPages: total === 0 ? 0 : Math.ceil(total / pageSize),
    totalSize: response.totalSize,
    nextPageToken: response.nextPageToken,
  }
}

/**
 * Screenshot related API service
 */
export class ScreenshotService {
  /**
   * Get screenshots by target
   * GET /v1/targets/{target_id}/screenshots/
   */
  static async getByTarget(
    targetId: number,
    params?: ScreenshotListQueryParams
  ): Promise<ScreenshotListResponse<Screenshot>> {
    const response = await api.get<AipScreenshotListResponse<Screenshot>>(
      `/targets/${targetId}/screenshots/`,
      { params: buildScreenshotListParams(params) }
    )
    return normalizeScreenshotListResponse(response.data, params)
  }

  /**
   * Get screenshot image URL
   * GET /v1/screenshots/{screenshot}/blob
   */
  static getImageUrl(screenshotId: number): string {
    return `/v1/screenshots/${screenshotId}/blob`
  }

  /**
   * Get screenshot snapshots by scan
   * GET /v1/scans/{scan_id}/screenshots/
   */
  static async getByScan(
    scanId: number,
    params?: ScreenshotListQueryParams
  ): Promise<ScreenshotListResponse<ScreenshotSnapshot>> {
    const response = await api.get<AipScreenshotListResponse<ScreenshotSnapshot>>(
      `/scans/${scanId}/screenshots/`,
      { params: buildScreenshotListParams(params) }
    )
    return normalizeScreenshotListResponse(response.data, params)
  }

  static async getTargetFilterOptions(
    targetId: number,
    field: ScreenshotFilterOptionField
  ): Promise<ScreenshotFilterOptionsResponse> {
    const response = await api.get<ScreenshotFilterOptionsResponse>(
      `/targets/${targetId}/screenshots/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  static async getScanFilterOptions(
    scanId: number,
    field: ScreenshotFilterOptionField
  ): Promise<ScreenshotFilterOptionsResponse> {
    const response = await api.get<ScreenshotFilterOptionsResponse>(
      `/scans/${scanId}/screenshots/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  /**
   * Get screenshot snapshot image URL
   * GET /v1/scans/{scan}/screenshotSnapshots/{screenshot_snapshot}/blob
   */
  static getSnapshotImageUrl(scanId: number, snapshotId: number): string {
    return `/v1/scans/${scanId}/screenshotSnapshots/${snapshotId}/blob`
  }

  /**
   * Bulk delete screenshots
   * POST /v1/screenshots:batchDelete
   */
  static async bulkDelete(targetId: number, ids: number[]): Promise<BulkDeleteScreenshotResponse> {
    const response = await api.post<BulkDeleteScreenshotResponse>(
      `/screenshots:batchDelete`,
      { names: ids.map((id) => screenshotName(targetId, id)) }
    )
    return response.data
  }
}
