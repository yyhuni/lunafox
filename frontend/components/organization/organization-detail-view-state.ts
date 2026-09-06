import React from "react"
import { useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"

import { useOrganization, useOrganizationTargets, useUnlinkTargetsFromOrganization } from "@/hooks/use-organizations"
import {
  useDeleteScheduledScan,
  useScheduledScans,
  useToggleScheduledScan,
} from "@/hooks/use-scheduled-scans"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import { pushWithRouteProgress } from "@/components/route-progress"
import {
  applyBusinessListControlChange,
  createBusinessListQuery,
  getCurrentCursorNextPageToken,
  getCursorPaginationNavigation,
  getCursorPageTransition,
  setBusinessListPage,
} from "@/components/shared/data-table/business-list-query"
import { getDateLocale } from "@/lib/date-utils"
import { createScheduledScanColumns } from "@/components/scan/scheduled/scheduled-scan-columns"
import { useInteractionOpenLoader } from "@/components/shared/loading/use-interaction-open-loader"
import { createTargetColumns } from "./targets/targets-columns"

import type { ScheduledScan } from "@/types/scheduled-scan.types"
import type { Target } from "@/types/target.types"
import type { TargetType } from "@/types/target.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"

interface OrganizationDetailViewStateOptions {
  organizationId: string
}

export type OrganizationDetailTab = "targets" | "scheduled"
export type OrganizationScheduledStatusFilter = Array<"enabled" | "paused">
export type OrganizationScheduledWorkflowFilter = string[]
export type OrganizationTargetTypeFilter = TargetType[]

function buildTargetTypeFilterQuery(typeFilter: OrganizationTargetTypeFilter): string | undefined {
  if (typeFilter.length <= 1) return undefined

  return typeFilter
    .map((type) => `type="${type}"`)
    .join(" || ")
}

function combineTargetFilterQuery(searchQuery: string, typeFilter: OrganizationTargetTypeFilter): string | undefined {
  const trimmedSearchQuery = searchQuery.trim()
  const typeFilterQuery = buildTargetTypeFilterQuery(typeFilter)

  if (!typeFilterQuery) return trimmedSearchQuery || undefined
  if (!trimmedSearchQuery) return typeFilterQuery

  return `(${trimmedSearchQuery}) && (${typeFilterQuery})`
}

const loadCreateScheduledScanDialog = () =>
  import("@/components/scan/scheduled/create-scheduled-scan-dialog").then(
    (mod) => mod.CreateScheduledScanDialog
  )

const loadEditScheduledScanDialog = () =>
  import("@/components/scan/scheduled/edit-scheduled-scan-dialog").then(
    (mod) => mod.EditScheduledScanDialog
  )

const EMPTY_SCHEDULED_SCANS: ScheduledScan[] = []

export function useOrganizationDetailViewState({ organizationId }: OrganizationDetailViewStateOptions) {
  const {
    pendingInteraction,
    cancelPending,
    openAfterLoad,
  } = useInteractionOpenLoader<"create-dialog" | "edit-dialog">()
  const organizationNumericId = Number.parseInt(organizationId, 10)
  const [activeTab, setActiveTab] = React.useState<OrganizationDetailTab>("targets")
  const [selectedTargets, setSelectedTargets] = React.useState<Target[]>([])
  const [isAddDialogOpen, setIsAddDialogOpen] = React.useState(false)
  const [isEditDialogOpen, setIsEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [targetToDelete, setTargetToDelete] = React.useState<Target | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [scheduledPagination, setScheduledPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })
  const [scheduledSearchQuery, setScheduledSearchQuery] = React.useState("")
  const [scheduledStatusFilter, setScheduledStatusFilter] =
    React.useState<OrganizationScheduledStatusFilter>([])
  const [scheduledWorkflowFilter, setScheduledWorkflowFilter] =
    React.useState<OrganizationScheduledWorkflowFilter>([])
  const [isCreateScheduledScanDialogOpen, setIsCreateScheduledScanDialogOpen] = React.useState(false)
  const [isEditScheduledScanDialogOpen, setIsEditScheduledScanDialogOpen] = React.useState(false)
  const [scheduledScanToEdit, setScheduledScanToEdit] = React.useState<ScheduledScan | null>(null)
  const [scheduledScanToDelete, setScheduledScanToDelete] = React.useState<ScheduledScan | null>(null)
  const [scheduledScanDeleteDialogOpen, setScheduledScanDeleteDialogOpen] = React.useState(false)

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tTarget = useTranslations("target")
  const tScan = useTranslations("scan")
  const tConfirm = useTranslations("common.confirm")
  const tOrg = useTranslations("organization")
  const locale = useLocale()

  const translations = React.useMemo(
    () => ({
      columns: {
        targetName: tColumns("target.target"),
        type: tColumns("common.type"),
        addedOn: tColumns("target.addedOn"),
        lastScanned: tColumns("target.lastScanned"),
      },
      actions: {
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      tooltips: {
        viewDetails: tTooltips("viewDetails"),
        unlinkTarget: tTooltips("unlinkTarget"),
        clickToCopy: tTooltips("clickToCopy"),
        copied: tTooltips("copied"),
      },
      types: {
        domain: tTarget("types.domain"),
        ip: tTarget("types.ip"),
        cidr: tTarget("types.cidr"),
      },
    }),
    [tColumns, tCommon, tTooltips, tTarget]
  )

  const scheduledTranslations = React.useMemo(
    () => ({
      columns: {
        taskName: tColumns("scheduledScan.taskName"),
        cronExpression: tColumns("scheduledScan.cronExpression"),
        scope: tColumns("scheduledScan.scope"),
        status: tColumns("common.status"),
        nextRun: tColumns("scheduledScan.nextRun"),
        handoffResults: tColumns("scheduledScan.handoffResults"),
        trigger: tColumns("scheduledScan.trigger"),
        success: tColumns("scheduledScan.success"),
        failure: tColumns("scheduledScan.failure"),
        lastRun: tColumns("scheduledScan.lastRun"),
      },
      actions: {
        editTask: tScan("editTask"),
        delete: tCommon("actions.delete"),
        openMenu: tCommon("actions.openMenu"),
        selectAll: tCommon("actions.selectAll"),
        selectRow: tCommon("actions.selectRow"),
      },
      status: {
        enabled: tCommon("status.enabled"),
        disabled: tCommon("status.disabled"),
      },
      cron: {
        everyMinute: tScan("cron.everyMinute"),
        everyNMinutes: tScan.raw("cron.everyNMinutes") as string,
        everyHour: tScan.raw("cron.everyHour") as string,
        everyNHours: tScan.raw("cron.everyNHours") as string,
        everyDay: tScan.raw("cron.everyDay") as string,
        everyWeek: tScan.raw("cron.everyWeek") as string,
        everyMonth: tScan.raw("cron.everyMonth") as string,
        weekdays: tScan.raw("cron.weekdays") as string[],
      },
    }),
    [tColumns, tCommon, tScan]
  )

  const [targetQuery, setTargetQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
  }))
  const [targetPageTokens, setTargetPageTokens] = React.useState<Record<number, string | undefined>>({
    1: undefined,
  })
  const targetPage = targetQuery.pageIndex ?? 1
  const targetPageSize = targetQuery.pageSize
  const searchQuery = targetQuery.search ?? ""
  const typeFilter = (targetQuery.filters.type ?? []) as OrganizationTargetTypeFilter
  const pagination = React.useMemo(
    () => ({ pageIndex: targetPage - 1, pageSize: targetPageSize }),
    [targetPage, targetPageSize]
  )

  const resetTargetPaging = React.useCallback(() => {
    setTargetPageTokens({ 1: undefined })
  }, [])

  const handleTypeFilterChange = React.useCallback((value: OrganizationTargetTypeFilter) => {
    resetTargetPaging()
    setTargetQuery((current) => applyBusinessListControlChange(current, {
      filters: { ...current.filters, type: value },
    }))
  }, [resetTargetPaging])

  const typeParam = typeFilter.length === 1 ? typeFilter[0] : undefined
  const targetFilterParam = combineTargetFilterQuery(searchQuery, typeFilter)

  const unlinkTargets = useUnlinkTargetsFromOrganization()
  const { mutate: toggleScheduledScan } = useToggleScheduledScan()
  const { mutate: deleteScheduledScan } = useDeleteScheduledScan()

  const {
    data: organization,
    isLoading: isLoadingOrg,
    error: orgError,
    refetch: refetchOrganization,
  } = useOrganization(organizationNumericId)

  const {
    data: targetsData,
    isLoading: isLoadingTargets,
    isFetching: isFetchingTargets,
    isPlaceholderData: isTargetsPlaceholderData,
    error: targetsError,
    refetch: refetchTargets,
  } = useOrganizationTargets(organizationNumericId, {
    pageToken: targetQuery.pageToken,
    pageSize: targetPageSize,
    search: typeFilter.length <= 1 ? searchQuery || undefined : undefined,
    filter: targetFilterParam,
    type: typeParam,
  })

  const {
    data: summaryTargetsData,
    isLoading: isLoadingSummaryTargets,
    refetch: refetchSummaryTargets,
  } = useOrganizationTargets(
    organizationNumericId,
    {
      pageSize: 1000,
    },
    { enabled: !!organizationNumericId }
  )

  const {
    data: scheduledScansData,
    isLoading: isLoadingScheduledScans,
    isFetching: isFetchingScheduledScans,
    refetch: refetchScheduledScans,
  } = useScheduledScans({
    organizationId: organizationNumericId,
    page: 1,
    pageSize: 1000,
  })

  const nextPageToken = getCurrentCursorNextPageToken(
    targetsData?.nextPageToken,
    isTargetsPlaceholderData,
  )

  React.useEffect(() => {
    if (nextPageToken) {
      setTargetPageTokens((tokens) => ({ ...tokens, [targetPage + 1]: nextPageToken }))
    }
  }, [nextPageToken, targetPage])

  const paginationNavigation: CursorPaginationNavigation = getCursorPaginationNavigation({
    currentPage: targetPage,
    pageTokens: targetPageTokens,
    nextPageToken,
  })
  const cursorPaginationSummary: CursorPaginationSummary = {
    total: targetsData?.totalSize ?? targetsData?.total ?? 0,
  }
  const isSearching = isFetchingTargets

  const commitSearch = React.useCallback((value: string) => {
    const normalizedSearch = value.trim()
    if (normalizedSearch === searchQuery) return

    resetTargetPaging()
    setTargetQuery((current) => applyBusinessListControlChange(current, {
      search: normalizedSearch || undefined,
    }))
  }, [resetTargetPaging, searchQuery])

  const {
    isSearching: isSearchingScheduledScans,
    commitSearch: commitScheduledSearch,
  } = useSearchState({
    isFetching: isFetchingScheduledScans,
    searchValue: scheduledSearchQuery,
    setSearchValue: setScheduledSearchQuery,
    onResetPage: () => setScheduledPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const isLoading = isLoadingOrg || isLoadingTargets
  // Every value in the summary strip participates in the first visible frame.
  // Releasing its handoff before these initial queries settle can make the
  // summary grow after the skeleton has started to exit on narrow screens.
  const isInitialContentLoading = isLoading || isLoadingSummaryTargets || isLoadingScheduledScans
  const error = orgError || targetsError
  const targetRows = targetsData?.results ?? []
  const summaryTargetRows = summaryTargetsData?.results ?? targetRows
  const scheduledScans = scheduledScansData?.scheduledScans ?? EMPTY_SCHEDULED_SCANS

  const scheduledWorkflowOptions = React.useMemo(() => {
    const workflowNames = new Set<string>()
    scheduledScans.forEach((scan) => {
      if (scan.scanWorkflow) workflowNames.add(scan.scanWorkflow)
    })
    return Array.from(workflowNames).sort((a, b) => a.localeCompare(b))
  }, [scheduledScans])

  const filteredScheduledScans = React.useMemo(() => {
    const normalizedSearch = scheduledSearchQuery.trim().toLowerCase()

    return scheduledScans.filter((scan) => {
      const matchesSearch =
        !normalizedSearch ||
        scan.name.toLowerCase().includes(normalizedSearch) ||
        (scan.scanWorkflow?.toLowerCase().includes(normalizedSearch) ?? false)
      const matchesStatus =
        scheduledStatusFilter.length === 0 ||
        (scheduledStatusFilter.includes("enabled") && scan.isEnabled) ||
        (scheduledStatusFilter.includes("paused") && !scan.isEnabled)
      const matchesWorkflow =
        scheduledWorkflowFilter.length === 0 ||
        (scan.scanWorkflow ? scheduledWorkflowFilter.includes(scan.scanWorkflow) : false)

      return matchesSearch && matchesStatus && matchesWorkflow
    })
  }, [scheduledScans, scheduledSearchQuery, scheduledStatusFilter, scheduledWorkflowFilter])

  const scheduledRows = React.useMemo(() => {
    const start = scheduledPagination.pageIndex * scheduledPagination.pageSize
    return filteredScheduledScans.slice(start, start + scheduledPagination.pageSize)
  }, [filteredScheduledScans, scheduledPagination.pageIndex, scheduledPagination.pageSize])

  const scheduledPaginationInfo = buildPaginationInfo({
    total: filteredScheduledScans.length,
    page: scheduledPagination.pageIndex + 1,
    pageSize: scheduledPagination.pageSize,
    totalPages: Math.ceil(filteredScheduledScans.length / scheduledPagination.pageSize),
    minTotalPages: 1,
  })

  const targetTypeSummary = React.useMemo(
    () =>
      summaryTargetRows.reduce(
        (summary, target) => {
          summary.total += 1
          summary[target.type] += 1
          if (target.lastScannedAt) {
            const time = new Date(target.lastScannedAt).getTime()
            if (!Number.isNaN(time) && (!summary.latestScanTime || time > summary.latestScanTime)) {
              summary.latestScanTime = time
              summary.latestScanAt = target.lastScannedAt
            }
          }
          return summary
        },
        {
          total: 0,
          domain: 0,
          ip: 0,
          cidr: 0,
          latestScanAt: undefined as string | undefined,
          latestScanTime: undefined as number | undefined,
        }
      ),
    [summaryTargetRows]
  )

  const organizationSummary = React.useMemo(
    () => ({
      totalTargets:
        summaryTargetsData?.total ??
        targetsData?.total ??
        organization?.targetCount ??
        organization?.stats?.totalTargets ??
        targetTypeSummary.total,
      domains: targetTypeSummary.domain,
      ips: targetTypeSummary.ip,
      cidrs: targetTypeSummary.cidr,
      latestScanAt: targetTypeSummary.latestScanAt,
      scheduledTotal: scheduledScans.length,
      scheduledEnabled: scheduledScans.filter((scan) => scan.isEnabled).length,
    }),
    [organization, scheduledScans, summaryTargetsData?.total, targetTypeSummary, targetsData?.total]
  )

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

  const router = useRouter()
  const navigate = React.useCallback(
    (path: string) => {
      pushWithRouteProgress(router, path)
    },
    [router]
  )

  const handleDeleteTarget = React.useCallback((target: Target) => {
    setTargetToDelete(target)
    setDeleteDialogOpen(true)
  }, [])

  const confirmDelete = async () => {
    if (!targetToDelete) return

    setDeleteDialogOpen(false)
    const targetId = targetToDelete.id
    setTargetToDelete(null)

    unlinkTargets.mutate({
      organizationId: organizationNumericId,
      targetIds: [targetId],
    })
  }

  const handleBulkDelete = () => {
    if (selectedTargets.length === 0) {
      return
    }
    setBulkDeleteDialogOpen(true)
  }

  const confirmBulkDelete = async () => {
    if (selectedTargets.length === 0) return

    const targetIds = selectedTargets.map((target) => target.id)

    setBulkDeleteDialogOpen(false)
    setSelectedTargets([])

    unlinkTargets.mutate({
      organizationId: organizationNumericId,
      targetIds,
    })
  }

  const handleEditOrganization = React.useCallback(() => {
    setIsAddDialogOpen(false)
    setIsEditDialogOpen(true)
  }, [])

  const handleAddTarget = () => {
    setIsEditDialogOpen(false)
    setIsAddDialogOpen(true)
  }

  const handleAddSuccess = () => {
    setIsAddDialogOpen(false)
    refetchTargets()
    refetchSummaryTargets()
  }

  const handleEditOrganizationSuccess = () => {
    refetchOrganization()
  }

  const handlePaginationChange = React.useCallback((newPagination: { pageIndex: number; pageSize: number }) => {
    const nextPage = newPagination.pageIndex + 1
    if (newPagination.pageSize !== targetPageSize) {
      resetTargetPaging()
      setTargetQuery((current) => applyBusinessListControlChange(current, {
        pageSize: newPagination.pageSize,
      }))
      setSelectedTargets([])
      return
    }

    if (nextPage === 1) {
      resetTargetPaging()
      setTargetQuery((current) => setBusinessListPage(current, {
        pageIndex: 1,
        pageToken: undefined,
      }))
      setSelectedTargets([])
      return
    }

    const transition = getCursorPageTransition({
      currentPage: targetPage,
      pageTokens: targetPageTokens,
      nextPageToken,
      requestedPage: nextPage,
    })
    if (!transition.reachable) return

    setTargetQuery((current) => setBusinessListPage(current, {
      pageIndex: nextPage,
      pageToken: transition.pageToken,
    }))
    setSelectedTargets([])
  }, [nextPageToken, resetTargetPaging, targetPage, targetPageSize, targetPageTokens])

  const handleScheduledStatusFilterChange = (values: OrganizationScheduledStatusFilter) => {
    setScheduledStatusFilter(values)
    setScheduledPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const handleScheduledWorkflowFilterChange = (values: OrganizationScheduledWorkflowFilter) => {
    setScheduledWorkflowFilter(values)
    setScheduledPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const handleScheduledPaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    setScheduledPagination(newPagination)
  }

  const handleAddScheduledScan = React.useCallback(() => {
    void openAfterLoad("create-dialog", loadCreateScheduledScanDialog, () => {
      setIsCreateScheduledScanDialogOpen(true)
    })
  }, [openAfterLoad])

  const handleEditScheduledScan = React.useCallback((scan: ScheduledScan) => {
    setScheduledScanToEdit(scan)
    void openAfterLoad("edit-dialog", loadEditScheduledScanDialog, () => {
      setIsEditScheduledScanDialogOpen(true)
    })
  }, [openAfterLoad])

  const handleDeleteScheduledScan = React.useCallback((scan: ScheduledScan) => {
    setScheduledScanToDelete(scan)
    setScheduledScanDeleteDialogOpen(true)
  }, [])

  const confirmDeleteScheduledScan = React.useCallback(() => {
    if (!scheduledScanToDelete) return
    deleteScheduledScan(scheduledScanToDelete.id)
    setScheduledScanDeleteDialogOpen(false)
    setScheduledScanToDelete(null)
  }, [deleteScheduledScan, scheduledScanToDelete])

  const handleToggleScheduledScanStatus = React.useCallback(
    (scan: ScheduledScan, enabled: boolean) => {
      toggleScheduledScan({ id: scan.id, isEnabled: enabled })
    },
    [toggleScheduledScan]
  )

  const handleScheduledScanMutationSuccess = React.useCallback(() => {
    refetchScheduledScans()
  }, [refetchScheduledScans])

  const targetColumns = React.useMemo(
    () =>
      createTargetColumns({
        formatDate,
        navigate,
        handleDelete: handleDeleteTarget,
        t: translations,
      }),
    [formatDate, navigate, handleDeleteTarget, translations]
  )

  const scheduledColumns = React.useMemo(
    () =>
      createScheduledScanColumns({
        formatDate,
        handleEdit: handleEditScheduledScan,
        handleDelete: handleDeleteScheduledScan,
        handleToggleStatus: handleToggleScheduledScanStatus,
        t: scheduledTranslations,
      }).filter((column) => {
        const columnWithIds = column as { id?: string; accessorKey?: string }
        return columnWithIds.id !== "select" && columnWithIds.accessorKey !== "scanMode"
      }),
    [
      formatDate,
      handleEditScheduledScan,
      handleDeleteScheduledScan,
      handleToggleScheduledScanStatus,
      scheduledTranslations,
    ]
  )

  const refetch = React.useCallback(() => {
    refetchOrganization()
    refetchTargets()
    refetchSummaryTargets()
    refetchScheduledScans()
  }, [refetchOrganization, refetchTargets, refetchSummaryTargets, refetchScheduledScans])

  return {
    activeTab,
    setActiveTab,
    tColumns,
    tCommon,
    tConfirm,
    tOrg,
    tTarget,
    tScan,
    organizationId: organizationNumericId,
    organization,
    organizationSummary,
    isLoading,
    isInitialContentLoading,
    isLoadingSummaryTargets,
    error,
    refetch,
    formatDate,
    targetRows,
    targetColumns,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    isSearching,
    commitSearch,
    searchQuery,
    typeFilter,
    handleTypeFilterChange,
    selectedTargets,
    setSelectedTargets,
    isAddDialogOpen,
    setIsAddDialogOpen,
    isEditDialogOpen,
    setIsEditDialogOpen,
    handleEditOrganization,
    deleteDialogOpen,
    setDeleteDialogOpen,
    targetToDelete,
    bulkDeleteDialogOpen,
    setBulkDeleteDialogOpen,
    handleDeleteTarget,
    confirmDelete,
    handleBulkDelete,
    confirmBulkDelete,
    handleAddTarget,
    handleAddSuccess,
    handleEditOrganizationSuccess,
    scheduledRows,
    scheduledScans,
    scheduledColumns,
    scheduledPagination,
    scheduledPaginationInfo,
    isLoadingScheduledScans,
    scheduledSearchQuery,
    isSearchingScheduledScans,
    commitScheduledSearch,
    handleScheduledPaginationChange,
    scheduledStatusFilter,
    handleScheduledStatusFilterChange,
    scheduledWorkflowFilter,
    handleScheduledWorkflowFilterChange,
    scheduledWorkflowOptions,
    pendingInteraction,
    cancelPendingInteraction: cancelPending,
    isCreateScheduledScanDialogOpen,
    setIsCreateScheduledScanDialogOpen,
    isEditScheduledScanDialogOpen,
    setIsEditScheduledScanDialogOpen,
    scheduledScanToEdit,
    scheduledScanDeleteDialogOpen,
    setScheduledScanDeleteDialogOpen,
    scheduledScanToDelete,
    handleAddScheduledScan,
    confirmDeleteScheduledScan,
    handleScheduledScanMutationSuccess,
  }
}

export type OrganizationDetailViewState = ReturnType<typeof useOrganizationDetailViewState>
