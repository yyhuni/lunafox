import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"
import type {
  CursorPaginationNavigation,
  CursorPaginationSummary,
} from "@/types/data-table.types"

interface UseScheduledScanDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
  page: number
  pageSize: number
  total: number
  totalPages?: number
  cursorPaginationSummary?: CursorPaginationSummary
  paginationNavigation?: CursorPaginationNavigation
}

export function useScheduledScanDataTableState({
  searchValue,
  onSearch,
  page,
  pageSize,
  total,
  totalPages = 1,
  cursorPaginationSummary,
  paginationNavigation,
}: UseScheduledScanDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tCommon = useTranslations("common")
  const tConfirm = useTranslations("common.confirm")
  const tScan = useTranslations("scan.scheduled")

  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  const pagination = { pageIndex: page - 1, pageSize }
  const paginationInfo = paginationNavigation?.mode === "cursor"
    ? undefined
    : buildPaginationInfo({
        total,
        page,
        pageSize,
        totalPages,
        minTotalPages: 1,
      })

  return {
    t,
    tCommon,
    tConfirm,
    tScan,
    localSearchValue,
    handleSearchInputChange,
    commitSearch,
    pagination,
    paginationInfo,
    cursorPaginationSummary,
    paginationNavigation,
  }
}
