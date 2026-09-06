import { api } from "@/lib/api-client"
import type { 
  Endpoint, 
  CreateEndpointRequest, 
  EndpointFilterOptionField,
  EndpointFilterOptionsResponse,
  EndpointListQueryParams,
  GetEndpointsRequest,
  GetEndpointsResponse,
  CreateEndpointsResponse,
  BatchDeleteEndpointsRequest,
  BatchDeleteEndpointsResponse
} from "@/types/endpoint.types"

// Bulk create endpoints response type
export interface BulkCreateEndpointsResponse {
  message: string
  createdCount: number
}

type AipEndpointListResponse = {
  results: Endpoint[]
  totalSize?: number
  nextPageToken?: string
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
}

function buildEndpointListParams(params?: EndpointListQueryParams) {
  return {
    pageSize: params?.pageSize ?? 10,
    ...(params?.pageToken && { pageToken: params.pageToken }),
    ...(params?.filter && { filter: params.filter }),
    ...(params?.orderBy && { orderBy: params.orderBy }),
  }
}

function normalizeEndpointListResponse(
  response: AipEndpointListResponse,
  params?: EndpointListQueryParams
): GetEndpointsResponse {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? params?.pageSize ?? 10
  const total = response.total ?? response.totalSize ?? response.results.length

  return {
    results: response.results,
    total,
    page,
    pageSize,
    totalPages: response.totalPages ?? (total === 0 ? 0 : Math.ceil(total / pageSize)),
    totalSize: response.totalSize,
    nextPageToken: response.nextPageToken,
  }
}

export class EndpointService {
  /**
   * Bulk create endpoints (bind to target)
   * POST /v1/targets/{target}/endpoints:batchCreate
   * Backend contract: max 5,000 endpoints per request.
   */
  static async bulkCreateEndpoints(
    targetId: number,
    urls: string[]
  ): Promise<BulkCreateEndpointsResponse> {
    const response = await api.post<BulkCreateEndpointsResponse>(
      `/targets/${targetId}/endpoints:batchCreate`,
      { urls }
    )
    return response.data
  }

  /**
   * Get single Endpoint details
   * @param id - Endpoint ID
   * @returns Promise<Endpoint>
   */
  static async getEndpointById(id: number): Promise<Endpoint> {
    const response = await api.get<Endpoint>(`/endpoints/${id}/`)
    return response.data
  }

  /**
   * Get Endpoint list
   * @param params - Query parameters
   * @returns Promise<GetEndpointsResponse>
   */
  static async getEndpoints(params: GetEndpointsRequest): Promise<GetEndpointsResponse> {
    const response = await api.get<GetEndpointsResponse>('/endpoints/', {
      params
    })
    return response.data
  }

  /**
   * Get Endpoint list by target ID (dedicated route)
   * @param targetId - Target ID
   * @param params - Other query parameters
   * @param filter - Smart filter query string
   * @returns Promise<GetEndpointsResponse>
   */
  static async getEndpointsByTargetId(
    targetId: number,
    params?: EndpointListQueryParams
  ): Promise<GetEndpointsResponse> {
    const response = await api.get<AipEndpointListResponse>(`/targets/${targetId}/endpoints/`, {
      params: buildEndpointListParams(params)
    })
    return normalizeEndpointListResponse(response.data, params)
  }

  /**
   * Get Endpoint list by scan ID (historical snapshot)
   * @param scanId - Scan task ID
   * @param params - Pagination and other query parameters
   * @param filter - Smart filter query string
   */
  static async getEndpointsByScanId(
    scanId: number,
    params?: EndpointListQueryParams,
  ): Promise<GetEndpointsResponse> {
    const response = await api.get<AipEndpointListResponse>(`/scans/${scanId}/endpoints/`, {
      params: buildEndpointListParams(params),
    })
    return normalizeEndpointListResponse(response.data, params)
  }

  static async getTargetEndpointFilterOptions(
    targetId: number,
    field: EndpointFilterOptionField
  ): Promise<EndpointFilterOptionsResponse> {
    const response = await api.get<EndpointFilterOptionsResponse>(
      `/targets/${targetId}/endpoints/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  static async getScanEndpointFilterOptions(
    scanId: number,
    field: EndpointFilterOptionField
  ): Promise<EndpointFilterOptionsResponse> {
    const response = await api.get<EndpointFilterOptionsResponse>(
      `/scans/${scanId}/endpoints/filterOptions`,
      { params: { field } }
    )
    return response.data
  }

  /**
   * Batch create Endpoints
   * @param data - Create request object
   * @param data.endpoints - Endpoint data array
   * @returns Promise<CreateEndpointsResponse>
   */
  static async createEndpoints(data: { endpoints: Array<CreateEndpointRequest> }): Promise<CreateEndpointsResponse> {
    const response = await api.post<CreateEndpointsResponse>('/endpoints/create/', data)
    return response.data
  }

  /**
   * Delete Endpoint
   * @param id - Endpoint ID
   * @returns Promise<void>
   */
  static async deleteEndpoint(id: number): Promise<void> {
    await api.delete(`/endpoints/${id}/`)
  }

  /**
   * Batch delete Endpoints
   * @param data - Batch delete request object
   * @param data.ids - Endpoint ID list scoped by targetId
   * @returns Promise<BatchDeleteEndpointsResponse>
   */
  static async batchDeleteEndpoints(data: BatchDeleteEndpointsRequest): Promise<BatchDeleteEndpointsResponse> {
    const response = await api.post<BatchDeleteEndpointsResponse>('/endpoints:batchDelete', {
      names: data.ids.map((id) => `targets/${data.targetId}/endpoints/${id}`),
    })
    return response.data
  }

  /** Export all endpoint URLs by target (text file, one per line) */
  static async exportEndpointsByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/endpoints/exportFiles/current`, {
      responseType: 'blob',
    })
    return response.data
  }

  /** Export all endpoint URLs by scan task (text file, one per line) */
  static async exportEndpointsByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/endpoints/exportFiles/current`, {
      responseType: 'blob',
    })
    return response.data
  }

}
