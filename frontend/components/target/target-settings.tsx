"use client"

import {
  TargetSettingsContent,
  TargetSettingsDialogs,
  TargetSettingsLoadingState,
} from "./target-settings-sections"
import { BlacklistSettingsWorkspace } from "@/components/settings/blacklist/blacklist-settings-workspace"
import { AppErrorState } from "@/components/shared/feedback/app-error-state"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { getDataTableSkeletonRowCount } from "@/components/shared/loading/data-table-skeleton"
import { useDetailShellReadySignal } from "@/components/shared/loading/detail-shell-ready-context"
import { normalizeError } from "@/lib/errors/normalize-error"
import { useTargetSettingsState } from "./target-settings-state"
import type { TargetSettingsSection } from "./target-settings-layout"

interface TargetSettingsProps {
  targetId: number
  section?: TargetSettingsSection
}

/**
 * Target settings entry point. Each section is routed independently so only
 * the active data source and workspace are mounted.
 */
export function TargetSettings({ targetId, section = "blacklist" }: TargetSettingsProps) {
  if (section === "blacklist") {
    return <BlacklistSettingsWorkspace embedded targetId={targetId} />
  }

  return <TargetScheduledScansSettings targetId={targetId} />
}

function TargetScheduledScansSettings({ targetId }: { targetId: number }) {
  const state = useTargetSettingsState({ targetId })
  const isInitialLoading = state.isLoading
  const loadingRowCount = getDataTableSkeletonRowCount(state.pageSize)
  const detailShellReady = useDetailShellReadySignal(!isInitialLoading)

  if (detailShellReady?.deferInitialSkeleton && isInitialLoading) {
    return null
  }

  if (state.error) {
    return (
      <AppErrorState
        error={normalizeError(state.error, { notFoundKind: "unexpected-error" })}
        onRetry={state.refetch}
        variant="section"
      />
    )
  }

  return (
    <ContentHandoff
      owner="target-settings-content"
      isLoading={isInitialLoading}
      skeleton={<TargetSettingsLoadingState state={state} rowCount={loadingRowCount} />}
    >
      <TargetSettingsContent state={state} />
      <TargetSettingsDialogs state={state} />
    </ContentHandoff>
  )
}
