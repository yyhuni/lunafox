"use client"

import {
  TargetsDetailViewDialogs,
  TargetsDetailViewEmptyState,
  TargetsDetailViewLoadingState,
  TargetsDetailViewTable,
} from "./targets-detail-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { normalizeError } from "@/lib/errors/normalize-error"
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
  const isInitialLoading = state.isLoading && !state.organization

  if (!isInitialLoading && state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        onRetry={state.refetch}
        variant="section"
      />
    )
  }

  if (!isInitialLoading && !state.organization) {
    return <TargetsDetailViewEmptyState tOrg={state.tOrg} />
  }

  return (
    <ContentHandoff
      owner="organization-targets-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<TargetsDetailViewLoadingState state={state} />}
    >
      {state.organization ? (
        <>
          <TargetsDetailViewTable state={state} />
          <TargetsDetailViewDialogs state={state} />
        </>
      ) : null}
    </ContentHandoff>
  )
}

export { OrganizationTargetsDetailView as TargetsDetailView }
