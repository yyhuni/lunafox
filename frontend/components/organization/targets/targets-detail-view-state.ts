import React from "react"
import { useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"

import { useOrganization, useOrganizationTargets, useUnlinkTargetsFromOrganization } from "@/hooks/use-organizations"
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
import { createTargetColumns } from "./targets-columns"

import type { Target } from "@/types/target.types"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"

interface TargetsDetailViewStateOptions {
  organizationId: string
}

export function useTargetsDetailViewState({ organizationId }: TargetsDetailViewStateOptions) {
  const [selectedTargets, setSelectedTargets] = React.useState<Target[]>([])
  const [isAddDialogOpen, setIsAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [targetToDelete, setTargetToDelete] = React.useState<Target | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)

  const tColumns = useTranslations("columns")
  const tCommon = useTranslations("common")
  const tTooltips = useTranslations("tooltips")
  const tTarget = useTranslations("target")
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

  const [targetQuery, setTargetQuery] = React.useState(() => createBusinessListQuery({
    pageSize: 10,
  }))
  const [targetPageTokens, setTargetPageTokens] = React.useState<Record<number, string | undefined>>({
    1: undefined,
  })
  const targetPage = targetQuery.pageIndex ?? 1
  const targetPageSize = targetQuery.pageSize
  const pagination = React.useMemo(
    () => ({ pageIndex: targetPage - 1, pageSize: targetPageSize }),
    [targetPage, targetPageSize]
  )

  const resetTargetPaging = React.useCallback(() => {
    setTargetPageTokens({ 1: undefined })
  }, [])

  const unlinkTargets = useUnlinkTargetsFromOrganization()

  const {
    data: organization,
    isLoading: isLoadingOrg,
    error: orgError,
  } = useOrganization(parseInt(organizationId))

  const {
    data: targetsData,
    isLoading: isLoadingTargets,
    isPlaceholderData: isTargetsPlaceholderData,
    error: targetsError,
    refetch,
  } = useOrganizationTargets(parseInt(organizationId), {
    pageToken: targetQuery.pageToken,
    pageSize: targetPageSize,
  })

  const isLoading = isLoadingOrg || isLoadingTargets
  const error = orgError || targetsError
  const targetRows = targetsData?.results ?? []

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
      organizationId: parseInt(organizationId),
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
      organizationId: parseInt(organizationId),
      targetIds,
    })
  }

  const handleAddTarget = () => {
    setIsAddDialogOpen(true)
  }

  const handleAddSuccess = () => {
    setIsAddDialogOpen(false)
    refetch()
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

  return {
    tColumns,
    tCommon,
    tConfirm,
    tOrg,
    tTarget,
    organization,
    isLoading,
    error,
    refetch,
    targetRows,
    targetColumns,
    pagination,
    cursorPaginationSummary,
    paginationNavigation,
    handlePaginationChange,
    selectedTargets,
    setSelectedTargets,
    isAddDialogOpen,
    setIsAddDialogOpen,
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
  }
}

export type TargetsDetailViewState = ReturnType<typeof useTargetsDetailViewState>
