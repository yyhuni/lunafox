import * as React from "react"
import { useTranslations } from "next-intl"

interface UseCommandsDataTableStateOptions<TData extends { id: number; displayName?: string }> {
  data: TData[]
  onBulkDelete?: (selectedIds: number[]) => void
}

export function useCommandsDataTableState<TData extends { id: number; displayName?: string }>({
  data,
  onBulkDelete,
}: UseCommandsDataTableStateOptions<TData>) {
  const t = useTranslations("tools.commands")
  const tCommon = useTranslations("common")

  const [searchValue, setSearchValue] = React.useState("")
  const [selectedRows, setSelectedRows] = React.useState<TData[]>([])

  const filteredData = React.useMemo(() => {
    if (!searchValue) return data
    return data.filter((item) => {
      const displayName = item.displayName || ""
      return displayName.toLowerCase().includes(searchValue.toLowerCase())
    })
  }, [data, searchValue])

  const handleBulkDelete = () => {
    if (onBulkDelete && selectedRows.length > 0) {
      onBulkDelete(selectedRows.map((row) => row.id))
    }
  }

  return {
    t,
    tCommon,
    searchValue,
    setSearchValue,
    selectedRows,
    setSelectedRows,
    filteredData,
    handleBulkDelete,
  }
}
