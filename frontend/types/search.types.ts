// Asset type
export type AssetType = 'website' | 'endpoint'

// Website search result type
export interface WebsiteSearchResult {
  id: number
  url: string
  host: string
  title: string
  technologies: string[]
  statusCode: number | null
  contentLength: number | null
  contentType: string
  webserver: string
  location: string
  vhost: boolean | null
  responseHeaders: Record<string, string>
  responseBody: string
  createdAt: string | null
  targetId: number
  vulnerabilities: Vulnerability[]
}

// Endpoint search result type
export interface EndpointSearchResult {
	id: number
	url: string
	host: string
  title: string
  technologies: string[]
  statusCode: number | null
  contentLength: number | null
  contentType: string
  webserver: string
  location: string
  vhost: boolean | null
  responseHeaders: Record<string, string>
	responseBody: string
	createdAt: string | null
	targetId: number
}

// Universal search result type (compatible with legacy code)
export type SearchResult = WebsiteSearchResult | EndpointSearchResult

export interface Vulnerability {
  id?: number
  name: string
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info' | 'unknown'
  vulnType: string
  url?: string
}

// search status
export type SearchState = 'initial' | 'searching' | 'results'

// Search response type
export interface SearchResponse {
  results: SearchResult[]
  total: number
  page: number
  pageSize: number
  totalPages: number
  assetType: AssetType
}

// Search operator type
export type SearchOperator = '=' | '==' | '!='

// single search criteria
export interface SearchCondition {
  field: string
  operator: SearchOperator
  value: string
}

// Search expression (supports AND/OR combinations)
export interface SearchExpression {
  conditions: SearchCondition[]  // Conditions within the same group are connected with AND
  orGroups?: SearchExpression[]  // Use OR to connect multiple groups
}

// Search parameters sent to the backend
export interface SearchParams {
  q?: string  // Complete search expression string
  asset_type?: AssetType  // Asset type
  page?: number
  pageSize?: number
}
