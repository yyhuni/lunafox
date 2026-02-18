import { api } from "@/lib/api-client"
import type { Subdomain, GetSubdomainsResponse, GetAllSubdomainsParams, GetAllSubdomainsResponse, GetSubdomainByIDResponse, BatchCreateSubdomainsResponse } from "@/types/subdomain.types"
import type { PaginatedResponse } from "@/types/api-response.types"
import {
  USE_MOCK,
  mockDelay,
  getMockSubdomains,
  getMockSubdomainById,
  mockSubdomains,
  getMockTargetById,
  getMockScanById,
} from '@/mock'

// Bulk create subdomains response type
export interface BulkCreateSubdomainsResponse {
  message: string
  createdCount: number
  skippedCount: number
  invalidCount: number
  mismatchedCount: number
  totalReceived: number
}

type BasicSubdomainResult = {
  id: number
  name: string
  createdAt: string
}

type ScanSubdomainResult = BasicSubdomainResult & {
  cname: string[]
  isCdn: boolean
  cdnName: string
  ports: Array<{
    number: number
    serviceName: string
    description: string
    isUncommon: boolean
  }>
  ipAddresses: string[]
}

export class SubdomainService {
  /**
   * Bulk create subdomains (bind to target)
   * POST /api/targets/{target_id}/subdomains/bulk-create/
   */
  static async bulkCreateSubdomains(
    targetId: number,
    subdomains: string[]
  ): Promise<BulkCreateSubdomainsResponse> {
    const response = await api.post<BulkCreateSubdomainsResponse>(
      `/targets/${targetId}/subdomains/bulk-create/`,
      { names: subdomains }
    )
    return response.data
  }
  // ========== Subdomain basic operations ==========

  /**
   * Bulk create subdomains (bind to assets)
   */
  static async createSubdomains(data: {
    domains: Array<{
      name: string
    }>
    assetId: number
  }): Promise<BatchCreateSubdomainsResponse> {
    const response = await api.post<BatchCreateSubdomainsResponse>('/domains/create/', {
      domains: data.domains,
      assetId: data.assetId  // [OK] CamelCase, interceptor converts to asset_id
    })
    return response.data
  }

  /**
   * Get single subdomain details
   */
  static async getSubdomainById(id: string | number): Promise<GetSubdomainByIDResponse> {
    if (USE_MOCK) {
      await mockDelay()
      const subdomain = getMockSubdomainById(Number(id))
      if (!subdomain) throw new Error('Subdomain not found')
      return subdomain
    }
    const response = await api.get<GetSubdomainByIDResponse>(`/domains/${id}/`)
    return response.data
  }

  /**
   * Update subdomain information (PATCH)
   */
  static async updateSubdomain(data: {
    id: number
    name?: string
    description?: string
  }): Promise<Subdomain> {
    const requestBody: { name?: string; description?: string } = {}
    if (data.name !== undefined) requestBody.name = data.name
    if (data.description !== undefined) requestBody.description = data.description
    const response = await api.patch<Subdomain>(`/domains/${data.id}/`, requestBody)
    return response.data
  }

  /** Bulk delete subdomains (supports single or multiple, using unified interface) */
  static async bulkDeleteSubdomains(
    ids: number[]
  ): Promise<{
    message: string
    deletedCount: number
    requestedIds: number[]
    cascadeDeleted: Record<string, number>
  }> {
    const response = await api.post<{
      message: string
      deletedCount: number
      requestedIds: number[]
      cascadeDeleted: Record<string, number>
    }>(
      `/subdomains/bulk-delete/`,
      { ids }
    )
    return response.data
  }

  /** Delete single subdomain (using separate DELETE API) */
  static async deleteSubdomain(id: number): Promise<{
    message: string
    subdomainId: number
    subdomainName: string
    deletedCount: number
    deletedSubdomains: string[]
    detail: {
      phase1: string
      phase2: string
    }
  }> {
    const response = await api.delete<{
      message: string
      subdomainId: number
      subdomainName: string
      deletedCount: number
      deletedSubdomains: string[]
      detail: {
        phase1: string
        phase2: string
      }
    }>(`/subdomains/${id}/`)
    return response.data
  }

  /** Bulk delete subdomains (alias, compatible with old code) */
  static async batchDeleteSubdomains(ids: number[]): Promise<{
    message: string
    deletedCount: number
    requestedIds: number[]
    cascadeDeleted: Record<string, number>
  }> {
    return this.bulkDeleteSubdomains(ids)
  }

  /** Batch remove subdomains from organization */
  static async batchDeleteSubdomainsFromOrganization(data: {
    organizationId: number
    domainIds: number[]
  }): Promise<{
    message: string
    successCount: number
    failedCount: number
  }> {
    const response = await api.post<{ message: string; successCount: number; failedCount: number }>(
      `/organizations/${data.organizationId}/domains/batch-remove/`,
      {
        domainIds: data.domainIds, // Interceptor converts to domain_ids
      }
    )
    return response.data
  }

  /** Get organization's subdomain list (server-side pagination) */
  static async getSubdomainsByOrgId(
    organizationId: number,
    params?: {
      page?: number
      pageSize?: number
    }
  ): Promise<GetSubdomainsResponse> {
    const response = await api.get<GetSubdomainsResponse>(
      `/organizations/${organizationId}/domains/`,
      {
        params: {
          page: params?.page || 1,
          pageSize: params?.pageSize || 10,
        }
      }
    )
    return response.data
  }

  /** Get all subdomains list (server-side pagination) */
  static async getAllSubdomains(params?: GetAllSubdomainsParams): Promise<GetAllSubdomainsResponse> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockSubdomains(params)
    }
    const response = await api.get<GetAllSubdomainsResponse>('/domains/', {
      params: {
        page: params?.page || 1,
        pageSize: params?.pageSize || 10,
      }
    })
    return response.data
  }

  /** Get target's subdomain list (supports pagination and filtering) */
  static async getSubdomainsByTargetId(
    targetId: number,
    params?: {
      page?: number
      pageSize?: number
      filter?: string
    }
  ): Promise<PaginatedResponse<BasicSubdomainResult>> {
    if (USE_MOCK) {
      await mockDelay()
      const page = params?.page || 1
      const pageSize = params?.pageSize || 10
      const filter = params?.filter?.toLowerCase() || ""
      const target = getMockTargetById(targetId)
      const domain = target?.name?.toLowerCase()

      let filtered = mockSubdomains

      if (domain) {
        filtered = filtered.filter((sub) => sub.name.toLowerCase().includes(domain))
      }

      if (filter) {
        filtered = filtered.filter((sub) => sub.name.toLowerCase().includes(filter))
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
    const response = await api.get<PaginatedResponse<BasicSubdomainResult>>(`/targets/${targetId}/subdomains/`, {
      params: {
        page: params?.page || 1,
        pageSize: params?.pageSize || 10,
        ...(params?.filter && { filter: params.filter }),
      }
    })
    return response.data
  }

  /** Get scan's subdomain list (supports pagination and filtering) */
  static async getSubdomainsByScanId(
    scanId: number,
    params?: {
      page?: number
      pageSize?: number
      filter?: string
    }
  ): Promise<PaginatedResponse<ScanSubdomainResult>> {
    if (USE_MOCK) {
      await mockDelay()
      const page = params?.page || 1
      const pageSize = params?.pageSize || 10
      const filter = params?.filter?.toLowerCase() || ""
      const scan = getMockScanById(scanId)
      const domain = scan?.target?.name?.toLowerCase()

      let filtered = mockSubdomains

      if (domain) {
        filtered = filtered.filter((sub) => sub.name.toLowerCase().includes(domain))
      }

      if (filter) {
        filtered = filtered.filter((sub) => sub.name.toLowerCase().includes(filter))
      }

      const total = filtered.length
      const totalPages = Math.ceil(total / pageSize)
      const start = (page - 1) * pageSize
      const results = filtered.slice(start, start + pageSize).map((sub) => ({
        id: sub.id,
        name: sub.name,
        createdAt: sub.createdAt,
        cname: [],
        isCdn: false,
        cdnName: "",
        ports: [],
        ipAddresses: [],
      }))

      return {
        results,
        total,
        page,
        pageSize,
        totalPages,
      }
    }
    const response = await api.get<PaginatedResponse<ScanSubdomainResult>>(`/scans/${scanId}/subdomains/`, {
      params: {
        page: params?.page || 1,
        pageSize: params?.pageSize || 10,
        ...(params?.filter && { filter: params.filter }),
      }
    })
    return response.data
  }

  /** Export all subdomain names by target (text file, one per line) */
  static async exportSubdomainsByTargetId(targetId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/targets/${targetId}/subdomains/export/`, {
      responseType: 'blob',
    })
    return response.data
  }

  /** Export all subdomain names by scan task (text file, one per line) */
  static async exportSubdomainsByScanId(scanId: number): Promise<Blob> {
    const response = await api.get<Blob>(`/scans/${scanId}/subdomains/export/`, {
      responseType: 'blob',
    })
    return response.data
  }
}
