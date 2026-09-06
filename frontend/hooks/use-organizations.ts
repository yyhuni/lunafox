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
import type { CreateOrganizationRequest, Organization, OrganizationsResponse, UpdateOrganizationRequest } from '@/types/organization.types'

type OrganizationListParams = {
  pageSize?: number
  pageToken?: string
  filter?: string
  orderBy?: string
  pageIndex?: number
}
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
export function useOrganizations(
  params: {
    pageSize?: number
    pageToken?: string
    filter?: string
    orderBy?: string
    pageIndex?: number
  } = {},
  options?: {
    enabled?: boolean
  }
) {
  return useQuery({
    queryKey: organizationKeys.list({
      pageSize: params.pageSize || 10,
      pageToken: params.pageToken,
      filter: params.filter || undefined,
      orderBy: params.orderBy || undefined,
    }),
    queryFn: () => OrganizationService.getOrganizations({
      pageSize: params.pageSize,
      pageToken: params.pageToken,
      filter: params.filter,
      orderBy: params.orderBy,
    }),
    select: (response) => {
      const total = response.totalSize ?? response.total ?? 0
      const page = response.page ?? params.pageIndex ?? 1
      const pageSize = response.pageSize ?? params.pageSize ?? 10
      const totalPages = response.totalPages ?? Math.ceil(total / pageSize)

      return {
        ...response,
        organizations: response.results ?? [],
        total,
        totalSize: total,
        page,
        pageSize,
        totalPages,
        nextPageToken: response.nextPageToken,
        pagination: {
          total,
          page,
          pageSize,
          totalPages,
        },
      } satisfies OrganizationsResponse<Organization> & {
        organizations: Organization[]
        pagination: { total: number; page: number; pageSize: number; totalPages: number }
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
    pageSize?: number
    pageToken?: string
    sortBy?: string
    sortOrder?: 'asc' | 'desc'
    search?: string
    filter?: string
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
      id: ({ id }) => `update-organization-${id}`,
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
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: getOrganizationDeleteToastId,
    },
    onMutate: async (deletedId, { queryClient }) => {
      await queryClient.cancelQueries({ queryKey: organizationKeys.all })
      const previousData = applyOrganizationOptimisticDelete(queryClient, [deletedId])

      return { previousData, deletedId }
    },
    onSuccess: async ({ data: response, toast, queryClient }) => {
      await invalidateOrganizationTargets(queryClient)
      const { organizationName } = response
      toast.success('toast.organization.delete.success', { name: organizationName })
    },
    onError: async ({ error, context, toast, queryClient }) => {
      rollbackOrganizationQueries(queryClient, context?.previousData)
      await invalidateOrganizationTargets(queryClient)
      toast.errorFromCode(getErrorCode(getErrorResponseData(error)), 'toast.organization.delete.error')
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
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: ORGANIZATION_BATCH_DELETE_TOAST_ID,
    },
    onMutate: async (deletedIds, { queryClient }) => {
      await queryClient.cancelQueries({ queryKey: organizationKeys.all })
      const previousData = applyOrganizationOptimisticDelete(queryClient, deletedIds)

      return { previousData, deletedIds }
    },
    onSuccess: async ({ data: response, toast, queryClient }) => {
      await invalidateOrganizationTargets(queryClient)
      toast.success('toast.organization.delete.bulkSuccess', {
        count: getAssetDeletedCount(response),
      })
    },
    onError: async ({ error, context, toast, queryClient }) => {
      rollbackOrganizationQueries(queryClient, context?.previousData)
      await invalidateOrganizationTargets(queryClient)
      toast.errorFromCode(getErrorCode(getErrorResponseData(error)), 'toast.organization.delete.error')
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
