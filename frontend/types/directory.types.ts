/**
 * Directory related type definitions
 */

export interface Directory {
  id: number
  url: string
  status: number | null
  contentLength: string | null
  contentType: string
  duration: string | null
  createdAt: string
}

export interface DirectoryFilters {
  url?: string
  status?: number
  contentType?: string
}

export interface DirectoryListResponse {
  results: Directory[]
  total: number
  page: number
  pageSize: number
  totalPages?: number
  totalSize?: number
  nextPageToken?: string
}

export interface DirectoryListQueryParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type DirectoryFilterOptionField = "status" | "contentType"

export interface DirectoryFilterOption {
  value: string
  label: string
  count?: number
}

export interface DirectoryFilterOptionsResponse {
  results: DirectoryFilterOption[]
}
