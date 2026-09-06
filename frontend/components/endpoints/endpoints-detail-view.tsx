"use client"

import { useCallback, useState } from "react"

import {
  EndpointsDetailViewContent,
  EndpointsDetailViewDialogs,
  EndpointsDetailViewLoadingState,
} from "./endpoints-detail-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useEndpointsDetailViewState } from "./endpoints-detail-view-state"
import { EndpointDetailDrawer } from "./endpoint-detail-drawer"

import type { Endpoint } from "@/types/endpoint.types"
import type { WebsiteAssetScope } from "@/types/website.types"

interface EndpointsDetailViewProps {
  targetId?: number
  scanId?: number
  websiteScope?: WebsiteAssetScope
}

/**
 * Target endpoint detail view component
 * Used to display and manage the endpoint list under a target
 */
export function EndpointsDetailView({ targetId, scanId, websiteScope }: EndpointsDetailViewProps) {
  const [activeEndpoint, setActiveEndpoint] = useState<Endpoint | null>(null)
  const state = useEndpointsDetailViewState({ targetId, scanId, websiteScope })
  const isInitialLoading = state.isLoading && !state.data
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  const handleSelectEndpoint = useCallback((endpoint: Endpoint) => {
    setActiveEndpoint(endpoint)
  }, [])

  const handleEndpointDetailOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setActiveEndpoint(null)
    }
  }, [])

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
      owner="endpoints-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<EndpointsDetailViewLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <EndpointsDetailViewContent state={state} onRowClick={handleSelectEndpoint} />
      <EndpointsDetailViewDialogs state={state} />
      <EndpointDetailDrawer
        endpoint={activeEndpoint}
        open={Boolean(activeEndpoint)}
        onOpenChange={handleEndpointDetailOpenChange}
        formatDate={state.formatDate}
      />
    </ContentHandoff>
  )
}
