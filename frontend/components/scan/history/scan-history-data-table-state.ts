import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/ui/data-table/use-simple-search"

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

  const {
    value: localSearchValue,
    setValue: setLocalSearchValue,
    submit: handleSearchSubmit,
  } = useSimpleSearchState({ searchValue, onSearch })

  const statusOptions: { value: ScanStatus | "all"; label: string }[] = [
    { value: "all", label: tScan("allStatus") },
    { value: "running", label: t("running") },
    { value: "completed", label: t("completed") },
    { value: "failed", label: t("failed") },
    { value: "pending", label: t("pending") },
    { value: "cancelled", label: t("cancelled") },
  ]

  return {
    t,
    tScan,
    tActions,
    localSearchValue,
    setLocalSearchValue,
    handleSearchSubmit,
    statusOptions,
  }
}

export type ScanHistoryDataTableState = ReturnType<typeof useScanHistoryDataTableState>
