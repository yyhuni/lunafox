import { useCallback } from "react"
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import { ScreenshotService } from '@/services/screenshot.service'
import type { ScreenshotFilterOptionField, ScreenshotListQueryParams } from "@/types/screenshot.types"

// Query Keys
const screenshotKeyBase = createResourceKeys("screenshots")

export const screenshotKeys = {
  ...screenshotKeyBase,
  target: (targetId: number, params: ScreenshotListQueryParams) =>
    [...screenshotKeyBase.all, 'target', targetId, params] as const,
  scan: (scanId: number, params: ScreenshotListQueryParams) =>
    [...screenshotKeyBase.all, 'scan', scanId, params] as const,
  filterOptions: (scope: "target" | "scan", id: number, field: ScreenshotFilterOptionField) =>
    [...screenshotKeyBase.all, "filterOptions", scope, id, field] as const,
}

// Get a list of screenshots of the target
export function useTargetScreenshots(
  targetId: number,
  params: ScreenshotListQueryParams,
  options?: { enabled?: boolean }
) {
  const resolved = {
    pageSize: params.pageSize ?? 12,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }
  return useQuery({
    queryKey: screenshotKeys.target(targetId, resolved),
    queryFn: () => ScreenshotService.getByTarget(targetId, resolved),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

// Get a list of scanned screenshots
export function useScanScreenshots(
  scanId: number,
  params: ScreenshotListQueryParams,
  options?: { enabled?: boolean }
) {
  const resolved = {
    pageSize: params.pageSize ?? 12,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }
  return useQuery({
    queryKey: screenshotKeys.scan(scanId, resolved),
    queryFn: () => ScreenshotService.getByScan(scanId, resolved),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

export function useTargetScreenshotFilterOptions(
  targetId: number,
  field: ScreenshotFilterOptionField,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: screenshotKeys.filterOptions("target", targetId, field),
    queryFn: () => ScreenshotService.getTargetFilterOptions(targetId, field),
    enabled: options?.enabled ?? !!targetId,
    placeholderData: keepPreviousData,
  })
}

export function useScanScreenshotFilterOptions(
  scanId: number,
  field: ScreenshotFilterOptionField,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: screenshotKeys.filterOptions("scan", scanId, field),
    queryFn: () => ScreenshotService.getScanFilterOptions(scanId, field),
    enabled: options?.enabled ?? !!scanId,
    placeholderData: keepPreviousData,
  })
}

export function useBulkDeleteScreenshots() {
  return useResourceMutation({
    mutationFn: (data: { targetId: number; ids: number[] }) =>
      ScreenshotService.bulkDelete(data.targetId, data.ids),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'bulk-delete-screenshots',
    },
    invalidate: [{ queryKey: screenshotKeys.all, exact: false, refetchType: 'active' }],
    onSuccess: ({ data, toast }) => {
      toast.success('toast.asset.screenshot.delete.bulkSuccess', {
        count: data.deletedCount,
      })
    },
    errorFallbackKey: 'toast.asset.screenshot.delete.error',
  })
}

export function useScreenshotImageUrlResolver(scanId?: number) {
  return useCallback((screenshotId: number) => {
    if (scanId) {
      return ScreenshotService.getSnapshotImageUrl(scanId, screenshotId)
    }
    return ScreenshotService.getImageUrl(screenshotId)
  }, [scanId])
}
