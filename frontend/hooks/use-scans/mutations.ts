import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import {
  handleScanMutationSuccess,
  getBulkInitiateScanSuccessCount,
  getBulkDeleteSuccessCount,
  getQuickScanSuccessCount,
  getStopScanSuccessCount,
  type BatchStopScansResponse,
  type BulkDeleteScansResponse,
  type BulkInitiateScanResponse,
  type InitiateScanResponse,
  type QuickScanResponse,
  type ScanMutationInvalidationKeys,
  type StopScanResponse,
} from "@/hooks/_shared/scan-mutation-helpers"
import { parseResponse } from "@/lib/response-parser"
import {
  quickScan,
  initiateScan,
  bulkInitiateScan,
  deleteScan,
  bulkDeleteScans,
  stopScan,
  batchStopScans,
} from "@/services/scan.service"
import type {
  QuickScanRequest,
  InitiateScanRequest,
  BulkInitiateScanRequest,
} from "@/types/scan.types"
import { scanKeys } from "./keys"

const scanInvalidationKeys: ScanMutationInvalidationKeys = {
  all: scanKeys.all,
  statistics: scanKeys.statistics(),
}

/**
 * Quickly scan mutation hooks
 */
export function useQuickScan() {
  return useResourceMutation({
    mutationFn: (data: QuickScanRequest) => quickScan(data),
    loadingToast: {
      key: "common.status.creating",
      params: {},
      id: "quick-scan",
    },
    onSuccess: async ({ data: response, queryClient, toast }) => {
      await handleScanMutationSuccess<QuickScanResponse>({
        response,
        parse: (payload) => parseResponse<QuickScanResponse>(payload),
        onValidData: (data) => {
          const count = getQuickScanSuccessCount(data)
          toast.success("toast.scan.quick.success", { count })
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.scan.quick.error",
  })
}

/**
 * Initiate scanning mutation hook
 */
export function useInitiateScan() {
  return useResourceMutation({
    mutationFn: (data: InitiateScanRequest) => initiateScan(data),
    loadingToast: {
      key: "common.status.creating",
      params: {},
      id: ({ targetId }) => `initiate-scan-${targetId}`,
    },
    onSuccess: async ({ data: response, queryClient, toast }) => {
      await handleScanMutationSuccess<InitiateScanResponse>({
        response,
        parse: (payload) => parseResponse<InitiateScanResponse>(payload),
        onValidData: () => {
          toast.success("toast.scan.initiate.success")
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.scan.initiate.error",
  })
}

/**
 * Bulk initiate scanning mutation hook
 */
export function useBulkInitiateScan() {
  return useResourceMutation({
    mutationFn: (data: BulkInitiateScanRequest) => bulkInitiateScan(data),
    loadingToast: {
      key: "common.status.batchCreating",
      params: {},
      id: "bulk-initiate-scan",
    },
    onSuccess: async ({ data: response, variables, queryClient, toast }) => {
      await handleScanMutationSuccess<BulkInitiateScanResponse>({
        response,
        parse: (payload) => parseResponse<BulkInitiateScanResponse>(payload),
        onValidData: (data) => {
          const fallbackCount = (variables.targetIds?.length ?? 0) + (variables.organizationIds?.length ?? 0)
          const count = getBulkInitiateScanSuccessCount(data, fallbackCount)
          toast.success("toast.scan.bulkInitiate.success", { count })
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.scan.bulkInitiate.error",
  })
}

/**
 * Delete scan mutation hook
 */
export function useDeleteScan() {
  return useResourceMutation({
    mutationFn: (id: number) => deleteScan(id),
    loadingToast: {
      key: "common.status.deleting",
      params: {},
      id: (id) => `delete-scan-${id}`,
    },
    onSuccess: async ({ data: response, variables: id, queryClient, toast }) => {
      await handleScanMutationSuccess<unknown>({
        response,
        parse: (payload) => parseResponse<unknown>(payload),
        onValidData: () => {
          toast.success("toast.scan.delete.success", {
            name: `Scan #${id}`,
          })
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.deleteFailed",
  })
}

/**
 * Batch delete scanning mutation hook
 */
export function useBulkDeleteScans() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => bulkDeleteScans(ids),
    loadingToast: {
      key: "common.status.batchDeleting",
      params: {},
      id: "bulk-delete-scans",
    },
    onSuccess: async ({ data: response, variables: ids, queryClient, toast }) => {
      await handleScanMutationSuccess<BulkDeleteScansResponse>({
        response,
        parse: (payload) => parseResponse<BulkDeleteScansResponse>(payload),
        onValidData: (data) => {
          const count = getBulkDeleteSuccessCount(data, ids.length)
          toast.success("toast.scan.delete.bulkSuccess", { count })
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.bulkDeleteFailed",
  })
}

/**
 * Stop scanning for mutation hooks
 */
export function useStopScan() {
  return useResourceMutation({
    mutationFn: (id: number) => stopScan(id),
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: (id) => `stop-scan-${id}`,
    },
    onSuccess: async ({ data: response, queryClient, toast }) => {
      await handleScanMutationSuccess<StopScanResponse>({
        response,
        parse: (payload) => parseResponse<StopScanResponse>(payload),
        onValidData: (data) => {
          const count = getStopScanSuccessCount(data)
          toast.success("toast.scan.stop.success", { count })
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.stopFailed",
  })
}

/**
 * Stop all selected active scans in one request-bound transaction.
 */
export function useBatchStopScans() {
  return useResourceMutation({
    mutationFn: (ids: number[]) => batchStopScans(ids),
    loadingToast: {
      key: "common.status.updating",
      params: {},
      id: "batch-stop-scans",
    },
    onSuccess: async ({ data: response, queryClient, toast }) => {
      await handleScanMutationSuccess<BatchStopScansResponse>({
        response,
        parse: (payload) => parseBatchStopScansResponse(payload),
        onValidData: (data) => {
          toast.success("toast.scan.stop.batchSuccess", {
            stoppedCount: data.stoppedCount,
            skippedCount: data.skippedCount,
          })
        },
        queryClient,
        invalidateKeys: scanInvalidationKeys,
      })
    },
    errorFallbackKey: "toast.scan.stop.batchError",
  })
}

function parseBatchStopScansResponse(payload: unknown): BatchStopScansResponse | null {
  if (typeof payload !== "object" || payload === null || Array.isArray(payload)) {
    return null
  }
  const value = payload as Record<string, unknown>
  const counts = [value.stoppedCount, value.skippedCount, value.revokedTaskCount]
  if (!counts.every((count) => typeof count === "number" && Number.isSafeInteger(count) && count >= 0)) {
    return null
  }
  return {
    stoppedCount: value.stoppedCount as number,
    skippedCount: value.skippedCount as number,
    revokedTaskCount: value.revokedTaskCount as number,
  }
}
