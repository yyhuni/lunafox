"use client"

import {
  OrganizationDetailViewDialogs,
  OrganizationDetailViewEmptyState,
  OrganizationDetailViewErrorState,
  OrganizationDetailViewLoadingState,
  OrganizationDetailViewTable,
} from "./organization-detail-view-sections"
import { useOrganizationDetailViewState } from "./organization-detail-view-state"

/**
 * Organization detail view component
 * Displays organization statistics and target list
 */
export function OrganizationDetailView({
  organizationId,
}: {
  organizationId: string
}) {
  const state = useOrganizationDetailViewState({ organizationId })

  if (state.error) {
    return (
      <OrganizationDetailViewErrorState
        error={state.error}
        onRetry={state.refetch}
        tCommon={state.tCommon}
      />
    )
  }

  if (state.isLoading) {
    return <OrganizationDetailViewLoadingState />
  }

  if (!state.organization) {
    return <OrganizationDetailViewEmptyState tOrg={state.tOrg} />
  }

  return (
    <>
      <OrganizationDetailViewTable state={state} />
      <OrganizationDetailViewDialogs state={state} />
    </>
  )
}
