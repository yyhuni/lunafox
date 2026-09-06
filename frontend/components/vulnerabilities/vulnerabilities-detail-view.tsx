"use client"

import { useCallback, useState } from "react"

import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import {
  VulnerabilitiesDetailViewContent,
  VulnerabilitiesDetailViewDialogs,
  VulnerabilitiesDetailViewLoadingState,
} from "./vulnerabilities-detail-view-sections"
import { useVulnerabilitiesDetailViewState } from "./vulnerabilities-detail-view-state"
import { VulnerabilityDetailDrawer } from "./vulnerability-detail-drawer"

import type { Vulnerability } from "@/types/vulnerability.types"
import type { WebsiteAssetScope } from "@/types/website.types"

interface VulnerabilitiesDetailViewProps {
  /** Used in scan history page: view vulnerabilities by scan dimension */
  scanId?: number
  /** Used in target detail page: view vulnerabilities by target dimension */
  targetId?: number
  /** Hide toolbar (search, column controls, etc.) */
  hideToolbar?: boolean
  websiteScope?: WebsiteAssetScope
}

export function VulnerabilitiesDetailView({
  scanId,
  targetId,
  hideToolbar = false,
  websiteScope,
}: VulnerabilitiesDetailViewProps) {
  const [activeVulnerability, setActiveVulnerability] = useState<Vulnerability | null>(null)

  const handleSelectVulnerability = useCallback((vulnerability: Vulnerability) => {
    setActiveVulnerability(vulnerability)
  }, [])

  const handleDetailOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setActiveVulnerability(null)
    }
  }, [])

  const state = useVulnerabilitiesDetailViewState({ scanId, targetId, websiteScope })
  const isInitialLoading = (state.isLoading || state.isQueryLoading) && !state.activeQuery.data
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  return (
    <ContentHandoff
      owner="vulnerabilities-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={(
        <VulnerabilitiesDetailViewLoadingState
          state={state}
          rowCount={loadingRowCount}
          hideToolbar={hideToolbar}
          onRowClick={handleSelectVulnerability}
        />
      )}
    >
      <VulnerabilitiesDetailViewContent
        state={state}
        hideToolbar={hideToolbar}
        onRowClick={handleSelectVulnerability}
      />
      <VulnerabilityDetailDrawer
        vulnerability={activeVulnerability}
        open={Boolean(activeVulnerability)}
        onOpenChange={handleDetailOpenChange}
      />
      <VulnerabilitiesDetailViewDialogs state={state} />
    </ContentHandoff>
  )
}
