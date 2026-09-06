import { api } from "@/lib/api-client"
import type { SearchParams, SearchResponse } from "@/types/search.types"

export class SearchService {
  static async search(params: SearchParams): Promise<SearchResponse> {
    const response = await api.get<SearchResponse>("/assets:search", {
      params: {
        q: params.q,
        assetType: params.assetType,
        ...(params.pageSize !== undefined ? { pageSize: params.pageSize } : {}),
        ...(params.pageToken ? { pageToken: params.pageToken } : {}),
      },
    })
    return response.data
  }
}
