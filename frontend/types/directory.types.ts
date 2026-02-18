/**
 * Directory related type definitions
 */

export interface Directory {
  id: number
  url: string
  status: number | null
  contentLength: number | null  // Backend returns contentLength
  words: number | null
  lines: number | null
  contentType: string
  duration: number | null
  websiteUrl: string  // Backend returns websiteUrl
  createdAt: string  // Backend returns createdAt
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
  totalPages: number
}
