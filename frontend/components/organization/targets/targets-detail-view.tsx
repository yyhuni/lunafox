"use client"

import {
  TargetsDetailViewDialogs,
  TargetsDetailViewEmptyState,
  TargetsDetailViewErrorState,
  TargetsDetailViewLoadingState,
  TargetsDetailViewTable,
} from "./targets-detail-view-sections"
import { useTargetsDetailViewState } from "./targets-detail-view-state"

/**
 * Organization goal details view component (using React Query)
 * Used to display and manage target lists under an organization
 * Supports obtaining data by organization ID
 */
export function OrganizationTargetsDetailView({
  organizationId,
}: {
  organizationId: string
}) {
  const state = useTargetsDetailViewState({ organizationId })

  if (state.error) {
    return (
      <TargetsDetailViewErrorState
        error={state.error}
        onRetry={state.refetch}
        tCommon={state.tCommon}
      />
    )
  }

  if (state.isLoading) {
    return <TargetsDetailViewLoadingState />
  }

  if (!state.organization) {
    return <TargetsDetailViewEmptyState tOrg={state.tOrg} />
  }

  return (
    <>
      <TargetsDetailViewTable state={state} />
      <TargetsDetailViewDialogs state={state} />
    </>
  )
}

export { OrganizationTargetsDetailView as TargetsDetailView }
