// Asset type
export type AssetType = "website" | "endpoint"

export interface SearchResult {
  id: number
  name: string
  url: string
  host: string
  title: string
  tech: string[]
  statusCode: number | null
  contentLength: number | null
  contentType: string
  webserver: string
  location: string
  vhost: boolean | null
  responseHeaders: string
  responseBody: string
  createdAt: string
  targetId?: number
  responseBodyTruncated?: boolean
  responseHeadersTruncated?: boolean
}

export type WebsiteSearchResult = SearchResult
export type EndpointSearchResult = SearchResult

export type SearchState = "initial" | "searching" | "results"

export interface SearchResponse {
  results: SearchResult[]
  nextPageToken?: string
}

export interface SearchParams {
  q: string
  assetType: AssetType
  pageSize?: number
  pageToken?: string
}
