"use client"

import { DashboardDataDialogs, DashboardDataTabs } from "./dashboard-data-table-sections"
import { useDashboardDataTableState } from "./dashboard-data-table-state"

export function DashboardDataTable() {
  const state = useDashboardDataTableState()

  return (
    <>
      <DashboardDataDialogs state={state} />
      <DashboardDataTabs state={state} />
    </>
  )
}
