"use client"

import {
  SubdomainsDetailViewContent,
  SubdomainsDetailViewDialogs,
  SubdomainsDetailViewLoadingState,
} from "./subdomains-detail-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useSubdomainsDetailViewState } from "./subdomains-detail-view-state"

interface SubdomainsDetailViewProps {
  targetId?: number
  scanId?: number
}

/**
 * Subdomain detail view component
 * Supports target and scan history modes.
 */
export function SubdomainsDetailView({ targetId, scanId }: SubdomainsDetailViewProps) {
  const state = useSubdomainsDetailViewState({ targetId, scanId })
  const isInitialLoading = state.isLoading && !state.subdomainsData
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  if (state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        onRetry={state.refetch}
        variant="section"
      />
    )
  }

  return (
    <ContentHandoff
      owner="subdomains-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<SubdomainsDetailViewLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <SubdomainsDetailViewContent state={state} />
      <SubdomainsDetailViewDialogs state={state} />
    </ContentHandoff>
  )
}
