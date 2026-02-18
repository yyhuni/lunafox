import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import {
  getAssetDeletedCount,
  resolveAssetBulkCreateToast,
} from '@/hooks/_shared/asset-mutation-helpers'
import { WebsiteService } from '@/services/website.service'

// Query Keys
const websiteKeyBase = createResourceKeys("websites")

export const websiteKeys = {
  ...websiteKeyBase,
  target: (targetId: number, params: { page: number; pageSize: number; filter?: string }) =>
    [...websiteKeyBase.all, 'target', targetId, params] as const,
  scan: (scanId: number, params: { page: number; pageSize: number; filter?: string }) =>
    [...websiteKeyBase.all, 'scan', scanId, params] as const,
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
  params: { page: number; pageSize: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: websiteKeys.target(targetId, params),
    queryFn: () => WebsiteService.getTargetWebSites(targetId, params),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

// Get a list of scanned websites
export function useScanWebSites(
  scanId: number,
  params: { page: number; pageSize: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: websiteKeys.scan(scanId, params),
    queryFn: () => WebsiteService.getScanWebSites(scanId, params),
    enabled: options?.enabled ?? true,
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
    mutationFn: WebsiteService.bulkDeleteWebSites,
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
