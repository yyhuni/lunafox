import React from "react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"

import { createAllTargetsColumns } from "@/components/target/all-targets-columns"
import { useTargets, useDeleteTarget, useBatchDeleteTargets } from "@/hooks/use-targets"
import { useSearchState } from "@/hooks/_shared/use-search-state"
import { formatDate } from "@/lib/utils"

import type { Target } from "@/types/target.types"
import type { AllTargetsTranslations } from "@/components/target/all-targets-columns"

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

  const translations: AllTargetsTranslations = React.useMemo(
    () => ({
      columns: {
        target: tColumns("target.target"),
        organization: tColumns("organization.organization"),
        addedOn: tColumns("target.addedOn"),
        lastScanned: tColumns("target.lastScanned"),
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
    }),
    [tColumns, tCommon, tTooltips]
  )

  const [pagination, setPagination] = React.useState({ pageIndex: 0, pageSize: 10 })
  const [searchQuery, setSearchQuery] = React.useState("")
  const [typeFilter, setTypeFilter] = React.useState("")
  const [selectedTargets, setSelectedTargets] = React.useState<Target[]>([])
  const [isAddDialogOpen, setIsAddDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [targetToDelete, setTargetToDelete] = React.useState<Target | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [shouldPrefetchOrgs, setShouldPrefetchOrgs] = React.useState(false)
  const [initiateScanDialogOpen, setInitiateScanDialogOpen] = React.useState(false)
  const [scheduleScanDialogOpen, setScheduleScanDialogOpen] = React.useState(false)
  const [targetToScan, setTargetToScan] = React.useState<Target | null>(null)
  const [targetToSchedule, setTargetToSchedule] = React.useState<Target | null>(null)

  const handlePaginationChange = React.useCallback(
    (newPagination: { pageIndex: number; pageSize: number }) => {
      setPagination(newPagination)
    },
    []
  )

  const handleTypeFilterChange = (value: string) => {
    setTypeFilter(value)
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const { data, isLoading, isFetching, error } = useTargets(
    pagination.pageIndex + 1,
    pagination.pageSize,
    typeFilter || undefined,
    searchQuery || undefined
  )
  const deleteTargetMutation = useDeleteTarget()
  const batchDeleteMutation = useBatchDeleteTargets()
  const { isSearching, handleSearchChange } = useSearchState({
    isFetching,
    setSearchValue: setSearchQuery,
    onResetPage: () => setPagination((prev) => ({ ...prev, pageIndex: 0 })),
  })

  const targets = data?.results || []
  const totalCount = data?.total || 0

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
      router.push(path)
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
    data,
    isLoading,
    error,
    targets,
    totalCount,
    columns,
    pagination,
    handlePaginationChange,
    searchQuery,
    isSearching,
    handleSearchChange,
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
    handleBatchDelete,
    confirmDelete,
    confirmBulkDelete,
  }
}

export type AllTargetsDetailViewState = ReturnType<typeof useAllTargetsDetailViewState>
