"use client"

import {
  DirectoriesViewContent,
  DirectoriesViewErrorState,
  DirectoriesViewLoadingState,
} from "./directories-view-sections"
import { useDirectoriesViewState } from "./directories-view-state"

export function DirectoriesView({
  targetId,
  scanId,
}: {
  targetId?: number
  scanId?: number
}) {
  const state = useDirectoriesViewState({ targetId, scanId })

  if (state.error) {
    return <DirectoriesViewErrorState state={state} />
  }

  if (state.isLoading && !state.data) {
    return <DirectoriesViewLoadingState />
  }

  return <DirectoriesViewContent state={state} />
}
