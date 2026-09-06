/**
 * WebSite related type definitions
 */

export interface WebSite {
  id: number
  /** Canonical backend resource name, for example `targets/1/websites/2`. */
  resourceName?: string
  scan?: number
  target?: number
  url: string
  host: string
  location: string
  title: string
  webserver: string
  contentType: string
  statusCode: number | null
  contentLength: number | null
  responseBody: string
  tech: string[]
  vhost: boolean | null
  subdomain: string
  responseHeaders?: string
  createdAt: string
  screenshot?: WebsiteScreenshotSummary
}

export interface WebsiteScreenshotSummary {
  id: number
  resourceName: string
  url: string
  statusCode: number | null
  createdAt: string
  updatedAt: string
}

/**
 * Read-only Website-detail range. It is a query projection, not a persisted
 * relationship between a Website and an asset record.
 */
export interface WebsiteAssetScope {
  url: string
  host: string
  readOnly: true
}

export interface Technology {
  id: number
  name: string
  version?: string
  category?: string
}

export interface WebSiteFilters {
  url?: string
  title?: string
  statusCode?: number
  webserver?: string
  contentType?: string
}

export interface WebSiteListResponse {
  results: WebSite[]
  total: number
  page: number
  pageSize: number
  totalPages?: number
  totalSize?: number
  nextPageToken?: string
}

export interface WebsiteListQueryParams {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

export type WebsiteFilterOptionField = "statusCode" | "tech" | "webserver" | "contentType" | "vhost"

export interface FilterOption {
  value: string
  label: string
  count?: number
}

export interface FilterOptionsResponse {
  results: FilterOption[]
}
