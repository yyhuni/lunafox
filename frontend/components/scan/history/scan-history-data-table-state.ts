import { useTranslations } from "next-intl"

import type { DataTableFacetedFilterOption } from "@/components/shared/data-table/faceted-filter"
import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"
import { useScanStatistics } from "@/hooks/use-scans"

import type { ScanStatus } from "@/types/scan.types"

interface ScanHistoryDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
}

export function useScanHistoryDataTableState({
  searchValue,
  onSearch,
}: ScanHistoryDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tScan = useTranslations("scan.history")
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const { data: scanStatistics } = useScanStatistics()

  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  const statusOptions: Array<DataTableFacetedFilterOption<ScanStatus>> = [
    { value: "running", label: t("running"), count: scanStatistics?.running ?? 0 },
    { value: "succeeded", label: t("succeeded"), count: scanStatistics?.succeeded ?? 0 },
    { value: "failed", label: t("failed"), count: scanStatistics?.failed ?? 0 },
    { value: "pending", label: t("pending"), count: scanStatistics?.pending ?? 0 },
    { value: "cancelled", label: t("cancelled"), count: scanStatistics?.cancelled ?? 0 },
  ]

  return {
    t,
    tScan,
    tActions,
    tDataTable,
    localSearchValue,
    handleSearchInputChange,
    commitSearch,
    statusOptions,
  }
}

export type ScanHistoryDataTableState = ReturnType<typeof useScanHistoryDataTableState>
