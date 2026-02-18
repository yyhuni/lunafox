"use client"

import {
  EndpointsDetailViewContent,
  EndpointsDetailViewDialogs,
  EndpointsDetailViewErrorState,
  EndpointsDetailViewLoadingState,
} from "./endpoints-detail-view-sections"
import { useEndpointsDetailViewState } from "./endpoints-detail-view-state"

interface EndpointsDetailViewProps {
  targetId?: number
  scanId?: number
}

/**
 * Target endpoint detail view component
 * Used to display and manage the endpoint list under a target
 */
export function EndpointsDetailView({ targetId, scanId }: EndpointsDetailViewProps) {
  const state = useEndpointsDetailViewState({ targetId, scanId })

  if (state.error) {
    return <EndpointsDetailViewErrorState state={state} />
  }

  if (state.isLoading && !state.data) {
    return <EndpointsDetailViewLoadingState />
  }

  return (
    <>
      <EndpointsDetailViewContent state={state} />
      <EndpointsDetailViewDialogs state={state} />
    </>
  )
}
