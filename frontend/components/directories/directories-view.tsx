"use client"

import {
  DirectoriesViewContent,
  DirectoriesViewLoadingState,
} from "./directories-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useDirectoriesViewState } from "./directories-view-state"
import type { WebsiteAssetScope } from "@/types/website.types"

export function DirectoriesView({
  targetId,
  scanId,
  websiteScope,
}: {
  targetId?: number
  scanId?: number
  websiteScope?: WebsiteAssetScope
}) {
  const state = useDirectoriesViewState({ targetId, scanId, websiteScope })
  const isInitialLoading = state.isLoading && !state.data
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
      owner="directories-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<DirectoriesViewLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <DirectoriesViewContent state={state} />
    </ContentHandoff>
  )
}
