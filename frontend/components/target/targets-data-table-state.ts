import * as React from "react"
import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/ui/data-table/use-simple-search"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"

interface UseTargetTargetsDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
  externalPagination?: { pageIndex: number; pageSize: number }
  onPaginationChange?: (pagination: { pageIndex: number; pageSize: number }) => void
  manualPagination: boolean
  totalCount?: number
}

export function useTargetTargetsDataTableState({
  searchValue,
  onSearch,
  externalPagination,
  onPaginationChange,
  manualPagination,
  totalCount,
}: UseTargetTargetsDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tActions = useTranslations("common.actions")
  const tTarget = useTranslations("target")

  const [internalPagination, setInternalPagination] = React.useState<{
    pageIndex: number
    pageSize: number
  }>({
    pageIndex: 0,
    pageSize: 10,
  })

  const {
    value: localSearchValue,
    setValue: setLocalSearchValue,
    submit: handleSearchSubmit,
  } = useSimpleSearchState({ searchValue, onSearch })

  const pagination = externalPagination || internalPagination

  const handlePaginationChange = (newPagination: { pageIndex: number; pageSize: number }) => {
    if (onPaginationChange) {
      onPaginationChange(newPagination)
    } else {
      setInternalPagination(newPagination)
    }
  }

  const paginationInfo =
    manualPagination && totalCount !== undefined
      ? buildPaginationInfo({
          total: totalCount ?? 0,
          page: pagination.pageIndex + 1,
          pageSize: pagination.pageSize,
          minTotalPages: 1,
        })
      : undefined

  return {
    t,
    tActions,
    tTarget,
    localSearchValue,
    setLocalSearchValue,
    handleSearchSubmit,
    pagination,
    setInternalPagination,
    handlePaginationChange,
    paginationInfo,
  }
}
