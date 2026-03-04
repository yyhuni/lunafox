import * as React from "react"
import { useTranslations } from "next-intl"

import type { ScanWorkflow } from "@/types/workflow.types"

interface UseWorkflowDataTableStateOptions {
  data: ScanWorkflow[]
}

export function useWorkflowDataTableState({ data }: UseWorkflowDataTableStateOptions) {
  const t = useTranslations("common.status")
  const tWorkflow = useTranslations("scan.workflow")

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
    tWorkflow,
    searchValue,
    setSearchValue,
    filteredData,
  }
}
