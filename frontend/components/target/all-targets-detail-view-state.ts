import React from "react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"

import { createAllTargetsColumns } from "@/components/target/all-targets-columns"
import { useTargets, useDeleteTarget, useBatchDeleteTargets } from "@/hooks/use-targets"
import {
  applyBusinessListControlChange,
  compileBusinessListFilter,
  compileBusinessListOrderBy,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  setBusinessListPage,
  toggleBusinessListSorting,
  type BusinessListFilterCompilerConfig,
  type BusinessListSortableFieldConfig,
  type BusinessListSorting,
} from "@/components/shared/data-table/business-list-query"
import { pushWithRouteProgress } from "@/components/route-progress"
import { formatDate } from "@/lib/utils"

import type { Target } from "@/types/target.types"
import type { TargetType } from "@/types/target.types"
import type { AllTargetsTranslations } from "@/components/target/all-targets-columns"
import type { CursorPaginationNavigation } from "@/types/data-table.types"

const TARGET_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "displayName", operator: "=" },
  facets: {
    type: { field: "type", operator: "==" },
  },
}

const TARGET_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  displayName: { orderBy: "displayName", firstDirection: "asc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
  lastScannedAt: { orderBy: "lastScannedAt", firstDirection: "desc" },
}

const TARGET_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

function targetQueryFieldToColumnId(field: string) {
  return field === "displayName" ? "name" : field
}

function targetColumnIdToQueryField(columnId: string) {
  return columnId === "name" ? "displayName" : columnId
}

interface AllTargetsDetailViewStateOptions {
  className?: string
  tableClassName?: string
  hideToolbar?: boolean
  hidePagination?: boolean
}

export function useAllTargetsDetailViewState({
  className,
  tableClassName,
  hideToolbar,
  hidePagination,
}: AllTargetsDetailViewStateOptions) {
  const router = useRouter()
  const tColumns = useTranslations("columns")
  const tTooltips = useTranslations("tooltips")
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tTarget = useTranslations("target")
  const tScanInitiate = useTranslations("scan.initiate")

  const translations: AllTargetsTranslations = React.useMemo(
    () => ({
      columns: {
        target: tColumns("target.target"),
        organization: tColumns("organization.organization"),
        addedOn: tColumns("target.addedOn"),
        lastScanned: tColumns("target.lastScanned"),
        actions: tColumns("common.actions"),
      },
      actions: {
        scheduleScan: tTooltips("scheduleScan"),
        delete: tCommon("actions.delete"),
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
        openMenu: tCommon("actions.openMenu"),
      },
      tooltips: {
        targetDetails: tTooltips("targetDetails"),
        targetSummary: tTooltips("targetSummary"),
        initiateScan: tTooltips("initiateScan"),
        clickToCopy: tTooltips("clickToCopy"),
        copied: tTooltips("copied"),
      },
      targetTypes: {
        domain: tTarget("types.domain"),
        ip: tTarget("types.ip"),
        cidr: tTarget("types.cidr"),
      },
    }),
    [tColumns, tCommon, tTarget, tTooltips]
  )

  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: TARGET_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })
  const [selectedTargets, setSelectedTargets] = React.useState<Target[]>([])
  const [isAddDialogOpen, setIsAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [targetToDelete, setTargetToDelete] = React.useState<Target | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [bulkInitiateScanDialogOpen, setBulkInitiateScanDialogOpen] = React.useState(false)
  const [shouldPrefetchOrgs, setShouldPrefetchOrgs] = React.useState(false)
  const [initiateScanDialogOpen, setInitiateScanDialogOpen] = React.useState(false)
  const [scheduleScanDialogOpen, setScheduleScanDialogOpen] = React.useState(false)
  const [targetToScan, setTargetToScan] = React.useState<Target | null>(null)
  const [targetToSchedule, setTargetToSchedule] = React.useState<Target | null>(null)

  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const searchQuery = query.search ?? ""
  const typeFilter = (query.filters.type ?? []) as TargetType[]
  const pagination = React.useMemo(
    () => ({ pageIndex: page - 1, pageSize }),
    [page, pageSize]
  )
  const sorting = React.useMemo<SortingState>(() => {
    if (!query.sorting) return []
    return [{
      id: targetQueryFieldToColumnId(query.sorting.field),
      desc: query.sorting.direction === "desc",
    }]
  }, [query.sorting])

  const compiledFilter = compileBusinessListFilter(
    { search: query.search, filters: query.filters },
    TARGET_FILTER_FIELDS
  )
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, TARGET_SORTABLE_FIELDS)

  const { data, isLoading, isFetching, isPlaceholderData, error, refetch } = useTargets({
    pageSize,
    pageToken: query.pageToken,
    filter: compiledFilter,
    orderBy: compiledOrderBy,
  })

  const nextPageToken = getCurrentCursorNextPageToken(
    data?.nextPageToken,
    isPlaceholderData,
  )

  React.useEffect(() => {
    if (nextPageToken) {
      setPageTokens((tokens) => ({ ...tokens, [page + 1]: nextPageToken }))
    }
  }, [nextPageToken, page])

  const paginationNavigation: CursorPaginationNavigation = getCursorPaginationNavigation({
    currentPage: page,
    pageTokens,
    nextPageToken,
  })

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])

  const handlePaginationChange = React.useCallback(
    (newPagination: { pageIndex: number; pageSize: number }) => {
      const nextPage = newPagination.pageIndex + 1
      if (newPagination.pageSize !== pageSize) {
        resetPaging()
        setQuery((current) => applyBusinessListControlChange(current, { pageSize: newPagination.pageSize }))
        return
      }

      if (nextPage === 1) {
        resetPaging()
        setQuery((current) => setBusinessListPage(current, { pageIndex: 1, pageToken: undefined }))
        return
      }

      const transition = getCursorPageTransition({
        currentPage: page,
        pageTokens,
        nextPageToken,
        requestedPage: nextPage,
      })
      if (!transition.reachable) return

      setQuery((current) => setBusinessListPage(current, {
        pageIndex: nextPage,
        pageToken: transition.pageToken,
      }))
    },
    [nextPageToken, page, pageSize, pageTokens, resetPaging]
  )

  const handleTypeFilterChange = (value: TargetType[]) => {
    resetPaging()
    setQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, type: value },
    }))
  }

  const deleteTargetMutation = useDeleteTarget()
  const batchDeleteMutation = useBatchDeleteTargets()
  const isSearching = isFetching

  const commitSearch = React.useCallback((value: string) => {
    const normalizedSearch = value.trim()
    if ((query.search ?? "") === normalizedSearch) {
      return
    }
    resetPaging()
    setQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [query.search, resetPaging])

  const handleSortingChange = React.useCallback((nextSorting: SortingState) => {
    const nextColumnId = nextSorting[0]?.id
    if (!nextColumnId) {
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, { sorting: TARGET_DEFAULT_SORTING }))
      return
    }

    resetPaging()
    setQuery((current) => toggleBusinessListSorting(
      current,
      targetColumnIdToQueryField(nextColumnId),
      TARGET_SORTABLE_FIELDS,
      TARGET_DEFAULT_SORTING
    ))
  }, [resetPaging])

  const handleAddTarget = React.useCallback(() => {
    setIsAddDialogOpen(true)
  }, [])

  const handleDeleteTarget = React.useCallback((target: Target) => {
    setTargetToDelete(target)
    setDeleteDialogOpen(true)
  }, [])

  const confirmDelete = async () => {
    if (!targetToDelete) return

    try {
      await deleteTargetMutation.mutateAsync({ id: targetToDelete.id, name: targetToDelete.name })
      setDeleteDialogOpen(false)
      setTargetToDelete(null)
    } catch {
      // Error already handled in hook
    }
  }

  const handleBatchDelete = React.useCallback(() => {
    if (selectedTargets.length === 0) return
    setBulkDeleteDialogOpen(true)
  }, [selectedTargets])

  const handleBulkInitiateScan = React.useCallback(() => {
    if (selectedTargets.length === 0) return
    setBulkInitiateScanDialogOpen(true)
  }, [selectedTargets.length])

  const handleBulkInitiateScanSuccess = React.useCallback(() => {
    setBulkInitiateScanDialogOpen(false)
    setSelectedTargets([])
  }, [])

  const confirmBulkDelete = async () => {
    if (selectedTargets.length === 0) return

    try {
      await batchDeleteMutation.mutateAsync({
        ids: selectedTargets.map((t) => t.id),
      })
      setBulkDeleteDialogOpen(false)
      setSelectedTargets([])
    } catch {
      // Error already handled in hook
    }
  }

  const handleInitiateScan = React.useCallback((target: Target) => {
    setTargetToScan(target)
    setInitiateScanDialogOpen(true)
  }, [])

  const handleScheduleScan = React.useCallback((target: Target) => {
    setTargetToSchedule(target)
    setScheduleScanDialogOpen(true)
  }, [])

  const navigate = React.useCallback(
    (path: string) => {
      pushWithRouteProgress(router, path)
    },
    [router]
  )

  const columns = React.useMemo(
    () =>
      createAllTargetsColumns({
        formatDate,
        navigate,
        handleDelete: handleDeleteTarget,
        handleInitiateScan,
        handleScheduleScan,
        t: translations,
      }),
    [handleDeleteTarget, handleInitiateScan, handleScheduleScan, navigate, translations]
  )

  return {
    tCommon,
    tConfirm,
    tTarget,
    tScanInitiate,
    data,
    isLoading,
    error,
    refetch,
    targets: data?.results ?? [],
    totalCount: data?.totalSize ?? data?.total ?? 0,
    columns,
    pagination,
    paginationNavigation,
    handlePaginationChange,
    sorting,
    handleSortingChange,
    searchQuery,
    isSearching,
    commitSearch,
    typeFilter,
    handleTypeFilterChange,
    className,
    tableClassName,
    hideToolbar,
    hidePagination,
    isAddDialogOpen,
    setIsAddDialogOpen,
    deleteDialogOpen,
    setDeleteDialogOpen,
    targetToDelete,
    deleteTargetMutation,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    bulkInitiateScanDialogOpen,
    setBulkInitiateScanDialogOpen,
    batchDeleteMutation,
    selectedTargets,
    setSelectedTargets,
    shouldPrefetchOrgs,
    setShouldPrefetchOrgs,
    initiateScanDialogOpen,
    setInitiateScanDialogOpen,
    scheduleScanDialogOpen,
    setScheduleScanDialogOpen,
    targetToScan,
    setTargetToScan,
    targetToSchedule,
    setTargetToSchedule,
    handleAddTarget,
    handleBulkInitiateScan,
    handleBulkInitiateScanSuccess,
    handleBatchDelete,
    confirmDelete,
    confirmBulkDelete,
  }
}

export type AllTargetsDetailViewState = ReturnType<typeof useAllTargetsDetailViewState>
