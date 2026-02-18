"use client"

import { useQuery, keepPreviousData } from "@tanstack/react-query"
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { normalizePagination } from "@/hooks/_shared/pagination"
import {
  getSubdomainBatchDeleteCount,
  getSubdomainBatchDeleteFromOrgCount,
  resolveSubdomainCreateToast,
} from "@/hooks/_shared/subdomain-mutation-helpers"
import { SubdomainService } from "@/services/subdomain.service"
import { OrganizationService } from "@/services/organization.service"
import type { GetAllSubdomainsParams } from "@/types/subdomain.types"
import type { PaginationParams } from "@/types/common.types"

// Query Keys
export const subdomainKeys = createResourceKeys("subdomains", {
  list: (params: PaginationParams & { organizationId?: string }) => params,
  detail: (id: number) => id,
})

function subdomainCascadeInvalidates() {
  return [
    { queryKey: subdomainKeys.all },
    { queryKey: ['targets'] },
    { queryKey: ['scans'] },
    { queryKey: ['organizations'] },
  ]
}

// Get details of a single subdomain
export function useSubdomain(id: number) {
  return useQuery({
    queryKey: subdomainKeys.detail(id),
    queryFn: () => SubdomainService.getSubdomainById(id),
    enabled: !!id,
  })
}

// Get a list of your organization's subdomains
export function useOrganizationSubdomains(
  organizationId: number,
  params?: { page?: number; pageSize?: number },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ['organizations', 'detail', organizationId, 'subdomains', {
      page: params?.page,
      pageSize: params?.pageSize,
    }],
    queryFn: () => SubdomainService.getSubdomainsByOrgId(organizationId, {
      page: params?.page || 1,
      pageSize: params?.pageSize || 10,
    }),
    enabled: options?.enabled !== undefined ? options.enabled : true,
    select: (response) => ({
      domains: response.domains || [],
      pagination: normalizePagination(response, params?.page ?? 1, params?.pageSize ?? 10),
    }),
  })
}

// Create subdomain (bind to asset)
export function useCreateSubdomain() {
  return useResourceMutation({
    mutationFn: (data: { domains: Array<{ name: string }>; assetId: number }) =>
      SubdomainService.createSubdomains(data),
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'create-subdomain',
    },
    invalidate: [
      { queryKey: subdomainKeys.all },
      { queryKey: ['assets'] },
    ],
    onSuccess: ({ data, toast }) => {
      const toastPayload = resolveSubdomainCreateToast(data)
      if (toastPayload.variant === "warning") {
        toast.warning(toastPayload.key, toastPayload.params)
      } else {
        toast.success(toastPayload.key, toastPayload.params)
      }
    },
    errorFallbackKey: 'toast.asset.subdomain.create.error',
  })
}

// Remove subdomain from organization
export function useDeleteSubdomainFromOrganization() {
  return useResourceMutation({
    mutationFn: (data: { organizationId: number; targetId: number }) =>
      OrganizationService.unlinkTargetsFromOrganization({
        organizationId: data.organizationId,
        targetIds: [data.targetId],
      }),
    loadingToast: {
      key: 'common.status.removing',
      params: {},
      id: ({ organizationId, targetId }) => `delete-${organizationId}-${targetId}`,
    },
    invalidate: [
      { queryKey: subdomainKeys.all },
      { queryKey: ['organizations'] },
    ],
    onSuccess: ({ toast }) => {
      toast.success('toast.asset.subdomain.delete.success')
    },
    errorFallbackKey: 'toast.asset.subdomain.delete.error',
  })
}

// Remove subdomains from your organization in bulk
export function useBatchDeleteSubdomainsFromOrganization() {
  return useResourceMutation({
    mutationFn: (data: { organizationId: number; domainIds: number[] }) => 
      SubdomainService.batchDeleteSubdomainsFromOrganization(data),
    loadingToast: {
      key: 'common.status.batchRemoving',
      params: {},
      id: ({ organizationId }) => `batch-delete-${organizationId}`,
    },
    invalidate: [
      { queryKey: subdomainKeys.all },
      { queryKey: ['organizations'] },
    ],
    onSuccess: ({ data, toast }) => {
      const successCount = getSubdomainBatchDeleteFromOrgCount(data)
      toast.success('toast.asset.subdomain.delete.bulkSuccess', { count: successCount })
    },
    errorFallbackKey: 'toast.asset.subdomain.delete.error',
  })
}

// Delete a single subdomain (using separate DELETE API)
export function useDeleteSubdomain() {
  return useResourceMutation({
    mutationFn: (id: number) => SubdomainService.deleteSubdomain(id),
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-subdomain-${id}`,
    },
    invalidate: subdomainCascadeInvalidates(),
    onSuccess: ({ toast }) => {
      toast.success('toast.asset.subdomain.delete.success')
    },
    errorFallbackKey: 'toast.asset.subdomain.delete.error',
  })
}

// Delete subdomain names in batches (using a unified batch deletion interface)
export function useBatchDeleteSubdomains() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => SubdomainService.batchDeleteSubdomains(ids),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'batch-delete-subdomains',
    },
    invalidate: subdomainCascadeInvalidates(),
    onSuccess: ({ data, toast }) => {
      toast.success('toast.asset.subdomain.delete.bulkSuccess', {
        count: getSubdomainBatchDeleteCount(data),
      })
    },
    errorFallbackKey: 'toast.asset.subdomain.delete.error',
  })
}

// Update subdomain
export function useUpdateSubdomain() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: { name?: string; description?: string } }) =>
      SubdomainService.updateSubdomain({ id, ...data }),
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: ({ id }) => `update-subdomain-${id}`,
    },
    invalidate: [
      { queryKey: subdomainKeys.all },
      { queryKey: ['organizations'] },
    ],
    onSuccess: ({ toast }) => {
      toast.success('common.status.updateSuccess')
    },
    errorFallbackKey: 'common.status.updateFailed',
  })
}

// Get a list of all subdomains
export function useAllSubdomains(
  params: GetAllSubdomainsParams = {},
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ['subdomains', 'all', { page: params.page, pageSize: params.pageSize }],
    queryFn: () => SubdomainService.getAllSubdomains(params),
    select: (response) => ({
      domains: response.domains || [],
      pagination: normalizePagination(response, params.page ?? 1, params.pageSize ?? 10),
    }),
    enabled: options?.enabled !== undefined ? options.enabled : true,
  })
}

// Get a list of target subdomains
export function useTargetSubdomains(
  targetId: number,
  params?: { page?: number; pageSize?: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ['targets', targetId, 'subdomains', { page: params?.page, pageSize: params?.pageSize, filter: params?.filter }],
    queryFn: () => SubdomainService.getSubdomainsByTargetId(targetId, params),
    enabled: options?.enabled !== undefined ? options.enabled : !!targetId,
    placeholderData: keepPreviousData,
  })
}

// Get a list of scanned subdomains
export function useScanSubdomains(
  scanId: number,
  params?: { page?: number; pageSize?: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: ['scans', scanId, 'subdomains', { page: params?.page, pageSize: params?.pageSize, filter: params?.filter }],
    queryFn: () => SubdomainService.getSubdomainsByScanId(scanId, params),
    enabled: options?.enabled !== undefined ? options.enabled : !!scanId,
    placeholderData: keepPreviousData,
  })
}

// Create subdomain names in batches (bind to target)
export function useBulkCreateSubdomains() {
  return useResourceMutation({
    mutationFn: (data: { targetId: number; subdomains: string[] }) =>
      SubdomainService.bulkCreateSubdomains(data.targetId, data.subdomains),
    loadingToast: {
      key: 'common.status.batchCreating',
      params: {},
      id: 'bulk-create-subdomains',
    },
    invalidate: [
      ({ variables }) => ({
        queryKey: ['targets', variables.targetId, 'subdomains'],
        exact: false,
        refetchType: 'active',
      }),
      {
        queryKey: subdomainKeys.all,
        exact: false,
        refetchType: 'active',
      },
    ],
    onSuccess: ({ data, toast }) => {
      const { createdCount, skippedCount = 0, invalidCount = 0, mismatchedCount = 0 } = data
      const totalSkipped = skippedCount + invalidCount + mismatchedCount

      if (totalSkipped > 0) {
        toast.warning('toast.asset.subdomain.create.partialSuccess', {
          success: createdCount,
          skipped: totalSkipped
        })
      } else if (createdCount > 0) {
        toast.success('toast.asset.subdomain.create.success', { count: createdCount })
      } else {
        toast.warning('toast.asset.subdomain.create.partialSuccess', {
          success: 0,
          skipped: 0
        })
      }
    },
    errorFallbackKey: 'toast.asset.subdomain.create.error',
  })
}
