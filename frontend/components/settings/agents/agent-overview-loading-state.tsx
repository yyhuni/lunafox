"use client"

import { AgentOverviewSection } from "./agent-overview-section"
import type { AgentClusterSummary } from "@/types/agent.types"

export function AgentOverviewLoadingState({ summary, activationPreview = false, showLicenseBanner = true }: { summary?: AgentClusterSummary; activationPreview?: boolean; showLicenseBanner?: boolean }) {
  return <AgentOverviewSection loading summary={summary} activationPreview={activationPreview} showLicenseBanner={showLicenseBanner} />
}
