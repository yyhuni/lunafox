"use client"

import {
  ScreenshotsGalleryContent,
  ScreenshotsGalleryEmptyState,
  ScreenshotsGalleryLoadingState,
} from "./screenshots-gallery-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useScreenshotsGalleryState } from "./screenshots-gallery-state"

interface ScreenshotsGalleryProps {
  targetId?: number
  scanId?: number
}

export function ScreenshotsGallery({ targetId, scanId }: ScreenshotsGalleryProps) {
  const state = useScreenshotsGalleryState({ targetId, scanId })
  const isInitialLoading = state.isLoading && !state.data
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
      owner="screenshots-gallery-content"
      isLoading={isInitialLoading}
      skeleton={<ScreenshotsGalleryLoadingState state={state} />}
    >
      {state.screenshots.length === 0 && !state.filterQuery ? (
        <ScreenshotsGalleryEmptyState state={state} />
      ) : (
        <ScreenshotsGalleryContent state={state} />
      )}
    </ContentHandoff>
  )
}
