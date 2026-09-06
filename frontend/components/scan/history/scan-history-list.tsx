"use client"

import {
  ScanHistoryListDialogsSection,
  ScanHistoryListLoadingState,
  ScanHistoryListTable,
} from "./scan-history-list-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useScanHistoryListViewState } from "./scan-history-list-view-state"

import type { LoadingLayer } from "@/components/shared/loading/loading-owner"

interface ScanHistoryListProps {
  hideToolbar?: boolean
  targetId?: number
  pageSize?: number
  hideTargetColumn?: boolean
  pageSizeOptions?: number[]
  hidePagination?: boolean
  layer?: LoadingLayer
}

export function ScanHistoryList({
  hideToolbar = false,
  targetId,
  pageSize: customPageSize,
  hideTargetColumn = false,
  pageSizeOptions,
  hidePagination = false,
  layer = "workspace",
}: ScanHistoryListProps) {
  const state = useScanHistoryListViewState({
    hideToolbar,
    targetId,
    pageSize: customPageSize,
    hideTargetColumn,
    pageSizeOptions,
    hidePagination,
  })
  const isInitialLoading = state.isLoading && !state.scans.length
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  if (!isInitialLoading && state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        onRetry={state.refetch}
        actionHref="/scan/history/"
      />
    )
  }

  return (
    <ContentHandoff
      owner="scan-history-list-view-content"
      layer={layer}
      isLoading={isInitialLoading}
      skeleton={
        <ScanHistoryListLoadingState
          hideToolbar={state.hideToolbar}
          hidePagination={state.hidePagination}
          rowCount={loadingRowCount}
          hideTargetColumn={hideTargetColumn}
        />
      }
    >
      <ScanHistoryListTable state={state} stableSurfaceRowCount={loadingRowCount} />
      <ScanHistoryListDialogsSection state={state} />
    </ContentHandoff>
  )
}
