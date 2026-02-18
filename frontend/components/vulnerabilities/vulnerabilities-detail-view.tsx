"use client"

import {
  VulnerabilitiesDetailViewContent,
  VulnerabilitiesDetailViewDialogs,
  VulnerabilitiesDetailViewLoadingState,
} from "./vulnerabilities-detail-view-sections"
import { useVulnerabilitiesDetailViewState } from "./vulnerabilities-detail-view-state"

interface VulnerabilitiesDetailViewProps {
  /** Used in scan history page: view vulnerabilities by scan dimension */
  scanId?: number
  /** Used in target detail page: view vulnerabilities by target dimension */
  targetId?: number
  /** Hide toolbar (search, column controls, etc.) */
  hideToolbar?: boolean
}

export function VulnerabilitiesDetailView({
  scanId,
  targetId,
  hideToolbar = false,
}: VulnerabilitiesDetailViewProps) {
  const state = useVulnerabilitiesDetailViewState({ scanId, targetId })

  if ((state.isLoading || state.isQueryLoading) && !state.activeQuery.data) {
    return <VulnerabilitiesDetailViewLoadingState />
  }

  return (
    <>
      <VulnerabilitiesDetailViewContent state={state} hideToolbar={hideToolbar} />
      <VulnerabilitiesDetailViewDialogs state={state} />
    </>
  )
}
