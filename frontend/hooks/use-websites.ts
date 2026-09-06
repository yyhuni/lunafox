import { useCallback } from "react"
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import {
  getAssetDeletedCount,
  resolveAssetBulkCreateToast,
} from '@/hooks/_shared/asset-mutation-helpers'
import { WebsiteService } from '@/services/website.service'
import type { WebsiteFilterOptionField, WebsiteListQueryParams } from '@/types/website.types'

// Query Keys
const websiteKeyBase = createResourceKeys("websites")

export const websiteKeys = {
  ...websiteKeyBase,
  target: (targetId: number, params: WebsiteListQueryParams) =>
    [...websiteKeyBase.all, 'target', targetId, params] as const,
  scan: (scanId: number, params: WebsiteListQueryParams) =>
    [...websiteKeyBase.all, 'scan', scanId, params] as const,
  detail: (websiteId: number) =>
    [...websiteKeyBase.all, "detail", websiteId] as const,
  filterOptions: (scope: "target" | "scan", id: number, field: WebsiteFilterOptionField) =>
    [...websiteKeyBase.all, "filterOptions", scope, id, field] as const,
}

function websiteCascadeInvalidates() {
  return [
    { queryKey: websiteKeys.all },
    { queryKey: ['targets'] as const },
    { queryKey: ['scans'] as const },
  ]
}

// Get the target website list
export function useTargetWebSites(
  targetId: number,
  params: WebsiteListQueryParams,
  options?: { enabled?: boolean }
) {
  const resolved = {
    pageSize: params.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }
  return useQuery({
    queryKey: websiteKeys.target(targetId, resolved),
    queryFn: () => WebsiteService.getTargetWebSites(targetId, resolved),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

export function useWebsite(websiteId: number, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: websiteKeys.detail(websiteId),
    queryFn: () => WebsiteService.getWebsite(websiteId),
    enabled: options?.enabled ?? Boolean(websiteId),
  })
}

// Get a list of scanned websites
export function useScanWebSites(
  scanId: number,
  params: WebsiteListQueryParams,
  options?: { enabled?: boolean }
) {
  const resolved = {
    pageSize: params.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }
  return useQuery({
    queryKey: websiteKeys.scan(scanId, resolved),
    queryFn: () => WebsiteService.getScanWebSites(scanId, resolved),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

export function useTargetWebsiteFilterOptions(
  targetId: number,
  field: WebsiteFilterOptionField,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: websiteKeys.filterOptions("target", targetId, field),
    queryFn: () => WebsiteService.getTargetWebsiteFilterOptions(targetId, field),
    enabled: options?.enabled ?? !!targetId,
    placeholderData: keepPreviousData,
  })
}

export function useScanWebsiteFilterOptions(
  scanId: number,
  field: WebsiteFilterOptionField,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: websiteKeys.filterOptions("scan", scanId, field),
    queryFn: () => WebsiteService.getScanWebsiteFilterOptions(scanId, field),
    enabled: options?.enabled ?? !!scanId,
    placeholderData: keepPreviousData,
  })
}

// Delete a single website (using separate DELETE API)
export function useDeleteWebSite() {
  return useResourceMutation({
    mutationFn: WebsiteService.deleteWebSite,
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-website-${id}`,
    },
    invalidate: websiteCascadeInvalidates(),
    onSuccess: ({ toast }) => {
      toast.success('toast.asset.website.delete.success')
    },
    errorFallbackKey: 'toast.asset.website.delete.error',
  })
}

// Delete websites in batches (using a unified batch deletion interface)
export function useBulkDeleteWebSites() {
  return useResourceMutation({
    mutationFn: (data: { targetId: number; ids: number[] }) =>
      WebsiteService.bulkDeleteWebSites(data.targetId, data.ids),
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'bulk-delete-websites',
    },
    invalidate: websiteCascadeInvalidates(),
    onSuccess: ({ data, toast }) => {
      toast.success('toast.asset.website.delete.bulkSuccess', {
        count: getAssetDeletedCount(data),
      })
    },
    errorFallbackKey: 'toast.asset.website.delete.error',
  })
}


// Create websites in batches (bind to targets)
export function useBulkCreateWebsites() {
  return useResourceMutation({
    mutationFn: (data: { targetId: number; urls: string[] }) =>
      WebsiteService.bulkCreateWebsites(data.targetId, data.urls),
    loadingToast: {
      key: 'common.status.batchCreating',
      params: {},
      id: 'bulk-create-websites',
    },
    invalidate: [
      {
        queryKey: websiteKeys.all,
        exact: false,
        refetchType: 'active',
      },
      ({ variables }) => ({
        queryKey: ['targets', variables.targetId],
        refetchType: 'active',
      }),
    ],
    onSuccess: ({ data, toast }) => {
      const toastPayload = resolveAssetBulkCreateToast(data.createdCount, {
        success: 'toast.asset.website.create.success',
        partial: 'toast.asset.website.create.partialSuccess',
      })
      if (toastPayload.variant === 'success') {
        toast.success(toastPayload.key, toastPayload.params)
      } else {
        toast.warning(toastPayload.key, toastPayload.params)
      }
    },
    errorFallbackKey: 'toast.asset.website.create.error',
  })
}

export function useExportWebsites({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  return useCallback(() => {
    if (scanId) {
      return WebsiteService.exportWebsitesByScanId(scanId)
    }
    if (targetId) {
      return WebsiteService.exportWebsitesByTargetId(targetId)
    }
    return Promise.resolve(null)
  }, [scanId, targetId])
}
