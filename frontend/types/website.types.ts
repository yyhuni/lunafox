/**
 * WebSite related type definitions
 */

export interface WebSite {
  id: number
  scan?: number
  target?: number
  url: string
  host: string
  location: string
  title: string
  webserver: string
  contentType: string
  statusCode: number
  contentLength: number
  responseBody: string
  tech: string[]
  vhost: boolean | null
  subdomain: string
  responseHeaders?: string
  createdAt: string
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
  totalPages: number
}
