import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import {
  getAssetDeletedCount,
  resolveAssetBulkCreateToast,
} from '@/hooks/_shared/asset-mutation-helpers'
import { DirectoryService } from '@/services/directory.service'

// Query Keys
const directoryKeyBase = createResourceKeys("directories")

export const directoryKeys = {
  ...directoryKeyBase,
  target: (targetId: number, params: { page: number; pageSize: number; filter?: string }) =>
    [...directoryKeyBase.all, 'target', targetId, params] as const,
  scan: (scanId: number, params: { page: number; pageSize: number; filter?: string }) =>
    [...directoryKeyBase.all, 'scan', scanId, params] as const,
}

function directoryCascadeInvalidates() {
  return [
    { queryKey: directoryKeys.all },
    { queryKey: ['targets'] as const },
    { queryKey: ['scans'] as const },
  ]
}

// Get the directory listing of the target
export function useTargetDirectories(
  targetId: number,
  params: { page: number; pageSize: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: directoryKeys.target(targetId, params),
    queryFn: () => DirectoryService.getTargetDirectories(targetId, params),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

// Get a list of scanned directories
export function useScanDirectories(
  scanId: number,
  params: { page: number; pageSize: number; filter?: string },
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: directoryKeys.scan(scanId, params),
    queryFn: () => DirectoryService.getScanDirectories(scanId, params),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

// Delete a single directory (using separate DELETE API)
export function useDeleteDirectory() {
  return useResourceMutation({
    mutationFn: DirectoryService.deleteDirectory,
    loadingToast: {
      key: 'common.status.deleting',
      params: {},
      id: (id) => `delete-directory-${id}`,
    },
    invalidate: directoryCascadeInvalidates(),
    onSuccess: ({ toast }) => {
      toast.success('toast.asset.directory.delete.success')
    },
    errorFallbackKey: 'toast.asset.directory.delete.error',
  })
}

// Deleting directories in batches (using a unified batch deletion interface)
export function useBulkDeleteDirectories() {
  return useResourceMutation({
    mutationFn: DirectoryService.bulkDeleteDirectories,
    loadingToast: {
      key: 'common.status.batchDeleting',
      params: {},
      id: 'bulk-delete-directories',
    },
    invalidate: directoryCascadeInvalidates(),
    onSuccess: ({ data, toast }) => {
      toast.success('toast.asset.directory.delete.bulkSuccess', {
        count: getAssetDeletedCount(data),
      })
    },
    errorFallbackKey: 'toast.asset.directory.delete.error',
  })
}


// Create directories in batches (bind to target)
export function useBulkCreateDirectories() {
  return useResourceMutation({
    mutationFn: (data: { targetId: number; urls: string[] }) =>
      DirectoryService.bulkCreateDirectories(data.targetId, data.urls),
    loadingToast: {
      key: 'common.status.batchCreating',
      params: {},
      id: 'bulk-create-directories',
    },
    invalidate: [
      {
        queryKey: directoryKeys.all,
        exact: false,
        refetchType: 'active',
      },
    ],
    onSuccess: ({ data, toast }) => {
      const toastPayload = resolveAssetBulkCreateToast(data.createdCount, {
        success: 'toast.asset.directory.create.success',
        partial: 'toast.asset.directory.create.partialSuccess',
      })
      if (toastPayload.variant === 'success') {
        toast.success(toastPayload.key, toastPayload.params)
      } else {
        toast.warning(toastPayload.key, toastPayload.params)
      }
    },
    errorFallbackKey: 'toast.asset.directory.create.error',
  })
}
