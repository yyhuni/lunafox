import React from "react"
import { useLocale, useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"

import { createOrganizationColumns } from "./organization-columns"
import { getDateLocale } from "@/lib/date-utils"
import {
  useBatchDeleteOrganizations,
  useDeleteOrganization,
  useOrganizations,
} from "@/hooks/use-organizations"
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

import type { Organization } from "@/types/organization.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"

const ORGANIZATION_FILTER_FIELDS: BusinessListFilterCompilerConfig = {
  search: { field: "displayName", operator: "=" },
  facets: {},
}

const ORGANIZATION_SORTABLE_FIELDS: Record<string, BusinessListSortableFieldConfig> = {
  displayName: { orderBy: "displayName", firstDirection: "asc" },
  createdAt: { orderBy: "createdAt", firstDirection: "desc" },
}

const ORGANIZATION_DEFAULT_SORTING: BusinessListSorting = { field: "createdAt", direction: "desc" }

function organizationQueryFieldToColumnId(field: string) {
  return field === "displayName" ? "name" : field
}

function organizationColumnIdToQueryField(columnId: string) {
  return columnId === "name" ? "displayName" : columnId
}

export function useOrganizationListState() {
  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tConfirm = useTranslations("common.confirm")
  const tOrg = useTranslations("organization")
  const tScanInitiate = useTranslations("scan.initiate")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        organization: tColumns("organization.organization"),
        description: tColumns("common.description"),
        totalTargets: tColumns("organization.totalTargets"),
        added: tColumns("organization.added"),
      },
      actions: {
        scheduleScan: tTooltips("scheduleScan"),
        delete: tCommon("actions.delete"),
        openMenu: tCommon("actions.openMenu"),
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        organizationDetails: tTooltips("organizationDetails"),
        initiateScan: tTooltips("initiateScan"),
      },
    }),
    [tColumns, tCommon, tTooltips]
  )

  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [addDialogOpen, setAddDialogOpen] = React.useState(false)
  const [initiateScanDialogOpen, setInitiateScanDialogOpen] = React.useState(false)
  const [scheduleScanDialogOpen, setScheduleScanDialogOpen] = React.useState(false)
  const [organizationToDelete, setOrganizationToDelete] = React.useState<Organization | null>(null)
  const [organizationToView, setOrganizationToView] = React.useState<Organization | null>(null)
  const [organizationToScan, setOrganizationToScan] = React.useState<Organization | null>(null)
  const [organizationToSchedule, setOrganizationToSchedule] = React.useState<Organization | null>(null)
  const [selectedOrganizations, setSelectedOrganizations] = React.useState<Organization[]>([])
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [bulkInitiateScanDialogOpen, setBulkInitiateScanDialogOpen] = React.useState(false)

  const [query, setQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
    sorting: ORGANIZATION_DEFAULT_SORTING,
  }))
  const [pageTokens, setPageTokens] = React.useState<Record<number, string | undefined>>({ 1: undefined })

  const page = query.pageIndex ?? 1
  const pageSize = query.pageSize
  const searchQuery = query.search ?? ""
  const pagination = React.useMemo(
    () => ({ pageIndex: page - 1, pageSize }),
    [page, pageSize]
  )
  const sorting = React.useMemo<SortingState>(() => {
    if (!query.sorting) return []
    return [{
      id: organizationQueryFieldToColumnId(query.sorting.field),
      desc: query.sorting.direction === "desc",
    }]
  }, [query.sorting])

  const compiledFilter = compileBusinessListFilter(
    { search: query.search, filters: query.filters },
    ORGANIZATION_FILTER_FIELDS
  )
  const compiledOrderBy = compileBusinessListOrderBy(query.sorting, ORGANIZATION_SORTABLE_FIELDS)

  const {
    data,
    isPlaceholderData,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useOrganizations(
    {
      pageSize,
      pageToken: query.pageToken,
      filter: compiledFilter,
      orderBy: compiledOrderBy,
      pageIndex: page,
    },
    { enabled: true }
  )

  const nextPageToken = getCurrentCursorNextPageToken(
    data?.nextPageToken,
    isPlaceholderData,
  )

  React.useEffect(() => {
    if (nextPageToken) {
      setPageTokens((tokens) => ({ ...tokens, [page + 1]: nextPageToken }))
    }
  }, [nextPageToken, page])

  const resetPaging = React.useCallback(() => {
    setPageTokens({ 1: undefined })
  }, [])

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

  const deleteOrganization = useDeleteOrganization()
  const batchDeleteOrganizations = useBatchDeleteOrganizations()

  const formatDate = React.useCallback(
    (dateString: string): string => {
      return new Date(dateString).toLocaleString(getDateLocale(locale), {
        year: "numeric",
        month: "numeric",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      })
    },
    [locale]
  )

  const handleDelete = React.useCallback((org: Organization) => {
    setOrganizationToDelete(org)
    setDeleteDialogOpen(true)
  }, [])

  const handleViewDetail = React.useCallback((org: Organization) => {
    setOrganizationToView(org)
  }, [])

  const handleDetailOpenChange = React.useCallback((open: boolean) => {
    if (!open) {
      setOrganizationToView(null)
    }
  }, [])

  const handleInitiateScan = React.useCallback((org: Organization) => {
    setOrganizationToScan(org)
    setInitiateScanDialogOpen(true)
  }, [])

  const handleScheduleScan = React.useCallback((org: Organization) => {
    setOrganizationToSchedule(org)
    setScheduleScanDialogOpen(true)
  }, [])

  const columns = React.useMemo(
    () =>
      createOrganizationColumns({
        formatDate,
        handleViewDetail,
        handleDelete,
        handleInitiateScan,
        handleScheduleScan,
        t: translations,
      }),
    [
      formatDate,
      handleViewDetail,
      handleDelete,
      handleInitiateScan,
      handleScheduleScan,
      translations,
    ]
  )

  const confirmDelete = async () => {
    if (!organizationToDelete) return

    setDeleteDialogOpen(false)
    setOrganizationToDelete(null)

    deleteOrganization.mutate(Number(organizationToDelete.id))
  }

  const handleBulkDelete = () => {
    if (selectedOrganizations.length === 0) {
      return
    }
    setBulkDeleteDialogOpen(true)
  }

  const handleBulkInitiateScan = () => {
    if (selectedOrganizations.length === 0) {
      return
    }
    setBulkInitiateScanDialogOpen(true)
  }

  const handleBulkInitiateScanSuccess = React.useCallback(() => {
    setBulkInitiateScanDialogOpen(false)
    setSelectedOrganizations([])
  }, [])

  const confirmBulkDelete = async () => {
    if (selectedOrganizations.length === 0) return

    const deletedIds = selectedOrganizations.map((org) => Number(org.id))

    setBulkDeleteDialogOpen(false)
    setSelectedOrganizations([])

    batchDeleteOrganizations.mutate(deletedIds)
  }

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

  const handleSortingChange = React.useCallback((nextSorting: SortingState) => {
    const nextColumnId = nextSorting[0]?.id
    if (!nextColumnId) {
      resetPaging()
      setQuery((current) => applyBusinessListControlChange(current, { sorting: ORGANIZATION_DEFAULT_SORTING }))
      return
    }

    resetPaging()
    setQuery((current) => toggleBusinessListSorting(
      current,
      organizationColumnIdToQueryField(nextColumnId),
      ORGANIZATION_SORTABLE_FIELDS,
      ORGANIZATION_DEFAULT_SORTING
    ))
  }, [resetPaging])

  const cursorPaginationSummary: CursorPaginationSummary = {
    total: data?.totalSize ?? data?.total ?? 0,
  }
  const paginationNavigation: CursorPaginationNavigation = getCursorPaginationNavigation({
    currentPage: page,
    pageTokens,
    nextPageToken,
  })

  return {
    tCommon,
    tConfirm,
    tOrg,
    tScanInitiate,
    data,
    organizations: data?.organizations ?? [],
    isLoading,
    error,
    refetch,
    translations,
    columns,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    sorting,
    handleSortingChange,
    searchQuery,
    isSearching,
    commitSearch,
    deleteDialogOpen,
    setDeleteDialogOpen,
    addDialogOpen,
    setAddDialogOpen,
    initiateScanDialogOpen,
    setInitiateScanDialogOpen,
    scheduleScanDialogOpen,
    setScheduleScanDialogOpen,
    organizationToDelete,
    organizationToView,
    organizationToScan,
    organizationToSchedule,
    setOrganizationToScan,
    setOrganizationToSchedule,
    handleDetailOpenChange,
    selectedOrganizations,
    setSelectedOrganizations,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    bulkInitiateScanDialogOpen,
    setBulkInitiateScanDialogOpen,
    deleteOrganization,
    batchDeleteOrganizations,
    handleDelete,
    handleViewDetail,
    handleInitiateScan,
    handleScheduleScan,
    confirmDelete,
    handleBulkDelete,
    handleBulkInitiateScan,
    handleBulkInitiateScanSuccess,
    confirmBulkDelete,
  }
}

export type OrganizationListState = ReturnType<typeof useOrganizationListState>
