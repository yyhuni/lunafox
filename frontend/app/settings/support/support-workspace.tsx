"use client"

import dynamic from "next/dynamic"
import { HiddenReadinessRouteBoundary } from "@/components/shared/loading/hidden-readiness-route-boundary"
import type { SupportPageContentProps } from "@/components/settings/support/support-page-content"
import { SupportPageLoadingState } from "@/components/settings/support/support-page-loading-state"

const SupportPageContent = dynamic<SupportPageContentProps>(
  () => import("@/components/settings/support/support-page-content"),
  {
    ssr: false,
    loading: () => null,
  }
)

export function SupportWorkspace() {
  return (
    <HiddenReadinessRouteBoundary
      owner="support-page-route"
      layer="workspace"
      intent="data"
      skeleton={<SupportPageLoadingState />}
      className="flex min-h-0 flex-1 flex-col"
      skeletonClassName="flex min-h-0 flex-1 flex-col"
      contentClassName="flex min-h-0 flex-1 flex-col"
    >
      {({ onReady, deferInitialSkeleton }) => (
        <SupportPageContent
          onReady={onReady}
          deferInitialSkeleton={deferInitialSkeleton}
        />
      )}
    </HiddenReadinessRouteBoundary>
  )
}
