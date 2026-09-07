"use client"

import { OverviewDataDialogs, OverviewScanTable } from "./overview-data-table-sections"
import { useOverviewDataTableState } from "./overview-data-table-state"

export function OverviewDataTable() {
  const state = useOverviewDataTableState()

  return (
    <>
      <OverviewDataDialogs state={state} />
      <OverviewScanTable state={state} />
    </>
  )
}
