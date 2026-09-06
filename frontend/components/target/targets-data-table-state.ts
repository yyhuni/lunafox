import * as React from "react"
import { useTranslations } from "next-intl"
import type { SortingState } from "@tanstack/react-table"

import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import type { CursorPaginationSummary } from "@/types/data-table.types"

interface UseTargetTargetsDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
  externalPagination?: { pageIndex: number; pageSize: number }
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  manualPagination: boolean
  totalCount?: number
  sorting?: SortingState
  onSortingChange?: (sorting: SortingState) => void
}

export function useTargetTargetsDataTableState({
  searchValue,
  onSearch,
  externalPagination,
  onPaginationChange,
  manualPagination,
  totalCount,
  sorting = [],
  onSortingChange,
}: UseTargetTargetsDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tColumns = useTranslations("columns")
  const tDataTable = useTranslations("dataTable")
  const tTarget = useTranslations("target")
  const tTooltips = useTranslations("tooltips")

  const [internalPagination, setInternalPagination] = React.useState<{
    pageIndex: number
    pageSize: number
  }>({
    pageIndex: 0,
    pageSize: 10,
  })

  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  const pagination = externalPagination || internalPagination

  const handlePaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    if (onPaginationChange) {
      onPaginationChange(newPagination)
    } else {
      setInternalPagination(newPagination)
    }
  }

  const cursorPaginationSummary: CursorPaginationSummary | undefined =
    manualPagination && totalCount !== undefined
      ? { total: totalCount ?? 0 }
      : undefined

  return {
    t,
    tActions,
    tColumns,
    tDataTable,
    tTarget,
    tTooltips,
    localSearchValue,
    handleSearchInputChange,
    commitSearch,
    pagination,
    setInternalPagination,
    handlePaginationChange,
    cursorPaginationSummary,
    sorting,
    onSortingChange,
  }
}
