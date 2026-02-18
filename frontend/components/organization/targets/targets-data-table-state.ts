import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/ui/data-table/use-simple-search"

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

  const {
    value: localSearchValue,
    setValue: setLocalSearchValue,
    submit: handleSearchSubmit,
  } = useSimpleSearchState({ searchValue, onSearch })

  return {
    t,
    tTarget,
    tTooltips,
    tCommon,
    localSearchValue,
    setLocalSearchValue,
    handleSearchSubmit,
  }
}
