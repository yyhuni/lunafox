"use client"

import dynamic from "next/dynamic"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"
import { ScheduledScanPageLoadingState } from "@/components/scan/scheduled/scheduled-scan-page-sections"
import type { ScheduledScanPageProps } from "@/components/scan/scheduled/scheduled-scan-page"

const ScheduledScanPageContent = dynamic<ScheduledScanPageProps>(
  () => import("@/components/scan/scheduled/scheduled-scan-page"),
  {
    ssr: false,
    loading: () => null,
  }
)

export function ScheduledScanWorkspace() {
  return (
    <HiddenReadinessRouteBoundary
      owner="scheduled-scan-page-route"
      layer="workspace"
      intent="data"
      skeleton={<ScheduledScanPageLoadingState />}
      className="flex flex-col"
    >
      {({ onReady, deferInitialSkeleton }) => (
        <ScheduledScanPageContent onReady={onReady} deferInitialSkeleton={deferInitialSkeleton} />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
