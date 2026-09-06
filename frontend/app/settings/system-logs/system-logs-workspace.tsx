"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"
import { SystemLogsLoadingState } from "@/components/settings/system-logs/system-logs-loading-state"
import {
  SYSTEM_LOGS_WORKSPACE_HANDOFF_CLASS,
  SYSTEM_LOGS_WORKSPACE_STATE_CLASS,
} from "@/components/settings/system-logs/system-logs-layout"
import type { SystemLogsViewProps } from "@/components/settings/system-logs/system-logs-view"

const SystemLogsView = dynamic<SystemLogsViewProps>(
  () => import("@/components/settings/system-logs/system-logs-view").then((m) => ({ default: m.SystemLogsView })),
  {
    ssr: false,
    loading: () => null,
  }
)

export function SystemLogsWorkspace() {
  const t = useTranslations("settings.systemLogs")
  const pageTitle = t("title")
  const pageDescription = t("description")

  return (
    <HiddenReadinessRouteBoundary
      owner="system-logs-page-route"
      layer="workspace"
      intent="data"
      skeleton={(
        <SystemLogsLoadingState
          pageTitle={pageTitle}
          pageDescription={pageDescription}
        />
      )}
      className={SYSTEM_LOGS_WORKSPACE_HANDOFF_CLASS}
      skeletonClassName={SYSTEM_LOGS_WORKSPACE_STATE_CLASS}
      contentClassName={SYSTEM_LOGS_WORKSPACE_STATE_CLASS}
    >
      {({ onReady, deferInitialSkeleton }) => (
        <SystemLogsView
          pageTitle={pageTitle}
          pageDescription={pageDescription}
          onReady={onReady}
          deferInitialSkeleton={deferInitialSkeleton}
        />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
