"use client"

import {
  IPAddressesViewContent,
  IPAddressesViewLoadingState,
} from "./ip-addresses-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useIPAddressesViewState } from "./ip-addresses-view-state"
import type { WebsiteAssetScope } from "@/types/website.types"

export function IPAddressesView({
  targetId,
  scanId,
  websiteScope,
}: {
  targetId?: number
  scanId?: number
  websiteScope?: WebsiteAssetScope
}) {
  const state = useIPAddressesViewState({ targetId, scanId, websiteScope })
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
      owner="ip-addresses-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<IPAddressesViewLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <IPAddressesViewContent state={state} />
    </ContentHandoff>
  )
}
