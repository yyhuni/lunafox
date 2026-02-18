import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/ui/data-table/use-simple-search"
import { buildPaginationInfo } from "@/hooks/_shared/pagination"

interface UseScheduledScanDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export function useScheduledScanDataTableState({
  searchValue,
  onSearch,
  page,
  pageSize,
  total,
  totalPages,
}: UseScheduledScanDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tScan = useTranslations("scan.scheduled")

  const {
    value: localSearchValue,
    setValue: setLocalSearchValue,
    submit: handleSearchSubmit,
  } = useSimpleSearchState({ searchValue, onSearch })

  const pagination = { pageIndex: page - 1, pageSize }
  const paginationInfo = buildPaginationInfo({
    total,
    page,
    pageSize,
    totalPages,
    minTotalPages: 1,
  })

  return {
    t,
    tScan,
    localSearchValue,
    setLocalSearchValue,
    handleSearchSubmit,
    pagination,
    paginationInfo,
  }
}
