import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { getAssetDeletedCount } from '@/hooks/_shared/asset-mutation-helpers'
import {
  applyOrganizationOptimisticDelete,
  getOrganizationDeleteToastId,
  invalidateOrganizationTargets,
  ORGANIZATION_BATCH_DELETE_TOAST_ID,
  rollbackOrganizationQueries,
} from '@/hooks/_shared/organization-mutation-helpers'
import { getErrorCode, getErrorResponseData } from '@/lib/response-parser'
import { OrganizationService } from '@/services/organization.service'
import type { CreateOrganizationRequest, UpdateOrganizationRequest } from '@/types/organization.types'

type OrganizationListParams = { page?: number; pageSize?: number; filter?: string }
// Query Keys - Unified query key management
export const organizationKeys = createResourceKeys("organizations", {
  list: (params?: OrganizationListParams) => params,
  detail: (id: number) => id,
})

/**
 * Hook for getting organization list
 * 
 * Features:
 * - Automatic loading state management
 * - Automatic error handling
 * - Pagination support
 * - Automatic caching and revalidation
 * - Conditional query support (enabled option)
 */
// Backend is fixed to sort by update time in descending order, does not support custom sorting
export function useOrganizations(
  params: {
    page?: number
    pageSize?: number
    filter?: string
  } = {},
  options?: {
    enabled?: boolean
  }
) {
  return useQuery({
    queryKey: organizationKeys.list({
      page: params.page || 1,
      pageSize: params.pageSize || 10,
      filter: params.filter || undefined,
    }),
    queryFn: () => OrganizationService.getOrganizations(params || {}),
    select: (response) => {
      // Handle DRF pagination response format
      const page = params.page || 1
      const pageSize = params.pageSize || 10
      const total = response.total || response.count || 0
      const totalPages = Math.ceil(total / pageSize)
      
      return {
        organizations: response.results || [],
        pagination: {
          total,
          page,
          pageSize,
          totalPages,
        }
      }
    },
    enabled: options?.enabled !== undefined ? options.enabled : true,
    placeholderData: keepPreviousData,
  })
}

/**
 * Get single organization details Hook
 */
export function useOrganization(id: number) {
  return useQuery({
    queryKey: organizationKeys.detail(id),
    queryFn: () => OrganizationService.getOrganizationById(id),
    enabled: !!id, // Only execute query when id exists
  })
}

/**
 * Get organization's target list Hook
 */
export function useOrganizationTargets(
  id: number,
  params?: {
    page?: number
    pageSize?: number
    sortBy?: string
    sortOrder?: 'asc' | 'desc'
    search?: string
    type?: string
  },
  options?: {
    enabled?: boolean
  }
) {
  return useQuery({
    queryKey: [...organizationKeys.detail(id), 'targets', params],
    queryFn: () => OrganizationService.getOrganizationTargets(id, params),
    enabled: options?.enabled !== undefined ? (options.enabled && !!id) : !!id,
    placeholderData: keepPreviousData,
  })
}

/**
 * Create organization Mutation Hook
 * 
 * Features:
 * - Automatic submission state management
 * - Automatic list refresh after success
 * - Automatic success/failure notifications
 */
export function useCreateOrganization() {
  return useResourceMutation({
    mutationFn: (data: CreateOrganizationRequest) => 
      OrganizationService.createOrganization(data),
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'create-organization',
    },
    invalidate: [{ queryKey: organizationKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.organization.create.success')
    },
    errorFallbackKey: 'toast.organization.create.error',
  })
}

/**
 * Update organization Mutation Hook
 */
export function useUpdateOrganization() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateOrganizationRequest }) =>
      OrganizationService.updateOrganization({ id, ...data }),
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: ({ id }) => `update-${id}`,
    },
    invalidate: [{ queryKey: organizationKeys.all }],
    onSuccess: ({ toast }) => {
      toast.success('toast.organization.update.success')
    },
    errorFallbackKey: 'toast.organization.update.error',
  })
}

/**
 * Deleting an organization's Mutation Hook (optimistic update)
 */
export function useDeleteOrganization() {
  return useResourceMutation({
    mutationFn: (id: number) => OrganizationService.deleteOrganization(id),
    onMutate: async (deletedId, { queryClient, toast }) => {
      const toastId = getOrganizationDeleteToastId(deletedId)
      toast.loading('common.status.deleting', {}, toastId)

      await queryClient.cancelQueries({ queryKey: organizationKeys.all })
      const previousData = applyOrganizationOptimisticDelete(queryClient, [deletedId])

      return { previousData, deletedId }
    },
    onSuccess: ({ data: response, context, toast }) => {
      if (context?.deletedId) {
        toast.dismiss(getOrganizationDeleteToastId(context.deletedId))
      }
      const { organizationName } = response
      toast.success('toast.organization.delete.success', { name: organizationName })
    },
    onError: ({ error, context, toast, queryClient }) => {
      if (context?.deletedId) {
        toast.dismiss(`delete-${context.deletedId}`)
      }

      rollbackOrganizationQueries(queryClient, context?.previousData)

      toast.errorFromCode(getErrorCode(getErrorResponseData(error)), 'toast.organization.delete.error')
    },
    onSettled: async ({ queryClient }) => {
      await invalidateOrganizationTargets(queryClient)
    },
  })
}

/**
 * Mutation Hook for bulk deletion of organizations (optimistic update)
 */
export function useBatchDeleteOrganizations() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => 
      OrganizationService.batchDeleteOrganizations(ids),
    onMutate: async (deletedIds, { queryClient, toast }) => {
      toast.loading('common.status.batchDeleting', {}, ORGANIZATION_BATCH_DELETE_TOAST_ID)

      await queryClient.cancelQueries({ queryKey: organizationKeys.all })
      const previousData = applyOrganizationOptimisticDelete(queryClient, deletedIds)

      return { previousData, deletedIds }
    },
    onSuccess: ({ data: response, toast }) => {
      toast.dismiss(ORGANIZATION_BATCH_DELETE_TOAST_ID)
      toast.success('toast.organization.delete.bulkSuccess', {
        count: getAssetDeletedCount(response),
      })
    },
    onError: ({ error, context, toast, queryClient }) => {
      toast.dismiss(ORGANIZATION_BATCH_DELETE_TOAST_ID)

      rollbackOrganizationQueries(queryClient, context?.previousData)

      toast.errorFromCode(getErrorCode(getErrorResponseData(error)), 'toast.organization.delete.error')
    },
    onSettled: async ({ queryClient }) => {
      await invalidateOrganizationTargets(queryClient)
    },
  })
}



/**
 * Unorganize a Mutation Hook associated with a target (batch)
 */
export function useUnlinkTargetsFromOrganization() {
  return useResourceMutation({
    mutationFn: (data: { organizationId: number; targetIds: number[] }) => 
      OrganizationService.unlinkTargetsFromOrganization(data),
    loadingToast: {
      key: 'common.status.unlinking',
      params: {},
      id: ({ organizationId }) => `unlink-${organizationId}`,
    },
    invalidate: [
      { queryKey: ['targets'] },
      { queryKey: organizationKeys.all },
    ],
    onSuccess: ({ variables: { targetIds }, toast }) => {
      toast.success('toast.target.unlink.bulkSuccess', { count: targetIds.length })
    },
    errorFallbackKey: 'toast.target.unlink.error',
  })
}
