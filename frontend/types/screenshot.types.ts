/**
 * Screenshot related type definitions.
 */

export interface Screenshot {
  id: number
  url: string
  statusCode: number | null
  createdAt: string
  updatedAt?: string
}

export interface ScreenshotSnapshot {
  id: number
  url: string
  statusCode: number | null
  createdAt: string
}

export interface ScreenshotListResponse<T = Screenshot> {
  results: T[]
  total: number
  page: number
  pageSize: number
  totalPages?: number
  totalSize?: number
  nextPageToken?: string
}

export interface ScreenshotListQueryParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type ScreenshotFilterOptionField = "statusCode"

export interface ScreenshotFilterOption {
  value: string
  label: string
  count?: number
}

export interface ScreenshotFilterOptionsResponse {
  results: ScreenshotFilterOption[]
}

export interface BulkDeleteScreenshotResponse {
  deletedCount: number
}
