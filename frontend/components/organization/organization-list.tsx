"use client"

import {
  OrganizationListDialogs,
  OrganizationListErrorState,
  OrganizationListSkeleton,
  OrganizationListTable,
} from "./organization-list-sections"
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

  if (state.error) {
    return (
      <OrganizationListErrorState
        error={state.error}
        onRetry={state.refetch}
        tCommon={state.tCommon}
      />
    )
  }

  if (state.isLoading || !state.data) {
    return <OrganizationListSkeleton />
  }

  return (
    <div className="space-y-4">
      <OrganizationListTable state={state} />
      <OrganizationListDialogs state={state} />
    </div>
  )
}
