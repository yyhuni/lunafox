"use client"

import * as React from "react"
import type { ColumnDef, SortingState } from "@tanstack/react-table"
import { useTranslations } from "next-intl"

import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { DetailDrawer } from "@/components/shared/detail-drawer"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { ImportFingerprintDialog } from "@/components/fingerprints/import-fingerprint-dialog"
import { useFingerprintFacetOptions } from "@/hooks/use-fingerprints/filter-options"
import { normalizeError } from "@/lib/errors/normalize-error"
import { getErrorMessage } from "@/lib/error-utils"
import { saveBlobAsFile } from "@/lib/file-save-utils"
import { toastFeedback } from "@/lib/toast-helpers"
import {
  createBusinessListQuery,
  type BusinessListQuery,
} from "@/components/shared/data-table/business-list-query"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
  PaginationState,
} from "@/types/data-table.types"
import type {
  CanonicalFingerprintName,
  FingerprintFilterOption,
  FingerprintFilterOptionField,
  FingerprintLibrary,
  FingerprintListParams,
  FingerprintListResponse,
  FingerprintResource,
} from "@/types/fingerprint.types"

import { FINGERPRINT_LIBRARY_LIST_CONFIG } from "./fingerprint-library-list-config"
import {
  buildFingerprintLibraryListParams,
  useFingerprintLibraryQueryState,
} from "./fingerprint-library-query-state"
import type { FingerprintDownload } from "@/hooks/_shared/fingerprint-hooks"

type QueryResult<T> = {
  data?: FingerprintListResponse<T>
  isLoading: boolean
  isFetching?: boolean
  isPlaceholderData?: boolean
  error: unknown
  refetch: () => unknown
}

type DetailQueryResult<T> = {
  data?: T
  isLoading: boolean
  isFetching: boolean
  error: unknown
  refetch: () => unknown
}

type AsyncMutation<TArgument> = {
  mutateAsync: (argument: TArgument) => Promise<unknown>
}

type FingerprintTableRenderProps<T extends FingerprintResource> = {
  data: T[]
  columns: ColumnDef<T, unknown>[]
  query: BusinessListQuery
  filterOptions: Partial<Record<FingerprintFilterOptionField, FingerprintFilterOption[]>>
  onSearchChange: (value: string) => void
  isSearching: boolean
  onFacetChange: (field: string, values: string[]) => void
  onSortingChange: (sorting: SortingState) => void
  onSelectionChange: (rows: T[]) => void
  onRowClick: (row: T) => void
  selectedRows: T[]
  onImport: () => void
  onExport: () => void
  onBulkDelete: () => void
  onDeleteAll: () => void
  totalCount: number
  pagination: PaginationState
  cursorPaginationSummary: CursorPaginationSummary
  paginationNavigation: CursorPaginationNavigation
  onPaginationChange: (pagination: PaginationState) => void
  loading?: boolean
  initialLoading?: boolean
  loadingRowCount: number
  stableSurfaceRowCount: number
}

type FingerprintDrawerRenderProps<T extends FingerprintResource> = {
  fingerprint: T | null
  open: boolean
  loading: boolean
  onOpenChange: (open: boolean) => void
  formatDate: (value: string) => string
}

type FingerprintLibraryWorkspaceProps<
  TList extends FingerprintResource,
  TDetail extends FingerprintResource,
> = {
  library: FingerprintLibrary
  owner: string
  columns: ColumnDef<TList, unknown>[]
  formatDate: (value: string) => string
  useList: (params: FingerprintListParams) => QueryResult<TList>
  useDetail: (name: CanonicalFingerprintName | null) => DetailQueryResult<TDetail>
  useBulkDelete: () => AsyncMutation<CanonicalFingerprintName[]>
  useClear: () => AsyncMutation<void>
  useExport: () => () => Promise<FingerprintDownload>
  renderTable: (props: FingerprintTableRenderProps<TList>) => React.ReactNode
  renderDrawer: (props: FingerprintDrawerRenderProps<TDetail>) => React.ReactNode
}

/**
 * Shared read-only workflow for the six persisted fingerprint libraries.
 * Each page supplies only its columns and detail presentation; canonical names
 * remain the single resource handle for selection, detail fetches, and delete.
 */
export function FingerprintLibraryWorkspace<
  TList extends FingerprintResource,
  TDetail extends FingerprintResource,
>({
  library,
  owner,
  columns,
  formatDate,
  useList,
  useDetail,
  useBulkDelete,
  useClear,
  useExport,
  renderTable,
  renderDrawer,
}: FingerprintLibraryWorkspaceProps<TList, TDetail>) {
  const tFingerprints = useTranslations("tools.fingerprints")
  const libraryConfig = FINGERPRINT_LIBRARY_LIST_CONFIG[library]
  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: libraryConfig.defaultSorting,
  }))
  const [selectedRows, setSelectedRows] = React.useState<TList[]>([])
  const [selectedName, setSelectedName] = React.useState<CanonicalFingerprintName | null>(null)
  const [importDialogOpen, setImportDialogOpen] = React.useState(false)
  const facetFields = React.useMemo(
    () => Object.keys(libraryConfig.filterConfig.facets) as FingerprintFilterOptionField[],
    [libraryConfig]
  )
  const facetOptionsQuery = useFingerprintFacetOptions(library, facetFields)
  const filterOptions = facetOptionsQuery.options

  const request = buildFingerprintLibraryListParams(
    query,
    libraryConfig.filterConfig,
    libraryConfig.sortableFields
  )
  const listQuery = useList(request)
  const detailQuery = useDetail(selectedName)
  const bulkDeleteMutation = useBulkDelete()
  const clearMutation = useClear()
  const exportAll = useExport()
  const queryState = useFingerprintLibraryQueryState({
    query,
    setQuery,
    data: listQuery.data,
    isPlaceholderData: listQuery.isPlaceholderData,
    filterConfig: libraryConfig.filterConfig,
    sortableFields: libraryConfig.sortableFields,
    defaultSorting: libraryConfig.defaultSorting,
  })
  const resetPagination = queryState.resetPagination

  const fingerprints = listQuery.data?.results ?? []
  const isInitialLoading = (listQuery.isLoading && !listQuery.data)
    || (facetFields.length > 0 && facetOptionsQuery.isLoading)
  const loadingRowCount = getDataTableSkeletonRowCount(queryState.pageSize)
  const detailError = selectedName !== null ? detailQuery.error : null

  const handleDrawerOpenChange = React.useCallback((open: boolean) => {
    if (!open) {
      setSelectedName(null)
    }
  }, [])

  const handleExport = React.useCallback(async () => {
    try {
      const download = await exportAll()
      saveBlobAsFile(download.blob, download.filename)
      toastFeedback.success(tFingerprints("toast.exportSuccess"))
    } catch (error) {
      toastFeedback.error(getErrorMessage(error) || tFingerprints("toast.exportFailed"))
    }
  }, [exportAll, tFingerprints])

  const handleBulkDelete = React.useCallback(async () => {
    if (selectedRows.length === 0) {
      return
    }

    try {
      const selectedNames = selectedRows.map((row) => row.name)
      await bulkDeleteMutation.mutateAsync(selectedNames)
      resetPagination()
      setSelectedRows([])
      setSelectedName((current) => current && selectedNames.includes(current) ? null : current)
    } catch {
      // The shared mutation already presents the localized failure feedback.
    }
  }, [bulkDeleteMutation, resetPagination, selectedRows])

  const handleClear = React.useCallback(async () => {
    try {
      await clearMutation.mutateAsync(undefined)
      resetPagination()
      setSelectedRows([])
      setSelectedName(null)
    } catch {
      // The shared mutation already presents the localized failure feedback.
    }
  }, [clearMutation, resetPagination])

  const tableProps: FingerprintTableRenderProps<TList> = {
    data: fingerprints,
    columns,
    query,
    filterOptions,
    onSearchChange: queryState.commitSearch,
    isSearching: Boolean(query.search && listQuery.isFetching),
    onFacetChange: queryState.updateFacet,
    onSortingChange: queryState.onSortingChange,
    onSelectionChange: setSelectedRows,
    onRowClick: (row) => setSelectedName(row.name),
    selectedRows,
    onImport: () => setImportDialogOpen(true),
    onExport: () => { void handleExport() },
    onBulkDelete: () => { void handleBulkDelete() },
    onDeleteAll: () => { void handleClear() },
    totalCount: listQuery.data?.totalSize ?? 0,
    pagination: queryState.pagination,
    cursorPaginationSummary: queryState.cursorPaginationSummary,
    paginationNavigation: queryState.paginationNavigation,
    onPaginationChange: queryState.onPaginationChange,
    loadingRowCount,
    stableSurfaceRowCount: loadingRowCount,
  }

  if (listQuery.error || facetOptionsQuery.error) {
    return (
      <AppErrorState
        error={normalizeError(listQuery.error ?? facetOptionsQuery.error, { notFoundKind: "unexpected-error" })}
        onRetry={listQuery.error ? listQuery.refetch : facetOptionsQuery.refetch}
      />
    )
  }

  return (
    <>
      <ContentHandoff
        owner={owner}
        layer="workspace"
        isLoading={isInitialLoading}
        skeleton={renderTable({
          ...tableProps,
          data: [],
          selectedRows: [],
          loading: true,
          initialLoading: true,
        })}
      >
        {renderTable(tableProps)}
      </ContentHandoff>

      <ImportFingerprintDialog
        open={importDialogOpen}
        onOpenChange={setImportDialogOpen}
        onSuccess={resetPagination}
      />

      {detailError ? (
        <DetailDrawer
          open
          onOpenChange={handleDrawerOpenChange}
          title={tFingerprints("title")}
        >
          <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
            <AppErrorState
              error={normalizeError(detailError)}
              onRetry={detailQuery.refetch}
              variant="section"
            />
          </div>
        </DetailDrawer>
      ) : renderDrawer({
        fingerprint: detailQuery.data ?? null,
        open: selectedName !== null,
        loading: detailQuery.isLoading || detailQuery.isFetching,
        onOpenChange: handleDrawerOpenChange,
        formatDate,
      })}
    </>
  )
}
