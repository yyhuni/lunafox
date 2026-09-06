import { api } from "@/lib/api-client"
import { organizationName, targetName } from "@/lib/resource-name"
import type { Organization, OrganizationsResponse } from "@/types/organization.types"
import type { Target, TargetsResponse } from "@/types/target.types"

type OrganizationAipDto = {
  id: number
  name: string
  displayName: string
  description: string
  createdAt: string
  updatedAt?: string
  targetCount?: number
  domainCount?: number
  endpointCount?: number
}

type TargetAipDto = {
  id: number
  name: string
  displayName: string
  type: Target["type"]
  createdAt: string
  lastScannedAt?: string
}

type AipPaginatedResponse<T> = {
  results: T[]
  totalSize?: number
  total?: number
  nextPageToken?: string
  page?: number
  pageSize?: number
  totalPages?: number
}

type OrganizationListParams = {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

type OrganizationTargetListParams = {
  pageSize?: number
  pageToken?: string
  search?: string
  filter?: string
  type?: string
}

type OrganizationDeleteResponse = {
  id?: number
  organizationId?: number
  organizationName: string
  deletedCount: number
}

type OrganizationBatchDeleteResponse = {
  deletedCount: number
}

function toOrganization(dto: OrganizationAipDto): Organization {
  return {
    id: dto.id,
    name: dto.displayName,
    resourceName: dto.name,
    description: dto.description,
    createdAt: dto.createdAt,
    updatedAt: dto.updatedAt ?? dto.createdAt,
    ...(dto.targetCount !== undefined ? { targetCount: dto.targetCount } : {}),
    ...(dto.domainCount !== undefined ? { domainCount: dto.domainCount } : {}),
    ...(dto.endpointCount !== undefined ? { endpointCount: dto.endpointCount } : {}),
  }
}

function toTarget(dto: TargetAipDto): Target {
  return {
    id: dto.id,
    name: dto.displayName,
    resourceName: dto.name,
    type: dto.type,
    createdAt: dto.createdAt,
    lastScannedAt: dto.lastScannedAt,
  }
}

function toOrganizationsResponse(
  response: AipPaginatedResponse<OrganizationAipDto>,
  fallbackParams?: { pageSize?: number }
): OrganizationsResponse<Organization> {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? fallbackParams?.pageSize ?? 10
  const total = response.totalSize ?? response.total ?? 0
  return {
    results: response.results.map(toOrganization),
    total,
    totalSize: total,
    nextPageToken: response.nextPageToken || undefined,
    page,
    pageSize,
    totalPages: response.totalPages ?? Math.ceil(total / pageSize),
  }
}

function toTargetsResponse(
  response: AipPaginatedResponse<TargetAipDto>,
  fallbackParams?: { pageSize?: number }
): TargetsResponse {
  const page = response.page ?? 1
  const pageSize = response.pageSize ?? fallbackParams?.pageSize ?? 10
  const total = response.totalSize ?? response.total ?? 0
  return {
    results: response.results.map(toTarget),
    total,
    totalSize: total,
    nextPageToken: response.nextPageToken || undefined,
    page,
    pageSize,
    totalPages: response.totalPages ?? Math.ceil(total / pageSize),
  }
}

export class OrganizationService {
  // ========== Organization basic operations ==========
  /**
   * Get organization list
   * @param params - Query parameter object
   * @param params.pageSize - Page size
   * @param params.pageToken - Opaque page token returned by the backend
   * @param params.filter - Canonical AIP filter string
   * @param params.orderBy - Canonical AIP orderBy string
   * @returns Promise<OrganizationsResponse<Organization>>
   */
  static async getOrganizations(params?: OrganizationListParams): Promise<OrganizationsResponse<Organization>> {
    const response = await api.get<AipPaginatedResponse<OrganizationAipDto>>(
      '/organizations',
      { params }
    )
    return toOrganizationsResponse(response.data, params)
  }

  /**
   * Get single organization details
   * @param id - Organization ID
   * @returns Promise<Organization>
   */
  static async getOrganizationById(id: string | number): Promise<Organization> {
    const response = await api.get<OrganizationAipDto>(`/organizations/${id}`)
    return toOrganization(response.data)
  }

  /**
   * Get organization's target list
   * @param id - Organization ID
   * @param params - Query parameters
   * @param params.pageToken - Opaque token returned by the preceding response
   * @returns Paginated organization targets
   */
  static async getOrganizationTargets(
    id: string | number,
    params?: OrganizationTargetListParams
  ): Promise<TargetsResponse> {
    const { filter, search, ...restParams } = params ?? {}
    const resolvedFilter = filter ?? search
    const response = await api.get<AipPaginatedResponse<TargetAipDto>>(
      `/organizations/${id}/targets`,
      {
        params: {
          ...restParams,
          ...(resolvedFilter ? { filter: resolvedFilter } : {}),
        },
      }
    )
    return toTargetsResponse(response.data, params)
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
    const response = await api.post<OrganizationAipDto>('/organizations', data)
    return toOrganization(response.data)
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
    const id = Number(data.id)
    const response = await api.patch<OrganizationAipDto>(`/organizations/${data.id}`, {
      name: organizationName(id),
      displayName: data.name,
      description: data.description,
      updateMask: "displayName,description",
    })
    return toOrganization(response.data)
  }
  /**
   * Delete organization (using separate DELETE API)
   * 
   * @param id - Organization ID, number type
   * @returns Promise<Delete response>
   */
  static async deleteOrganization(id: number): Promise<OrganizationDeleteResponse> {
    const response = await api.delete<Partial<OrganizationDeleteResponse> | undefined>(
      `/organizations/${id}`
    )
    return {
      id,
      organizationName: response.data?.organizationName ?? organizationName(id),
      deletedCount: response.data?.deletedCount ?? 1,
    }
  }

  /**
   * Batch delete organizations
   * @param organizationIds - Array of organization IDs, number type
   * @returns Promise<{ message: string; deletedOrganizationCount: number }>
   * 
   * Note: Deleting organizations will not delete domain entities, only unlink associations
   */
  static async batchDeleteOrganizations(organizationIds: number[]): Promise<OrganizationBatchDeleteResponse> {
    const response = await api.post<OrganizationBatchDeleteResponse>('/organizations:batchDelete', {
      names: organizationIds.map(organizationName)
    })
    return {
      deletedCount: response.data.deletedCount,
    }
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
      `/organizations/${data.organizationId}/targets:batchLink`,
      {
        targets: [targetName(data.targetId)]
      }
    )
    return response.data
  }

  /**
   * Remove targets from organization (batch)
   * Backend contract: max 5,000 targets per link/unlink request.
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
      `/organizations/${data.organizationId}/targets:batchUnlink`,
      {
        targets: data.targetIds.map(targetName)
      }
    )
    return response.data
  }

}
