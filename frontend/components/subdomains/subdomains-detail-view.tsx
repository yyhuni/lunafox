"use client"

import {
  SubdomainsDetailViewContent,
  SubdomainsDetailViewDialogs,
  SubdomainsDetailViewErrorState,
  SubdomainsDetailViewLoadingState,
} from "./subdomains-detail-view-sections"
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

  if (state.error) {
    return <SubdomainsDetailViewErrorState state={state} />
  }

  if (state.isLoading && !state.subdomainsData) {
    return <SubdomainsDetailViewLoadingState />
  }

  return (
    <>
      <SubdomainsDetailViewContent state={state} />
      <SubdomainsDetailViewDialogs state={state} />
    </>
  )
}
