"use client"

import dynamic from "next/dynamic"
import { useTranslations } from "next-intl"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"

import { DatabaseHealthLoadingState } from "@/components/settings/database-health/database-health-loading-state"
import {
  DATABASE_HEALTH_WORKSPACE_HANDOFF_CLASS,
  DATABASE_HEALTH_WORKSPACE_STATE_CLASS,
} from "@/components/settings/database-health/database-health-layout"
import {
  type DatabaseHealthViewProps,
} from "@/components/settings/database-health/database-health-view"

const DatabaseHealthView = dynamic<DatabaseHealthViewProps>(
  () => import("@/components/settings/database-health/database-health-view").then((m) => ({ default: m.DatabaseHealthView })),
  {
    ssr: false,
    loading: () => null,
  }
)

export function DatabaseHealthWorkspace() {
  const tPage = useTranslations("pages.databaseHealth")
  const pageTitle = tPage("title")
  const pageDescription = tPage("description")

  return (
    <HiddenReadinessRouteBoundary
      owner="database-health-page-route"
      layer="workspace"
      intent="data"
      skeleton={(
        <DatabaseHealthLoadingState
          pageTitle={pageTitle}
          pageDescription={pageDescription}
        />
      )}
      className={DATABASE_HEALTH_WORKSPACE_HANDOFF_CLASS}
      skeletonClassName={DATABASE_HEALTH_WORKSPACE_STATE_CLASS}
      contentClassName={DATABASE_HEALTH_WORKSPACE_STATE_CLASS}
    >
      {({ onReady, deferInitialSkeleton }) => (
        <DatabaseHealthView
          pageTitle={pageTitle}
          pageDescription={pageDescription}
          onReady={onReady}
          deferInitialSkeleton={deferInitialSkeleton}
        />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
