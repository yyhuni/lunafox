"use client"

import {
  WebSitesViewContent,
  WebSitesViewErrorState,
  WebSitesViewLoadingState,
} from "./websites-view-sections"
import { useWebSitesViewState } from "./websites-view-state"

export function WebSitesView({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  const state = useWebSitesViewState({ targetId, scanId })

  if (state.error) {
    return <WebSitesViewErrorState state={state} />
  }

  if (state.isLoading && !state.data) {
    return <WebSitesViewLoadingState />
  }

  return <WebSitesViewContent state={state} />
}
