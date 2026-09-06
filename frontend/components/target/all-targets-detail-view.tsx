"use client"

import {
  AllTargetsDetailViewDialogs,
  AllTargetsDetailViewLoadingState,
  AllTargetsDetailViewTable,
} from "./all-targets-detail-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useAllTargetsDetailViewState } from "./all-targets-detail-view-state"

/**
 * All targets detail view component
 * Displays a list of all targets in the system, supports search, pagination, delete operations
 */
interface AllTargetsDetailViewProps {
  className?: string
  tableClassName?: string
  hideToolbar?: boolean
  hidePagination?: boolean
}

export function AllTargetsDetailView({
  className,
  tableClassName,
  hideToolbar,
  hidePagination,
}: AllTargetsDetailViewProps) {
  const state = useAllTargetsDetailViewState({
    className,
    tableClassName,
    hideToolbar,
    hidePagination,
  })
  const isInitialLoading = state.isLoading && !state.data
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  if (!isInitialLoading && state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        onRetry={state.refetch}
        actionHref="/overview/"
      />
    )
  }

  return (
    <ContentHandoff
      owner="all-targets-detail-view-content"
      layer="workspace"
      isLoading={isInitialLoading}
      skeleton={<AllTargetsDetailViewLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <AllTargetsDetailViewTable state={state} stableSurfaceRowCount={loadingRowCount} />
      <AllTargetsDetailViewDialogs state={state} />
    </ContentHandoff>
  )
}
