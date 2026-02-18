import { api } from "@/lib/api-client"
import type { Organization, OrganizationsResponse } from "@/types/organization.types"
import type { TargetsResponse } from "@/types/target.types"
import { USE_MOCK, mockDelay, getMockOrganizations, mockOrganizations } from '@/mock'


export class OrganizationService {
  // ========== Organization basic operations ==========

  /**
   * Get organization list
   * @param params - Query parameter object
   * @param params.page - Current page number, 1-based
   * @param params.pageSize - Page size
   * @param params.filter - Filter string (plain text or smart filter syntax)
   * @returns Promise<OrganizationsResponse<Organization>>
   * @description Backend is fixed to sort by update time in descending order, does not support custom sorting
   */
  static async getOrganizations(params?: {
    page?: number
    pageSize?: number
    filter?: string
  }): Promise<OrganizationsResponse<Organization>> {
    if (USE_MOCK) {
      await mockDelay()
      return getMockOrganizations(params)
    }
    const response = await api.get<OrganizationsResponse<Organization>>(
      '/organizations/',
      { params }
    )
    return response.data
  }

  /**
   * Get single organization details
   * @param id - Organization ID
   * @returns Promise<Organization>
   */
  static async getOrganizationById(id: string | number): Promise<Organization> {
    if (USE_MOCK) {
      await mockDelay()
      const org = mockOrganizations.find(o => o.id === Number(id))
      if (!org) throw new Error('Organization not found')
      return org
    }
    const response = await api.get<Organization>(`/organizations/${id}/`)
    return response.data
  }

  /**
   * Get organization's target list
   * @param id - Organization ID
   * @param params - Query parameters
   * @returns Promise<any>
   */
  static async getOrganizationTargets(id: string | number, params?: {
    page?: number
    pageSize?: number
    search?: string
    type?: string
  }): Promise<TargetsResponse> {
    const response = await api.get<TargetsResponse>(
      `/organizations/${id}/targets/`,
      { params }
    )
    return response.data
  }

  /**
   * Create new organization
   * @param data - Organization information object
   * @param data.name - Organization name
   * @param data.description - Organization description
   * @returns Promise<Organization> - Organization information object after successful creation
   */
  static async createOrganization(data: {
    name: string
    description: string
  }): Promise<Organization> {
    const response = await api.post<Organization>('/organizations/', data)
    return response.data
  }

  /**
   * Update organization information
   * @param data - Organization information object
   * @param data.id - Organization ID, number or string type
   * @param data.name - Organization name
   * @param data.description - Organization description
   * @returns Promise<Organization> - Organization information object after successful update
   */
  static async updateOrganization(data: {
    id: string | number
    name: string
    description: string
  }): Promise<Organization> {
    const response = await api.put<Organization>(`/organizations/${data.id}/`, {
      name: data.name,
      description: data.description
    })
    return response.data
  }
  /**
   * Delete organization (using separate DELETE API)
   * 
   * @param id - Organization ID, number type
   * @returns Promise<Delete response>
   */
  static async deleteOrganization(id: number): Promise<{
    message: string
    organizationId: number
    organizationName: string
    deletedCount: number
    deletedOrganizations: string[]
    detail: {
      phase1: string
      phase2: string
    }
  }> {
    const response = await api.delete<{
      message: string
      organizationId: number
      organizationName: string
      deletedCount: number
      deletedOrganizations: string[]
      detail: {
        phase1: string
        phase2: string
      }
    }>(`/organizations/${id}/`)
    return response.data
  }

  /**
   * Batch delete organizations
   * @param organizationIds - Array of organization IDs, number type
   * @returns Promise<{ message: string; deletedOrganizationCount: number }>
   * 
   * Note: Deleting organizations will not delete domain entities, only unlink associations
   */
  static async batchDeleteOrganizations(organizationIds: number[]): Promise<{
    message: string
    deletedCount: number
    deletedOrganizations: string[]
  }> {
    const response = await api.post<{
      message: string
      deletedCount: number
      deletedOrganizations: string[]
    }>('/organizations/bulk-delete/', {
      ids: organizationIds  // Backend expects 'ids' parameter
    })
    return response.data
  }

  // ========== Organization and target association operations ==========

  /**
   * Link target to organization (single)
   * @param data - Link request object
   * @param data.organizationId - Organization ID
   * @param data.targetId - Target ID
   * @returns Promise<{ message: string }>
   */
  static async linkTargetToOrganization(data: {
    organizationId: number
    targetId: number
  }): Promise<{ message: string }> {
    const response = await api.post<{ message: string }>(
      `/organizations/${data.organizationId}/targets/`,
      {
        targetId: data.targetId  // Interceptor will convert to target_id
      }
    )
    return response.data
  }

  /**
   * Remove targets from organization (batch)
   * @param data - Remove request object
   * @param data.organizationId - Organization ID
   * @param data.targetIds - Array of target IDs
   * @returns Promise<{ unlinkedCount: number; message: string }>
   */
  static async unlinkTargetsFromOrganization(data: {
    organizationId: number
    targetIds: number[]
  }): Promise<{ unlinkedCount: number; message: string }> {
    const response = await api.post<{ unlinkedCount: number; message: string }>(
      `/organizations/${data.organizationId}/unlink_targets/`,
      {
        targetIds: data.targetIds  // Interceptor will convert to target_ids
      }
    )
    return response.data
  }

}
