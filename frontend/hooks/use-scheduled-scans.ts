import { useQuery, keepPreviousData } from '@tanstack/react-query'
import {
  getScheduledScans,
  getScheduledScan,
  createScheduledScan,
  updateScheduledScan,
  deleteScheduledScan,
  batchDeleteScheduledScans,
	batchUpdateScheduledScanStatus,
  toggleScheduledScan,
	getScheduledScanOverviewSummary,
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
	ScheduledScanOverviewSummary,
	BatchUpdateScheduledScanStatusInput,
} from '@/types/scheduled-scan.types'

// Query Keys
export const scheduledScanKeys = createResourceKeys("scheduled-scans", {
  list: (params: {
    page?: number
    pageSize?: number
    pageToken?: string
    search?: string
    targetId?: number
    organizationId?: number
  }) => params,
  detail: (id: number) => id,
})

export const scheduledScanOverviewKey = [...scheduledScanKeys.all, "overview"] as const

/**
 * Get scheduled scan list
 */
export function useScheduledScans(params: {
  page?: number
  pageSize?: number
  pageToken?: string
  search?: string
  targetId?: number
  organizationId?: number
} = { pageSize: 10 }) {
  return useQuery({
    queryKey: scheduledScanKeys.list(params),
    queryFn: () => getScheduledScans(params),
    placeholderData: keepPreviousData,
  })
}

export function useScheduledScanOverviewSummary() {
	const query = useQuery<ScheduledScanOverviewSummary>({
		queryKey: scheduledScanOverviewKey,
			queryFn: () => getScheduledScanOverviewSummary(),
	})
	const hasSuccessfulData = query.data !== undefined && query.dataUpdatedAt > 0
	return {
		...query,
		isInitialError: query.isError && !hasSuccessfulData,
		isRefetchStale: query.isError && hasSuccessfulData,
		lastSuccessfulAt: hasSuccessfulData ? query.dataUpdatedAt : null,
	}
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
    loadingToast: {
      key: 'common.status.creating',
      params: {},
      id: 'create-scheduled-scan',
    },
    invalidate: [{ queryKey: scheduledScanKeys.all }, { queryKey: scheduledScanOverviewKey }],
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
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: ({ id }) => `update-scheduled-scan-${id}`,
    },
    invalidate: [
      { queryKey: scheduledScanKeys.all },
      { queryKey: scheduledScanKeys.details() },
      { queryKey: scheduledScanOverviewKey },
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
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-scheduled-scan-${id}`,
    },
    invalidate: [{ queryKey: scheduledScanKeys.all }, { queryKey: scheduledScanOverviewKey }],
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
 * Delete scheduled scans in batches
 */
export function useBatchDeleteScheduledScans() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => batchDeleteScheduledScans(ids),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'batch-delete-scheduled-scans',
    },
    invalidate: [{ queryKey: scheduledScanKeys.all }, { queryKey: scheduledScanOverviewKey }],
    onSuccess: ({ data: response, toast }) => {
      handleScheduledScanMutationSuccess({
        response,
        onSuccess: () => {
          toast.success('toast.scheduledScan.delete.bulkSuccess', {
            count: response.deletedCount,
          })
        },
      })
    },
    errorFallbackKey: 'toast.scheduledScan.delete.error',
  })
}

/**
 * Assign one explicit enabled state to the selected Scheduled Scans.
 */
export function useBatchUpdateScheduledScanStatus() {
  return useResourceMutation({
    mutationFn: (input: BatchUpdateScheduledScanStatusInput) =>
      batchUpdateScheduledScanStatus(input),
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: 'batch-update-scheduled-scan-status',
    },
    invalidate: [{ queryKey: scheduledScanKeys.all }, { queryKey: scheduledScanOverviewKey }],
    onSuccess: ({ data: response, variables, toast }) => {
      handleScheduledScanMutationSuccess({
        response,
        onSuccess: () => {
          toast.success(
            variables.isEnabled
              ? 'toast.scheduledScan.batchStatus.enabled'
              : 'toast.scheduledScan.batchStatus.disabled',
            { count: response.updatedCount }
          )
        },
      })
    },
    errorFallbackKey: 'toast.scheduledScan.batchStatus.error',
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
    loadingToast: {
      key: 'common.status.updating',
      params: {},
      id: ({ id }) => `toggle-scheduled-scan-${id}`,
    },
		invalidate: [{ queryKey: scheduledScanOverviewKey }],
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
          if (!old?.scheduledScans) return old
          return {
            ...old,
            scheduledScans: old.scheduledScans.map((item: ScheduledScan) =>
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
