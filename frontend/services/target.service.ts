/**
 * Target Service - Target management API
 */
import { api } from '@/lib/api-client'
import { organizationName, targetName } from '@/lib/resource-name'
import type {
  Target,
  TargetDetail,
  TargetsResponse,
  CreateTargetRequest,
  UpdateTargetRequest,
  BatchDeleteTargetsRequest,
  BatchDeleteTargetsResponse,
  BatchCreateTargetsRequest,
  BatchCreateTargetsResponse,
} from '@/types/target.types'
import type { Endpoint, EndpointListQueryParams, GetEndpointsResponse } from '@/types/endpoint.types'

type TargetAipDto = {
  id: number
  name: string
  displayName: string
  type: Target["type"]
  createdAt: string
  lastScannedAt?: string
  organizations?: Array<{
    id: number
    name: string
    displayName: string
  }>
  summary?: TargetDetail["summary"]
}

export type TargetListRequestParams = {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
}

type AipEndpointListResponse = {
  results?: Endpoint[]
  total?: number
  totalSize?: number
  nextPageToken?: string
  page?: number
  pageSize?: number
  totalPages?: number
}

type TargetsAipResponse = {
  results: TargetAipDto[]
  total?: number
  totalSize?: number
  nextPageToken?: string
  page?: number
  pageSize?: number
  totalPages?: number
}

type BatchCreateTargetsAipResponse = Omit<BatchCreateTargetsResponse, "reusedCount"> & {
  reusedCount?: number
}

function toTarget(dto: TargetAipDto): Target {
  return {
    id: dto.id,
    name: dto.displayName,
    resourceName: dto.name,
    type: dto.type,
    createdAt: dto.createdAt,
    lastScannedAt: dto.lastScannedAt,
    organizations: dto.organizations?.map((organization) => ({
      id: organization.id,
      name: organization.displayName,
      resourceName: organization.name,
    })),
  }
}

function toTargetDetail(dto: TargetAipDto): TargetDetail {
  return {
    ...toTarget(dto),
    summary: dto.summary ?? {
      subdomains: 0,
      websites: 0,
      endpoints: 0,
      ips: 0,
      directories: 0,
      screenshots: 0,
      vulnerabilities: {
        total: 0,
        critical: 0,
        high: 0,
        medium: 0,
        low: 0,
      },
    },
  }
}

function toTargetsResponse(response: TargetsAipResponse, fallbackParams?: TargetListRequestParams): TargetsResponse {
  const pageSize = response.pageSize ?? fallbackParams?.pageSize ?? 10
  const total = response.totalSize ?? response.total ?? 0

  return {
    ...response,
    results: response.results.map(toTarget),
    total,
    totalSize: total,
    nextPageToken: response.nextPageToken,
    page: response.page ?? 1,
    pageSize,
    totalPages: response.totalPages ?? Math.max(1, Math.ceil(total / pageSize)),
  }
}

function toBatchCreateTargetsResponse(
  response: BatchCreateTargetsAipResponse
): BatchCreateTargetsResponse {
  return {
    createdCount: response.createdCount,
    reusedCount: response.reusedCount ?? 0,
    failedCount: response.failedCount,
    failedTargets: response.failedTargets,
    message: response.message,
  }
}

/**
 * Get all targets list (paginated)
 */
export async function getTargets(params: TargetListRequestParams = {}): Promise<TargetsResponse> {
  const response = await api.get<TargetsAipResponse>('/targets', {
    params: {
      pageSize: params.pageSize ?? 10,
      ...(params.pageToken && { pageToken: params.pageToken }),
      ...(params.filter && { filter: params.filter }),
      ...(params.orderBy && { orderBy: params.orderBy }),
    },
  })
  return toTargetsResponse(response.data, params)
}

/**
 * Get single target details
 */
export async function getTargetById(id: number): Promise<Target> {
  const response = await api.get<TargetAipDto>(`/targets/${id}`)
  return toTargetDetail(response.data)
}

/**
 * Create target
 */
export async function createTarget(data: CreateTargetRequest): Promise<Target> {
  const response = await api.post<TargetAipDto>('/targets', data)
  return toTarget(response.data)
}

/**
 * Update target
 */
export async function updateTarget(id: number, data: UpdateTargetRequest): Promise<Target> {
  const response = await api.patch<TargetAipDto>(`/targets/${id}`, {
    name: targetName(id),
    displayName: data.name,
    updateMask: "displayName",
  })
  return toTarget(response.data)
}

/**
 * Delete single target (RESTful 204 No Content)
 */
export async function deleteTarget(id: number): Promise<void> {
  await api.delete(`/targets/${id}`)
}

/**
 * Batch delete targets
 */
export async function batchDeleteTargets(
  data: BatchDeleteTargetsRequest
): Promise<BatchDeleteTargetsResponse> {
  const response = await api.post<BatchDeleteTargetsResponse>('/targets:batchDelete', {
    names: data.ids.map(targetName),
  })
  return response.data
}

/**
 * Batch create targets.
 * Backend contract: max 5,000 targets per request (BatchCreateTargetRequest.Targets).
 * See server/internal/modules/catalog/dto/target_dto.go.
 */
export async function batchCreateTargets(
  data: BatchCreateTargetsRequest
): Promise<BatchCreateTargetsResponse> {
  const organizationIds = data.organizationIds ?? []
  if (organizationIds.length > 1) {
    throw new Error('batchCreateTargets supports one organization until the backend accepts organizations[]')
  }

  const response = await api.post<BatchCreateTargetsAipResponse>('/targets:batchCreate', {
    targets: data.targets,
    ...(organizationIds.length === 1 ? { organization: organizationName(organizationIds[0]) } : {}),
  })
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
  return toBatchCreateTargetsResponse(response.data)
}

/**
 * Get target's organization list
 */
export async function getTargetOrganizations(id: number, page = 1, pageSize = 10) {
  const response = await api.get(`/targets/${id}/organizations`, { params: { page, pageSize } })
  return response.data
}

/**
 * Link organizations to target
 */
export async function linkTargetOrganizations(
  id: number,
  organizationIds: number[]
): Promise<{ message: string }> {
  await Promise.all(
    organizationIds.map((organizationId) =>
      api.post<{ message: string }>(`/${organizationName(organizationId)}/targets:batchLink`, {
        targets: [targetName(id)],
      }),
    ),
  )
  return { message: "ok" }
}

/**
 * Unlink target from organizations
 */
export async function unlinkTargetOrganizations(
  id: number,
  organizationIds: number[]
): Promise<{ message: string }> {
  await Promise.all(
    organizationIds.map((organizationId) =>
      api.post<{ message: string }>(`/${organizationName(organizationId)}/targets:batchUnlink`, {
        targets: [targetName(id)],
      }),
    ),
  )
  return { message: "ok" }
}

/**
 * Get target's endpoint list
 */
export async function getTargetEndpoints(
  id: number,
  params: EndpointListQueryParams = {}
): Promise<GetEndpointsResponse> {
  const response = await api.get<AipEndpointListResponse>(`/targets/${id}/endpoints/`, {
    params: {
      pageSize: params.pageSize ?? 10,
      ...(params.pageToken && { pageToken: params.pageToken }),
      ...(params.filter && { filter: params.filter }),
      ...(params.orderBy && { orderBy: params.orderBy }),
    },
  })
  const results = response.data.results ?? []
  const pageSize = response.data.pageSize ?? params.pageSize ?? 10
  const total = response.data.total ?? response.data.totalSize ?? results.length

  return {
    results,
    total,
    page: response.data.page ?? 1,
    pageSize,
    totalPages: response.data.totalPages ?? (total === 0 ? 0 : Math.ceil(total / pageSize)),
    totalSize: response.data.totalSize,
    nextPageToken: response.data.nextPageToken,
  }
}
