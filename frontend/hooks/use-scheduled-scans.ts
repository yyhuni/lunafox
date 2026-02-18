import { useQuery, keepPreviousData } from '@tanstack/react-query'
import {
  getScheduledScans,
  getScheduledScan,
  createScheduledScan,
  updateScheduledScan,
  deleteScheduledScan,
  toggleScheduledScan,
} from '@/services/scheduled-scan.service'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { handleScheduledScanMutationSuccess } from '@/hooks/_shared/scheduled-scan-mutation-helpers'
import { getErrorCode, getErrorResponseData } from '@/lib/response-parser'
import type {
  CreateScheduledScanRequest,
  UpdateScheduledScanRequest,
  GetScheduledScansResponse,
  ScheduledScan,
} from '@/types/scheduled-scan.types'

// Query Keys
export const scheduledScanKeys = createResourceKeys("scheduled-scans", {
  list: (params: {
    page?: number
    pageSize?: number
    search?: string
    targetId?: number
    organizationId?: number
  }) => params,
  detail: (id: number) => id,
})

/**
 * Get scheduled scan list
 */
export function useScheduledScans(params: {
  page?: number
  pageSize?: number
  search?: string
  targetId?: number
  organizationId?: number
} = { page: 1, pageSize: 10 }) {
  return useQuery({
    queryKey: scheduledScanKeys.list(params),
    queryFn: () => getScheduledScans(params),
    placeholderData: keepPreviousData,
  })
}

/**
 * Get scheduled scan details
 */
export function useScheduledScan(id: number) {
  return useQuery({
    queryKey: scheduledScanKeys.detail(id),
    queryFn: () => getScheduledScan(id),
    enabled: !!id,
  })
}

/**
 * Create a scheduled scan
 */
export function useCreateScheduledScan() {
  return useResourceMutation({
    mutationFn: (data: CreateScheduledScanRequest) => createScheduledScan(data),
    invalidate: [{ queryKey: scheduledScanKeys.all }],
    onSuccess: ({ data: response, toast }) => {
      handleScheduledScanMutationSuccess({
        response,
        onSuccess: () => {
          // Show success prompt using i18n message
          toast.success('toast.scheduledScan.create.success')
        },
      })
    },
    errorFallbackKey: 'toast.scheduledScan.create.error',
  })
}

/**
 * Update scheduled scan
 */
export function useUpdateScheduledScan() {
  return useResourceMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateScheduledScanRequest }) =>
      updateScheduledScan(id, data),
    invalidate: [
      { queryKey: scheduledScanKeys.all },
      { queryKey: scheduledScanKeys.details() },
    ],
    onSuccess: ({ data: response, toast }) => {
      handleScheduledScanMutationSuccess({
        response,
        onSuccess: () => {
          // Show success prompt using i18n message
          toast.success('toast.scheduledScan.update.success')
        },
      })
    },
    errorFallbackKey: 'toast.scheduledScan.update.error',
  })
}

/**
 * Delete scheduled scan
 */
export function useDeleteScheduledScan() {
  return useResourceMutation({
    mutationFn: (id: number) => deleteScheduledScan(id),
    invalidate: [{ queryKey: scheduledScanKeys.all }],
    onSuccess: ({ data: response, toast }) => {
      handleScheduledScanMutationSuccess({
        response,
        onSuccess: () => {
          // Show success prompt using i18n message
          toast.success('toast.scheduledScan.delete.success')
        },
      })
    },
    errorFallbackKey: 'toast.scheduledScan.delete.error',
  })
}

/**
 * Switch scheduled scan enable status
 * Use optimistic updates to avoid re-fetching data causing the list to be reordered
 */
export function useToggleScheduledScan() {
  return useResourceMutation({
    mutationFn: ({ id, isEnabled }: { id: number; isEnabled: boolean }) =>
      toggleScheduledScan(id, isEnabled),
    onMutate: async ({ id, isEnabled }, context) => {
      const { queryClient } = context
      // Cancel an ongoing query
      await queryClient.cancelQueries({ queryKey: scheduledScanKeys.all })

      // Get all currently cached scheduled-scans queries
      const previousQueries = queryClient.getQueriesData({ queryKey: scheduledScanKeys.all })

      // Optimistically updates all matching query caches
      queryClient.setQueriesData(
        { queryKey: scheduledScanKeys.all },
        (old: GetScheduledScansResponse | undefined) => {
          if (!old?.results) return old
          return {
            ...old,
            results: old.results.map((item: ScheduledScan) =>
              item.id === id ? { ...item, isEnabled } : item
            ),
          }
        }
      )

      // Return context for rollback
      return { previousQueries }
    },
    onSuccess: ({ data: response, variables: { isEnabled }, toast }) => {
      handleScheduledScanMutationSuccess({
        response,
        onSuccess: () => {
          // Show success prompt using i18n message
          if (isEnabled) {
            toast.success('toast.scheduledScan.toggle.enabled')
          } else {
            toast.success('toast.scheduledScan.toggle.disabled')
          }
        },
      })
      // Do not call invalidateQueries, keep the current sorting
    },
    onError: ({ error, context, toast, queryClient }) => {
      // Roll back to previous state
      if (context?.previousQueries) {
        context.previousQueries.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data)
        })
      }
      const errorCode = getErrorCode(getErrorResponseData(error))
      if (errorCode) {
        toast.errorFromCode(errorCode)
      } else {
        toast.error('toast.scheduledScan.toggle.error')
      }
    },
  })
}
