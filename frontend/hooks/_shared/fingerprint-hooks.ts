import { useCallback } from "react"
import { useQuery } from "@tanstack/react-query"

import { useResourceMutation } from "@/hooks/_shared/create-resource-mutation"
import type {
  CanonicalFingerprintName,
  FingerprintBatchDeleteResponse,
  FingerprintClearResponse,
  FingerprintImportResponse,
  FingerprintListParams,
  FingerprintListResponse,
  FingerprintResource,
} from "@/types/fingerprint.types"

export type FingerprintQueryOptions = { enabled?: boolean }
export type FingerprintDownload = { blob: Blob; filename: string }

export type FingerprintKeyGroup = {
  all: () => readonly unknown[]
  list: (params: FingerprintListParams) => readonly unknown[]
  detail: (name: CanonicalFingerprintName) => readonly unknown[]
}

export type FingerprintLibraryService<
  TList extends FingerprintResource,
  TDetail extends FingerprintResource,
> = {
  list: (params?: FingerprintListParams) => Promise<FingerprintListResponse<TList>>
  detail: (name: CanonicalFingerprintName) => Promise<TDetail>
  importFromFile: (file: File) => Promise<FingerprintImportResponse>
  batchDelete: (names: CanonicalFingerprintName[]) => Promise<FingerprintBatchDeleteResponse>
  clear: () => Promise<FingerprintClearResponse>
  exportAll: () => Promise<FingerprintDownload>
}

export type FingerprintHooksConfig<
  TList extends FingerprintResource,
  TDetail extends FingerprintResource,
> = {
  keys: FingerprintKeyGroup
  service: FingerprintLibraryService<TList, TDetail>
  statsKey: readonly unknown[]
}

/**
 * Shared query/mutation wiring for a read-only fingerprint library. File
 * imports own their structured error display in the dialog, so the generic
 * mutation deliberately suppresses the fallback toast on failure.
 */
export function createFingerprintHooks<
  TList extends FingerprintResource,
  TDetail extends FingerprintResource,
>(config: FingerprintHooksConfig<TList, TDetail>) {
  const toastScope = config.keys.all().map(String).join("-")

  const useList = (params: FingerprintListParams = {}, options?: FingerprintQueryOptions) =>
    useQuery({
      queryKey: config.keys.list(params),
      queryFn: () => config.service.list(params),
      ...options,
    })

  const useDetail = (name: CanonicalFingerprintName | null | undefined, options?: FingerprintQueryOptions) =>
    useQuery({
      queryKey: config.keys.detail(name ?? ""),
      queryFn: () => config.service.detail(name!),
      enabled: Boolean(name) && options?.enabled !== false,
    })

  const useImport = () =>
    useResourceMutation({
      mutationFn: (file: File) => config.service.importFromFile(file),
      loadingToast: {
        key: "tools.fingerprints.import.importing",
        params: {},
        id: `import-${toastScope}`,
      },
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      skipDefaultErrorHandler: true,
      onSuccess: ({ data, toast }) => {
        toast.success("tools.fingerprints.import.importSuccessDetail", {
          createdCount: data.createdCount,
          updatedCount: data.updatedCount,
          unchangedCount: data.unchangedCount,
        })
      },
    })

  const useBulkDelete = () =>
    useResourceMutation({
      mutationFn: (names: CanonicalFingerprintName[]) => config.service.batchDelete(names),
      loadingToast: {
        key: "common.status.batchDeleting",
        params: {},
        id: `bulk-delete-${toastScope}`,
      },
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      errorFallbackKey: "tools.fingerprints.toast.deleteFailed",
      onSuccess: ({ data, toast }) => {
        toast.success("tools.fingerprints.toast.deleteSuccess", { count: data.deletedCount })
      },
    })

  const useDeleteAll = () =>
    useResourceMutation<FingerprintClearResponse, void>({
      mutationFn: () => config.service.clear(),
      loadingToast: {
        key: "common.status.batchDeleting",
        params: {},
        id: `clear-${toastScope}`,
      },
      invalidate: [
        { queryKey: config.keys.all() },
        { queryKey: config.statsKey },
      ],
      errorFallbackKey: "tools.fingerprints.toast.deleteFailed",
      onSuccess: ({ data, toast }) => {
        toast.success("tools.fingerprints.toast.deleteSuccess", { count: data.deletedCount })
      },
    })

  const useExport = () => useCallback(() => config.service.exportAll(), [])

  return {
    useList,
    useDetail,
    useImport,
    useBulkDelete,
    useDeleteAll,
    useExport,
  }
}
