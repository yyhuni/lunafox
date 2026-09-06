import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"

interface UseOrganizationDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
}

export function useOrganizationDataTableState({
  searchValue,
  onSearch,
}: UseOrganizationDataTableStateOptions) {
  const t = useTranslations("organization")
  const tActions = useTranslations("common.actions")
  const tDataTable = useTranslations("dataTable")
  const tTooltips = useTranslations("tooltips")

  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  return {
    t,
    tActions,
    tDataTable,
    tTooltips,
    localSearchValue,
    handleSearchInputChange,
    commitSearch,
  }
}
