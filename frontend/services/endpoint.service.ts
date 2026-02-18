import { api } from "@/lib/api-client"
import type { 
  Endpoint, 
  CreateEndpointRequest, 
  GetEndpointsRequest,
  GetEndpointsResponse,
  CreateEndpointsResponse,
  BatchDeleteEndpointsRequest,
  BatchDeleteEndpointsResponse
} from "@/types/endpoint.types"
import type { PaginatedResponse } from "@/types/api-response.types"
import {
  USE_MOCK,
  mockDelay,
  getMockEndpoints,
  getMockEndpointById,
  mockEndpoints,
  getMockTargetById,
  getMockScanById,
} from '@/mock'

// Bulk create endpoints response type
export interface BulkCreateEndpointsResponse {
  message: string
  createdCount: number
}

// Bulk delete response type
export interface BulkDeleteResponse {
  deletedCount: number
}

export class EndpointService {

  /**
   * Bulk delete endpoints
   * POST /api/endpoints/bulk-delete/
   */
  static async bulkDelete(ids: number[]): Promise<BulkDeleteResponse> {
    const response = await api.post<BulkDeleteResponse>(
      `/endpoints/bulk-delete/`,
      { ids }
    )
    return response.data
  }

  /**
   * Bulk create endpoints (bind to target)
   * POST /api/targets/{target_id}/endpoints/bulk-create/
   */
  static async bulkCreateEndpoints(
    targetId: number,
    urls: string[]
  ): Promise<BulkCreateEndpointsResponse> {
    const response = await api.post<BulkCreateEndpointsResponse>(
      `/targets/${targetId}/endpoints/bulk-create/`,
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
    if (USE_MOCK) {
      await mockDelay()
      const endpoint = getMockEndpointById(id)
      if (!endpoint) throw new Error('Endpoint not found')
      return endpoint
    }
    const response = await api.get<Endpoint>(`/endpoints/${id}/`)
    return response.data
  }

  /**
   * Get Endpoint list
   * @param params - Query parameters
   * @returns Promise<GetEndpointsResponse>
   */
  static async getEndpoints(params: GetEndpointsRequest): Promise<GetEndpointsResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockEndpoints(params)
    }
    // api-client.ts automatically converts camelCase params to snake_case
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
  static async getEndpointsByTargetId(targetId: number, params: GetEndpointsRequest, filter?: string): Promise<GetEndpointsResponse> {
    if (USE_MOCK) {
      await mockDelay()
      const page = params.page || 1
      const pageSize = params.pageSize || 10
      const search = (filter || params.search || "").toLowerCase()
      const target = getMockTargetById(targetId)
      const domain = target?.name?.toLowerCase()

      let filtered = mockEndpoints

      if (domain) {
        filtered = filtered.filter((ep) =>
          ep.url.toLowerCase().includes(domain) ||
          (ep.host || "").toLowerCase().includes(domain)
        )
      }

      if (search) {
        filtered = filtered.filter((ep) => {
          const url = ep.url.toLowerCase()
          const title = (ep.title || "").toLowerCase()
          const host = (ep.host || "").toLowerCase()
          return url.includes(search) || title.includes(search) || host.includes(search)
        })
      }

      const total = filtered.length
      const totalPages = Math.ceil(total / pageSize)
      const start = (page - 1) * pageSize
      const endpoints = filtered.slice(start, start + pageSize)

      return {
        endpoints,
        total,
        page,
        pageSize,
        totalPages,
      }
    }
    // api-client.ts automatically converts camelCase params to snake_case
    const response = await api.get<GetEndpointsResponse>(`/targets/${targetId}/endpoints/`, {
      params: { ...params, filter }
    })
    return response.data
  }

  /**
   * Get Endpoint list by scan ID (historical snapshot)
   * @param scanId - Scan task ID
   * @param params - Pagination and other query parameters
   * @param filter - Smart filter query string
   */
  static async getEndpointsByScanId(
    scanId: number,
    params: GetEndpointsRequest,
    filter?: string,
  ): Promise<PaginatedResponse<Endpoint>> {
    if (USE_MOCK) {
      await mockDelay()
      const page = params.page || 1
      const pageSize = params.pageSize || 10
      const search = (filter || params.search || "").toLowerCase()
      const scan = getMockScanById(scanId)
      const domain = scan?.target?.name?.toLowerCase()

      let filtered = mockEndpoints

      if (domain) {
        filtered = filtered.filter((ep) =>
          ep.url.toLowerCase().includes(domain) ||
          (ep.host || "").toLowerCase().includes(domain)
        )
      }

      if (search) {
        filtered = filtered.filter((ep) => {
          const url = ep.url.toLowerCase()
          const title = (ep.title || "").toLowerCase()
          const host = (ep.host || "").toLowerCase()
          return url.includes(search) || title.includes(search) || host.includes(search)
        })
      }

      const total = filtered.length
      const totalPages = Math.ceil(total / pageSize)
      const start = (page - 1) * pageSize
      const results = filtered.slice(start, start + pageSize)

      return {
        results,
        total,
        page,
        pageSize,
        totalPages,
      }
    }
    const response = await api.get<PaginatedResponse<Endpoint>>(`/scans/${scanId}/endpoints/`, {
      params: { ...params, filter },
    })
    return response.data
  }

  /**
   * Batch create Endpoints
   * @param data - Create request object
   * @param data.endpoints - Endpoint data array
   * @returns Promise<CreateEndpointsResponse>
   */
  static async createEndpoints(data: { endpoints: Array<CreateEndpointRequest> }): Promise<CreateEndpointsResponse> {
    // api-client.ts automatically converts camelCase request body to snake_case
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
   * @param data.endpointIds - Endpoint ID list
   * @returns Promise<BatchDeleteEndpointsResponse>
   */
  static async batchDeleteEndpoints(data: BatchDeleteEndpointsRequest): Promise<BatchDeleteEndpointsResponse> {
    // api-client.ts automatically converts camelCase request body to snake_case
    const response = await api.post<BatchDeleteEndpointsResponse>('/endpoints/batch-delete/', data)
    return response.data
  }

  /** Export all endpoint URLs by target (text file, one per line) */
  static async exportEndpointsByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/endpoints/export/`, {
      responseType: 'blob',
    })
    return response.data
  }

  /** Export all endpoint URLs by scan task (text file, one per line) */
  static async exportEndpointsByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/endpoints/export/`, {
      responseType: 'blob',
    })
    return response.data
  }

}
