import * as React from "react"
import { useTranslations } from "next-intl"

import type { ScanEngine } from "@/types/engine.types"

interface UseEngineDataTableStateOptions {
  data: ScanEngine[]
}

export function useEngineDataTableState({ data }: UseEngineDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tEngine = useTranslations("scan.engine")

  const [searchValue, setSearchValue] = React.useState("")

  const filteredData = React.useMemo(() => {
    if (!searchValue) return data
    return data.filter((item) => {
      const name = item.name || ""
      return name.toLowerCase().includes(searchValue.toLowerCase())
    })
  }, [data, searchValue])

  return {
    t,
    tEngine,
    searchValue,
    setSearchValue,
    filteredData,
  }
}
