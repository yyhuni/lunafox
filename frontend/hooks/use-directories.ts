import { useCallback } from "react"
import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { useResourceMutation } from '@/hooks/_shared/create-resource-mutation'
import { createResourceKeys } from "@/hooks/_shared/query-keys"
import {
  getAssetDeletedCount,
  resolveAssetBulkCreateToast,
} from '@/hooks/_shared/asset-mutation-helpers'
import { DirectoryService } from '@/services/directory.service'
import type { DirectoryFilterOptionField, DirectoryListQueryParams } from '@/types/directory.types'
import type { WebsiteAssetScope } from '@/types/website.types'

// Query Keys
const directoryKeyBase = createResourceKeys("directories")

export const directoryKeys = {
  ...directoryKeyBase,
  target: (targetId: number, params: DirectoryListQueryParams, websiteScope?: WebsiteAssetScope) =>
    [...directoryKeyBase.all, 'target', targetId, params, websiteScope] as const,
  scan: (scanId: number, params: DirectoryListQueryParams) =>
    [...directoryKeyBase.all, 'scan', scanId, params] as const,
  filterOptions: (scope: "target" | "scan", id: number, field: DirectoryFilterOptionField) =>
    [...directoryKeyBase.all, "filterOptions", scope, id, field] as const,
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
  params: DirectoryListQueryParams,
  options?: { enabled?: boolean; websiteScope?: WebsiteAssetScope }
) {
  const resolved = {
    pageSize: params.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }
  return useQuery({
    queryKey: directoryKeys.target(targetId, resolved, options?.websiteScope),
    queryFn: () => DirectoryService.getTargetDirectories(targetId, resolved),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

// Get a list of scanned directories
export function useScanDirectories(
  scanId: number,
  params: DirectoryListQueryParams,
  options?: { enabled?: boolean }
) {
  const resolved = {
    pageSize: params.pageSize ?? 10,
    pageToken: params?.pageToken,
    filter: params?.filter,
    orderBy: params?.orderBy,
  }
  return useQuery({
    queryKey: directoryKeys.scan(scanId, resolved),
    queryFn: () => DirectoryService.getScanDirectories(scanId, resolved),
    enabled: options?.enabled ?? true,
    placeholderData: keepPreviousData,
  })
}

export function useTargetDirectoryFilterOptions(
  targetId: number,
  field: DirectoryFilterOptionField,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: directoryKeys.filterOptions("target", targetId, field),
    queryFn: () => DirectoryService.getTargetDirectoryFilterOptions(targetId, field),
    enabled: options?.enabled ?? !!targetId,
    placeholderData: keepPreviousData,
  })
}

export function useScanDirectoryFilterOptions(
  scanId: number,
  field: DirectoryFilterOptionField,
  options?: { enabled?: boolean }
) {
  return useQuery({
    queryKey: directoryKeys.filterOptions("scan", scanId, field),
    queryFn: () => DirectoryService.getScanDirectoryFilterOptions(scanId, field),
    enabled: options?.enabled ?? !!scanId,
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
    mutationFn: (data: { targetId: number; ids: number[] }) =>
      DirectoryService.bulkDeleteDirectories(data.targetId, data.ids),
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

export function useExportDirectories({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  return useCallback(() => {
    if (scanId) {
      return DirectoryService.exportDirectoriesByScanId(scanId)
    }
    if (targetId) {
      return DirectoryService.exportDirectoriesByTargetId(targetId)
    }
    return Promise.resolve(null)
  }, [scanId, targetId])
}
