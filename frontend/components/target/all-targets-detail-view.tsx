"use client"

import {
  AllTargetsDetailViewDialogs,
  AllTargetsDetailViewErrorState,
  AllTargetsDetailViewLoadingState,
  AllTargetsDetailViewTable,
} from "./all-targets-detail-view-sections"
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

  if (state.isLoading) {
    return <AllTargetsDetailViewLoadingState />
  }

  if (state.error) {
    return <AllTargetsDetailViewErrorState error={state.error} tCommon={state.tCommon} />
  }

  return (
    <>
      <AllTargetsDetailViewTable state={state} />
      <AllTargetsDetailViewDialogs state={state} />
    </>
  )
}
