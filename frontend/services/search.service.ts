import { api } from "@/lib/api-client"
import type { SearchParams, SearchResponse, AssetType } from "@/types/search.types"

/**
 * Asset search API service
 * 
 * Search syntax:
 * - field="value" fuzzy matching (ILIKE %value%)
 * - field=="value" exact match
 * - field!="value" not equal
 * - && AND operator
 * - || OR operator
 * 
 * Supported asset types:
 * - website: website (default)
 * - endpoint: endpoint
 * 
 * Example:
 * - host="api" && tech="nginx"
 * - tech="vue" || tech="react"
 * - status=="200" && host!="test"
 */
export class SearchService {
  /**
   * Search assets
   * GET /api/assets/search/
   */
  static async search(params: SearchParams): Promise<SearchResponse> {
    const queryParams = new URLSearchParams()
    
    if (params.q) queryParams.append('q', params.q)
    if (params.asset_type) queryParams.append('asset_type', params.asset_type)
    if (params.page) queryParams.append('page', params.page.toString())
    if (params.pageSize) queryParams.append('pageSize', params.pageSize.toString())
    
    const response = await api.get<SearchResponse>(
      `/assets/search/?${queryParams.toString()}`
    )
    return response.data
  }

  /**
   * Export search results to CSV
   * GET /api/assets/search/export/
   * 
   * Use the browser's native download flow so progress remains visible.
   */
  static async exportCSV(query: string, assetType: AssetType): Promise<void> {
    const queryParams = new URLSearchParams()
    queryParams.append('q', query)
    queryParams.append('asset_type', assetType)
    
    // Open the download link directly and use the browser's native download manager
    // This will show download progress without blocking the page
    const downloadUrl = `/api/assets/search/export/?${queryParams.toString()}`
    window.open(downloadUrl, '_blank')
  }
}
