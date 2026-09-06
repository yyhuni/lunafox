"use client"

import { useCallback, useState } from "react"

import {
  WebSitesViewContent,
  WebSitesViewLoadingState,
} from "./websites-view-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useWebSitesViewState } from "./websites-view-state"
import { WebsiteDetailDrawer } from "./website-detail-drawer"

import type { WebSite } from "@/types/website.types"

export function WebSitesView({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  const [activeWebsite, setActiveWebsite] = useState<WebSite | null>(null)
  const state = useWebSitesViewState({ targetId, scanId })
  const isInitialLoading = state.isLoading && !state.data
  const loadingRowCount = getDataTableSkeletonRowCount(state.pagination.pageSize)

  const handleSelectWebsite = useCallback((website: WebSite) => {
    setActiveWebsite(website)
  }, [])

  const handleWebsiteDetailOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setActiveWebsite(null)
    }
  }, [])

  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  if (state.error) {
    const detailHref = targetId
      ? `/targets/${targetId}/overview/`
      : scanId
        ? `/scan/history/${scanId}/overview/`
        : "/overview/"

    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        onRetry={state.refetch}
        actionHref={detailHref}
      />
    )
  }

  return (
    <ContentHandoff
      owner="websites-detail-view-content"
      isLoading={isInitialLoading}
      skeleton={<WebSitesViewLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <WebSitesViewContent state={state} onRowClick={handleSelectWebsite} />
      <WebsiteDetailDrawer
        website={activeWebsite}
        open={Boolean(activeWebsite)}
        onOpenChange={handleWebsiteDetailOpenChange}
        formatDate={state.formatDate}
      />
    </ContentHandoff>
  )
}
