"use client"

import { useQuery } from "@tanstack/react-query"
import { SearchService } from "@/services/search.service"
import type { SearchParams, SearchResponse } from "@/types/search.types"

export function useAssetSearch(params: SearchParams | undefined, options?: { enabled?: boolean; requestNonce?: number }) {
  const isEnabled = (options?.enabled ?? true) && Boolean(params?.q.trim())

  return useQuery<SearchResponse>({
    queryKey: ["asset-search", params ?? null, options?.requestNonce ?? 0],
    queryFn: () => {
      if (!params) {
        throw new Error("A global asset search query is required")
      }
      return SearchService.search(params)
    },
    enabled: isEnabled,
    staleTime: 30_000,
  })
}
