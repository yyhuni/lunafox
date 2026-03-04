"use client"

import { DashboardDataDialogs, DashboardScanTable } from "./dashboard-data-table-sections"
import { useDashboardDataTableState } from "./dashboard-data-table-state"

export function DashboardDataTable() {
  const state = useDashboardDataTableState()

  return (
    <>
      <DashboardDataDialogs state={state} />
      <DashboardScanTable state={state} />
    </>
  )
}
