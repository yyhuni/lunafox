"use client"

import { type ReactNode } from "react"
import {
  OrganizationDetailActionPanel,
  OrganizationDetailViewDialogs,
  OrganizationDetailViewLoadingState,
  OrganizationDetailViewTable,
  OrganizationDetailViewWorkbench,
} from "./organization-detail-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { DetailDrawer } from "@/components/shared/detail-drawer"
import { createAppError } from "@/lib/errors/app-error"
import { normalizeError } from "@/lib/errors/normalize-error"
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
  const isInitialLoading = state.isInitialContentLoading && !state.error

  if (!isInitialLoading && state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error)}
        onRetry={state.refetch}
        actionHref="/organizations/"
      />
    )
  }

  if (!isInitialLoading && !state.organization) {
    return (
      <AppErrorState
        error={createAppError("resource-not-found", { retryable: false })}
        title={state.tOrg("notFound")}
        description={state.tOrg("notFoundDesc", { id: organizationId })}
        resourceLabel={state.tOrg("detail.breadcrumb")}
        actionHref="/organizations/"
      />
    )
  }

  return (
    <ContentHandoff
      owner="organization-detail-view-content"
      layer="workspace"
      isLoading={isInitialLoading}
      skeleton={<OrganizationDetailViewLoadingState state={state} />}
      prepareContentBeforeHandoff
    >
      {state.organization ? (
        <>
          <OrganizationDetailViewWorkbench state={state} />
          <OrganizationDetailViewDialogs state={state} />
        </>
      ) : null}
    </ContentHandoff>
  )
}

export function OrganizationDetailDrawerView({
  organizationId,
  open,
  onOpenChange,
  title,
  description,
  previewDescription,
}: {
  organizationId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  previewDescription?: string
}) {
  if (!organizationId) {
    return (
      <DetailDrawer
        open={open}
        onOpenChange={onOpenChange}
        title={title}
        description={description}
      >
        {null}
      </DetailDrawer>
    )
  }

  return (
    <OrganizationDetailDrawerViewLoaded
      organizationId={organizationId}
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      previewDescription={previewDescription}
    />
  )
}

function OrganizationDetailDrawerViewLoaded({
  organizationId,
  open,
  onOpenChange,
  title,
  description,
  previewDescription,
}: {
  organizationId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  previewDescription?: string
}) {
  const state = useOrganizationDetailViewState({ organizationId })
  const isInitialLoading = state.isInitialContentLoading && !state.error
  const sidecar =
    state.isEditDialogOpen || state.isAddDialogOpen ? (
      <OrganizationDetailActionPanel state={state} />
    ) : null
  const handleSidecarClose = () => {
    state.setIsEditDialogOpen(false)
    state.setIsAddDialogOpen(false)
  }

  return (
    <DetailDrawer
      open={open}
      onOpenChange={onOpenChange}
      title={title}
      description={description}
      sidecar={sidecar}
      onSidecarClose={handleSidecarClose}
    >
      <div className="min-h-0 flex-1 overflow-y-auto py-5">
        <ContentHandoff
            owner="organization-detail-drawer-content"
            isLoading={isInitialLoading}
            skeleton={(
              <OrganizationDetailViewLoadingState
                state={state}
                previewName={typeof description === "string" ? description : undefined}
                previewDescription={previewDescription}
                summarySurface="drawer"
              />
            )}
            className="min-h-0 flex-1"
            skeletonClassName="min-h-0 flex-1"
            contentClassName="min-h-0 flex-1"
          >
            {!isInitialLoading && state.error ? (
              <AppErrorState
                error={normalizeError(state.error)}
                onRetry={state.refetch}
                actionHref="/organizations/"
              />
            ) : null}

            {!isInitialLoading && !state.error && !state.organization ? (
              <AppErrorState
                error={createAppError("resource-not-found", { retryable: false })}
                title={state.tOrg("notFound")}
                description={state.tOrg("notFoundDesc", { id: organizationId })}
                resourceLabel={state.tOrg("detail.breadcrumb")}
                actionHref="/organizations/"
              />
            ) : null}

            {state.organization ? (
              <>
                <OrganizationDetailViewTable state={state} summarySurface="drawer" />
                <OrganizationDetailViewDialogs state={state} />
              </>
            ) : null}
        </ContentHandoff>
      </div>
    </DetailDrawer>
  )
}
