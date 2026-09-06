"use client"

import {
  TargetOverviewContent,
  TargetOverviewDialog,
  TargetOverviewLoadingState,
} from "./target-overview-sections"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { createAppError } from "@/lib/errors/app-error"
import { normalizeError } from "@/lib/errors/normalize-error"
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
  const isInitialLoading = state.isLoading && !state.target
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  if (!isInitialLoading && (state.error || !state.target)) {
    return (
      <AppErrorState
        error={state.error ? normalizeError(state.error) : createAppError("resource-not-found", { retryable: false })}
        resourceLabel={state.locale === "zh" ? "目标" : "Target"}
        onRetry={state.refetch}
        actionHref="/targets/"
      />
    )
  }

  return (
    <ContentHandoff
      owner="target-overview-content"
      isLoading={isInitialLoading}
      skeleton={<TargetOverviewLoadingState />}
    >
      {state.target ? (
        <>
          <TargetOverviewContent state={state} />
          <TargetOverviewDialog state={state} />
        </>
      ) : null}
    </ContentHandoff>
  )
}
