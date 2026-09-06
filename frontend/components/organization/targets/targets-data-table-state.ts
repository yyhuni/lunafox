import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/shared/data-table/use-simple-search"

interface UseOrganizationTargetsDataTableStateOptions {
  searchValue?: string
  onSearch?: (value: string) => void
}

export function useOrganizationTargetsDataTableState({
  searchValue,
  onSearch,
}: UseOrganizationTargetsDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tTarget = useTranslations("target")
  const tTooltips = useTranslations("tooltips")
  const tCommon = useTranslations("common")
  const tColumns = useTranslations("columns")
  const tDataTable = useTranslations("dataTable")

  const {
    value: localSearchValue,
    handleSearchInputChange,
    commitNow: commitSearch,
  } = useSimpleSearchState({ searchValue, onSearch })

  return {
    t,
    tTarget,
    tTooltips,
    tCommon,
    tColumns,
    tDataTable,
    localSearchValue,
    handleSearchInputChange,
    commitSearch,
  }
}
