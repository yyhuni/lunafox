"use client"

import {
  ScreenshotsGalleryContent,
  ScreenshotsGalleryEmptyState,
  ScreenshotsGalleryErrorState,
  ScreenshotsGalleryLoadingState,
} from "./screenshots-gallery-sections"
import { useScreenshotsGalleryState } from "./screenshots-gallery-state"

interface ScreenshotsGalleryProps {
  targetId?: number
  scanId?: number
}

export function ScreenshotsGallery({ targetId, scanId }: ScreenshotsGalleryProps) {
  const state = useScreenshotsGalleryState({ targetId, scanId })

  if (state.error) {
    return <ScreenshotsGalleryErrorState state={state} />
  }

  if (state.isLoading && !state.data) {
    return <ScreenshotsGalleryLoadingState />
  }

  if (state.screenshots.length === 0 && !state.filterQuery) {
    return <ScreenshotsGalleryEmptyState state={state} />
  }

  return <ScreenshotsGalleryContent state={state} />
}
