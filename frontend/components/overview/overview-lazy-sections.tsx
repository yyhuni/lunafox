"use client"

import { AgentLocationMap } from "@/components/overview/agent-location-map"
import { OverviewAssetOverview } from "@/components/overview/overview-asset-overview"
import { OverviewRiskSummary } from "@/components/overview/overview-risk-summary"
import { OverviewScanQueue, OverviewRuntimeDetails } from "@/components/overview/overview-runtime-details"
import {
  OVERVIEW_ASSET_REGION_LOADING_SLOT,
  OVERVIEW_OPERATIONAL_REGION_LOADING_SLOT,
  OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS,
  OVERVIEW_RUNTIME_REGION_LOADING_SLOT,
  OVERVIEW_SECTIONS_SHELL_CLASS,
  OverviewOperationalGrid,
  OverviewSingleSection,
} from "@/components/overview/overview-section-layouts"
import { cn } from "@/lib/utils"

export type OverviewLazySectionsProps = {
  className?: string
}

export function OverviewLazySections({ className }: OverviewLazySectionsProps) {
  return (
    <div className={cn(OVERVIEW_SECTIONS_SHELL_CLASS, className)}>
      <OverviewSingleSection data-loading-slot={OVERVIEW_RUNTIME_REGION_LOADING_SLOT}>
        <OverviewRuntimeDetails
          leading={<OverviewRiskSummary className={OVERVIEW_RUNTIME_DETAILS_CARD_SECTION_CLASS} />}
        />
      </OverviewSingleSection>

      <OverviewOperationalGrid data-loading-slot={OVERVIEW_OPERATIONAL_REGION_LOADING_SLOT}>
        <AgentLocationMap />
        <OverviewScanQueue />
      </OverviewOperationalGrid>

      <OverviewSingleSection data-loading-slot={OVERVIEW_ASSET_REGION_LOADING_SLOT}>
        <OverviewAssetOverview />
      </OverviewSingleSection>
    </div>
  )
}
