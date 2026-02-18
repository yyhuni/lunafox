import { useTranslations } from "next-intl"

import { useSimpleSearchState } from "@/components/ui/data-table/use-simple-search"

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

  const {
    value: localSearchValue,
    setValue: setLocalSearchValue,
    submit: handleSearchSubmit,
  } = useSimpleSearchState({ searchValue, onSearch })

  const defaultSorting = [{ id: "createdAt", desc: true }]

  return {
    t,
    tActions,
    localSearchValue,
    setLocalSearchValue,
    handleSearchSubmit,
    defaultSorting,
  }
}
