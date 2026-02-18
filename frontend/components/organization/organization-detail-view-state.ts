import React from "react"
import { useRouter } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"

import { useOrganization, useOrganizationTargets, useUnlinkTargetsFromOrganization } from "@/hooks/use-organizations"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { buildPaginationInfo, normalizePagination } from "@/hooks/_shared/pagination"
import { getDateLocale } from "@/lib/date-utils"
import { createTargetColumns } from "./targets/targets-columns"

import type { Target } from "@/types/target.types"

interface OrganizationDetailViewStateOptions {
  organizationId: string
}

export function useOrganizationDetailViewState({ organizationId }: OrganizationDetailViewStateOptions) {
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

  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  })

  const [searchQuery, setSearchQuery] = React.useState("")
  const [typeFilter, setTypeFilter] = React.useState<string>("")

  const handleTypeFilterChange = (value: string) => {
    setTypeFilter(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const unlinkTargets = useUnlinkTargetsFromOrganization()

  const {
    data: organization,
    isLoading: isLoadingOrg,
    error: orgError,
  } = useOrganization(parseInt(organizationId))

  const {
    data: targetsData,
    isLoading: isLoadingTargets,
    isFetching: isFetchingTargets,
    error: targetsError,
    refetch,
  } = useOrganizationTargets(parseInt(organizationId), {
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    search: searchQuery || undefined,
    type: typeFilter || undefined,
  })

  const { isSearching, handleSearchChange } = useSearchState({
    isFetching: isFetchingTargets,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const paginationInfo = targetsData
    ? buildPaginationInfo({
        ...normalizePagination(
          targetsData,
          pagination.pageIndex + 1,
          pagination.pageSize
        ),
        minTotalPages: 1,
      })
    : undefined

  const isLoading = isLoadingOrg || isLoadingTargets
  const error = orgError || targetsError
  const targetRows = targetsData?.results ?? []

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
      router.push(path)
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

  const handlePaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    setPagination(newPagination)
    setSelectedTargets([])
  }

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
    setPagination,
    paginationInfo,
    handlePaginationChange,
    isSearching,
    handleSearchChange,
    searchQuery,
    typeFilter,
    handleTypeFilterChange,
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

export type OrganizationDetailViewState = ReturnType<typeof useOrganizationDetailViewState>
