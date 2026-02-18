"use client"

import {
  ScanHistoryListDialogsSection,
  ScanHistoryListErrorState,
  ScanHistoryListLoadingState,
  ScanHistoryListTable,
} from "./scan-history-list-sections"
import { useScanHistoryListViewState } from "./scan-history-list-view-state"

interface ScanHistoryListProps {
  hideToolbar?: boolean
  targetId?: number
  pageSize?: number
  hideTargetColumn?: boolean
  pageSizeOptions?: number[]
  hidePagination?: boolean
}

export function ScanHistoryList({
  hideToolbar = false,
  targetId,
  pageSize: customPageSize,
  hideTargetColumn = false,
  pageSizeOptions,
  hidePagination = false,
}: ScanHistoryListProps) {
  const state = useScanHistoryListViewState({
    hideToolbar,
    targetId,
    pageSize: customPageSize,
    hideTargetColumn,
    pageSizeOptions,
    hidePagination,
  })

  if (state.error) {
    return <ScanHistoryListErrorState state={state} />
  }

  if (state.isLoading) {
    return <ScanHistoryListLoadingState />
  }

  return (
    <>
      <ScanHistoryListTable state={state} />
      <ScanHistoryListDialogsSection state={state} />
    </>
  )
}
