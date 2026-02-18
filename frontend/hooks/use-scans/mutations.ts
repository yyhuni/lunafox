import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import {
  handleScanMutationSuccess,
  getBulkDeleteSuccessCount,
  getQuickScanSuccessCount,
  getStopScanSuccessCount,
  type BulkDeleteScansResponse,
  type InitiateScanResponse,
  type QuickScanResponse,
  type ScanMutationInvalidationKeys,
  type StopScanResponse,
} from "@/hooks/_shared/scan-mutation-helpers"
import { parseResponse } from "@/lib/response-parser"
import {
  quickScan,
  initiateScan,
  deleteScan,
  bulkDeleteScans,
  stopScan,
} from "@/services/scan.service"
import type {
  QuickScanRequest,
  InitiateScanRequest,
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
 * Delete scan mutation hook
 */
export function useDeleteScan() {
  return useResourceMutation({
    mutationFn: (id: number) => deleteScan(id),
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
