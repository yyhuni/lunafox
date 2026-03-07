"use client"

import { useQuery, keepPreviousData } from "@tanstack/react-query"
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { normalizePagination } from "@/hooks/_shared/pagination"
import {
  getAssetDeletedCount,
  resolveAssetBulkCreateToast,
} from "@/hooks/_shared/asset-mutation-helpers"
import { EndpointService } from "@/services/endpoint.service"
import type { 
  Endpoint, 
  CreateEndpointRequest,
  GetEndpointsRequest,
  GetEndpointsResponse,
  BatchDeleteEndpointsRequest
} from "@/types/endpoint.types"

type EndpointPageResponse = {
  results?: Endpoint[]
  total?: number
  page?: number
  pageSize?: number
  totalPages?: number
}

// Query Keys
const endpointKeyBase = createResourceKeys("endpoints", {
  list: (params: GetEndpointsRequest) => params,
  detail: (id: number) => id,
})

export const endpointKeys = {
  ...endpointKeyBase,
  byTarget: (targetId: number, params: GetEndpointsRequest) =>
    [...endpointKeyBase.all, 'target', targetId, params] as const,
  bySubdomain: (subdomainId: number, params: GetEndpointsRequest) =>
    [...endpointKeyBase.all, 'subdomain', subdomainId, params] as const,
  byScan: (scanId: number, params: GetEndpointsRequest) =>
    [...endpointKeyBase.all, 'scan', scanId, params] as const,
}

function endpointAllInvalidates() {
  return [{ queryKey: endpointKeys.all }]
}

// Get individual Endpoint details
export function useEndpoint(id: number) {
  return useQuery({
    queryKey: endpointKeys.detail(id),
    queryFn: () => EndpointService.getEndpointById(id),
    select: (response) => {
      // REST-style response: return data directly.
      return response as Endpoint
    },
    enabled: !!id,
  })
}

// Get endpoint list
export function useEndpoints(params?: GetEndpointsRequest) {
  const defaultParams: GetEndpointsRequest = {
    page: 1,
    pageSize: 10,
    ...params
  }
  
  return useQuery({
    queryKey: endpointKeys.list(defaultParams),
    queryFn: () => EndpointService.getEndpoints(defaultParams),
    select: (response) => {
      // REST-style response: return data directly.
      return response as GetEndpointsResponse
    },
  })
}

// Get a list of Endpoints based on target ID (using private routing)
export function useEndpointsByTarget(targetId: number, params?: Omit<GetEndpointsRequest, 'targetId'>, filter?: string) {
  const defaultParams: GetEndpointsRequest = {
    page: 1,
    pageSize: 10,
    ...params
  }
  
  return useQuery({
    queryKey: [...endpointKeys.byTarget(targetId, defaultParams), filter],
    queryFn: () => EndpointService.getEndpointsByTargetId(targetId, defaultParams, filter),
    select: (response) => {
      // REST-style response: return data directly.
      return response as GetEndpointsResponse
    },
    enabled: !!targetId,
    placeholderData: keepPreviousData,
  })
}

// Get Endpoint list based on scan ID (historical snapshot)
export function useScanEndpoints(scanId: number, params?: Omit<GetEndpointsRequest, 'targetId'>, options?: { enabled?: boolean }, filter?: string) {
  const defaultParams: GetEndpointsRequest = {
    page: 1,
    pageSize: 10,
    ...params,
  }

  return useQuery({
    queryKey: [...endpointKeys.byScan(scanId, defaultParams), filter],
    queryFn: () => EndpointService.getEndpointsByScanId(scanId, defaultParams, filter),
    enabled: options?.enabled !== undefined ? options.enabled : !!scanId,
    select: (response: EndpointPageResponse) => {
      // Backend uses shared pagination fields: results/total/page/pageSize/totalPages.
      return {
        endpoints: response.results || [],
        pagination: normalizePagination(response, defaultParams.page, defaultParams.pageSize),
      }
    },
    placeholderData: keepPreviousData,
  })
}

// Create Endpoint (fully automated)
export function useCreateEndpoint() {
  return useResourceMutation({
    mutationFn: (data: {
      endpoints: Array<CreateEndpointRequest>
    }) => EndpointService.createEndpoints(data),
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'create-endpoint',
    },
    invalidate: endpointAllInvalidates(),
    onSuccess: ({ data, toast }) => {
      const { createdCount, existedCount } = data

      if (existedCount > 0) {
        toast.warning('toast.asset.endpoint.create.partialSuccess', {
          success: createdCount,
          skipped: existedCount
        })
      } else {
        toast.success('toast.asset.endpoint.create.success', { count: createdCount })
      }
    },
    errorFallbackKey: 'toast.asset.endpoint.create.error',
  })
}

// Delete a single Endpoint
export function useDeleteEndpoint() {
  return useResourceMutation({
    mutationFn: (id: number) => EndpointService.deleteEndpoint(id),
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-endpoint-${id}`,
    },
    invalidate: endpointAllInvalidates(),
    onSuccess: ({ toast }) => {
      toast.success('toast.asset.endpoint.delete.success')
    },
    errorFallbackKey: 'toast.asset.endpoint.delete.error',
  })
}

// Deleting Endpoints in Batch
export function useBatchDeleteEndpoints() {
  return useResourceMutation({
    mutationFn: (data: BatchDeleteEndpointsRequest) => EndpointService.batchDeleteEndpoints(data),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'batch-delete-endpoints',
    },
    invalidate: endpointAllInvalidates(),
    onSuccess: ({ data, toast }) => {
      toast.success('toast.asset.endpoint.delete.bulkSuccess', {
        count: getAssetDeletedCount(data),
      })
    },
    errorFallbackKey: 'toast.asset.endpoint.delete.error',
  })
}

// Create endpoints in batches (bind to targets)
export function useBulkCreateEndpoints() {
  return useResourceMutation({
    mutationFn: (data: { targetId: number; urls: string[] }) =>
      EndpointService.bulkCreateEndpoints(data.targetId, data.urls),
    loadingToast: {
      key: 'common.status.batchCreating',
      params: {},
      id: 'bulk-create-endpoints',
    },
    invalidate: [
      ({ variables }) => ({
        queryKey: endpointKeys.byTarget(variables.targetId, {}),
        exact: false,
        refetchType: 'active',
      }),
      {
        queryKey: endpointKeys.all,
        exact: false,
        refetchType: 'active',
      },
    ],
    onSuccess: ({ data, toast }) => {
      const toastPayload = resolveAssetBulkCreateToast(data.createdCount, {
        success: 'toast.asset.endpoint.create.success',
        partial: 'toast.asset.endpoint.create.partialSuccess',
      })
      if (toastPayload.variant === 'success') {
        toast.success(toastPayload.key, toastPayload.params)
      } else {
        toast.warning(toastPayload.key, toastPayload.params)
      }
    },
    errorFallbackKey: 'toast.asset.endpoint.create.error',
  })
}
