"use client"

import {
  TargetOverviewContent,
  TargetOverviewDialog,
  TargetOverviewErrorState,
  TargetOverviewLoadingState,
} from "./target-overview-sections"
import { useTargetOverviewState } from "./target-overview-state"

interface TargetOverviewProps {
  targetId: number
}

/**
 * Target overview component
 * Displays statistics cards for the target
 */
export function TargetOverview({ targetId }: TargetOverviewProps) {
  const state = useTargetOverviewState({ targetId })

  if (state.isLoading) {
    return <TargetOverviewLoadingState />
  }

  if (state.error || !state.target) {
    return <TargetOverviewErrorState t={state.t} />
  }

  return (
    <>
      <TargetOverviewContent state={state} />
      <TargetOverviewDialog state={state} />
    </>
  )
}
