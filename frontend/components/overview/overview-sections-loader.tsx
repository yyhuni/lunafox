"use client"

import dynamic from "next/dynamic"
import * as React from "react"
import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import { OverviewSectionsSkeleton } from "@/components/overview/overview-sections-skeleton"
import { loadOverviewLazySections, type OverviewLazySectionsProps } from "@/components/overview/overview-sections-dynamic"
import type { OverviewSectionsReadyProbeProps } from "@/components/overview/overview-sections-ready-probe"

const OverviewDynamicSections = dynamic<OverviewLazySectionsProps>(
  loadOverviewLazySections,
  {
    ssr: false,
    loading: () => null,
  }
)

const OverviewSectionsReadyProbe = dynamic<OverviewSectionsReadyProbeProps>(
  () => import("@/components/overview/overview-sections-ready-probe").then((mod) => mod.OverviewSectionsReadyProbe),
  {
    ssr: false,
    loading: () => null,
  }
)

export function OverviewSectionsLoader() {
  const [isReady, setIsReady] = React.useState(false)
  const handleReady = React.useCallback(() => setIsReady(true), [])

  return (
    <>
      {!isReady ? (
        <OverviewSectionsReadyProbe onReady={handleReady} />
      ) : null}
      <ContentHandoff
        owner="overview-sections-loader"
        isLoading={!isReady}
        skeleton={<OverviewSectionsSkeleton />}
        className="flex flex-col"
      >
        <OverviewDynamicSections />
      </ContentHandoff>
    </>
  )
}
