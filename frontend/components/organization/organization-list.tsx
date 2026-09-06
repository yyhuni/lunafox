"use client"

import {
  OrganizationListDialogs,
  OrganizationListLoadingState,
  OrganizationListTable,
} from "./organization-list-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useOrganizationListState } from "./organization-list-state"

/**
 * Organize list component (using React Query)
 * 
 * Features:
 * 1. Unified Loading state management
 * 2. Automatic caching and revalidation
 * 3. Optimistic updates
 * 4. Automatic error handling
 * 5. Better user experience
 */
export function OrganizationList() {
  const state = useOrganizationListState()
  const isInitialLoading = state.isLoading || !state.data
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  if (state.error) {
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
      owner="organization-list-content"
      layer="workspace"
      isLoading={isInitialLoading}
      skeleton={<OrganizationListLoadingState state={state} rowCount={loadingRowCount} />}
      prepareContentBeforeHandoff
    >
      <OrganizationListTable state={state} stableSurfaceRowCount={loadingRowCount} />
      <OrganizationListDialogs state={state} />
    </ContentHandoff>
  )
}
