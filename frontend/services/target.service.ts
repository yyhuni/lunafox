/**
 * Target Service - Target management API
 */
import { api } from '@/lib/api-client'
import type {
  Target,
  TargetsResponse,
  CreateTargetRequest,
  UpdateTargetRequest,
  BatchDeleteTargetsRequest,
  BatchDeleteTargetsResponse,
  BatchCreateTargetsRequest,
  BatchCreateTargetsResponse,
} from '@/types/target.types'
import type { Endpoint } from '@/types/endpoint.types'
import type { PaginatedResponse } from '@/types/api-response.types'
import {
  USE_MOCK,
  mockDelay,
  getMockTargets,
  getMockTargetById,
  getMockTargetBlacklist,
  updateMockTargetBlacklist,
  mockEndpoints,
} from '@/mock'

/**
 * Get all targets list (paginated)
 */
export async function getTargets(page = 1, pageSize = 10, filter?: string, type?: string): Promise<TargetsResponse> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockTargets({ page, pageSize, search: filter })
  }
  const response = await api.get<TargetsResponse>('/targets/', {
    params: {
      page,
      pageSize,
      ...(filter && { filter }),
      ...(type && { type }),
    },
  })
  return response.data
}

/**
 * Get single target details
 */
export async function getTargetById(id: number): Promise<Target> {
  if (USE_MOCK) {
    await mockDelay()
    const target = getMockTargetById(id)
    if (!target) throw new Error('Target not found')
    return target
  }
  const response = await api.get<Target>(`/targets/${id}/`)
  return response.data
}

/**
 * Create target
 */
export async function createTarget(data: CreateTargetRequest): Promise<Target> {
  const response = await api.post<Target>('/targets/', data)
  return response.data
}

/**
 * Update target
 */
export async function updateTarget(id: number, data: UpdateTargetRequest): Promise<Target> {
  const response = await api.patch<Target>(`/targets/${id}/`, data)
  return response.data
}

/**
 * Delete single target (RESTful 204 No Content)
 */
export async function deleteTarget(id: number): Promise<void> {
  await api.delete(`/targets/${id}/`)
}

/**
 * Batch delete targets
 */
export async function batchDeleteTargets(
  data: BatchDeleteTargetsRequest
): Promise<BatchDeleteTargetsResponse> {
  const response = await api.post<BatchDeleteTargetsResponse>('/targets/bulk-delete/', data)
  return response.data
}

/**
 * Batch create targets
 */
export async function batchCreateTargets(
  data: BatchCreateTargetsRequest
): Promise<BatchCreateTargetsResponse> {
  const response = await api.post<BatchCreateTargetsResponse>('/targets/bulk-create/', data)
  // Handle 204 No Content response - return default success response
  if (response.status === 204 || !response.data) {
    return {
      createdCount: data.targets.length,
      reusedCount: 0,
      failedCount: 0,
      failedTargets: [],
      message: 'success',
    }
  }
  return response.data
}

/**
 * Get target's organization list
 */
export async function getTargetOrganizations(id: number, page = 1, pageSize = 10) {
  const response = await api.get(`/targets/${id}/organizations/`, { params: { page, pageSize } })
  return response.data
}

/**
 * Link organizations to target
 */
export async function linkTargetOrganizations(
  id: number,
  organizationIds: number[]
): Promise<{ message: string }> {
  const response = await api.post<{ message: string }>(`/targets/${id}/organizations/`, { organizationIds })
  return response.data
}

/**
 * Unlink target from organizations
 */
export async function unlinkTargetOrganizations(
  id: number,
  organizationIds: number[]
): Promise<{ message: string }> {
  const response = await api.post<{ message: string }>(`/targets/${id}/organizations/unlink/`, { organizationIds })
  return response.data
}

/**
 * Get target's endpoint list
 */
export async function getTargetEndpoints(
  id: number,
  page = 1,
  pageSize = 10,
  filter?: string
): Promise<PaginatedResponse<Endpoint>> {
  if (USE_MOCK) {
    await mockDelay()
    const target = getMockTargetById(id)
    const domain = target?.name?.toLowerCase()
    const search = filter?.toLowerCase() || ""

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
  const response = await api.get<PaginatedResponse<Endpoint>>(`/targets/${id}/endpoints/`, {
    params: {
      page,
      pageSize,
      ...(filter && { filter }),
    },
  })
  return response.data
}

/**
 * Get target's blacklist rules
 */
export async function getTargetBlacklist(id: number): Promise<{ patterns: string[] }> {
  if (USE_MOCK) {
    await mockDelay()
    return getMockTargetBlacklist(id)
  }
  const response = await api.get<{ patterns: string[] }>(`/targets/${id}/blacklist/`)
  return response.data
}

/**
 * Update target's blacklist rules (full replace)
 */
export async function updateTargetBlacklist(
  id: number,
  patterns: string[]
): Promise<{ count: number }> {
  if (USE_MOCK) {
    await mockDelay()
    const result = updateMockTargetBlacklist(id, { patterns })
    return { count: result.patterns.length }
  }
  const response = await api.put<{ count: number }>(`/targets/${id}/blacklist/`, { patterns })
  return response.data
}
