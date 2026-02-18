import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { SearchService } from '@/services/search.service'
import type { SearchParams, SearchResponse } from '@/types/search.types'

/**
 * Asset Search Hook
 * 
 * @param params search parameters
 * @param options query options
 * @returns search results
 */
export function useAssetSearch(
  params: SearchParams,
  options?: { enabled?: boolean }
) {
  // Check if there is a valid search query
  const hasSearchParams = !!(params.q && params.q.trim())

  return useQuery<SearchResponse>({
    queryKey: ['asset-search', params],
    queryFn: () => SearchService.search(params),
    enabled: (options?.enabled ?? true) && hasSearchParams,
    placeholderData: keepPreviousData,
    staleTime: 30000, // No re-request within 30 seconds
  })
}
