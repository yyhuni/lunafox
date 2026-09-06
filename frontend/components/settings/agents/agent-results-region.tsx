import type { ReactNode } from "react"

import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"

import { AGENT_LIST_RESULTS_PRIMARY_REGION_SLOT } from "./agent-layout-contract"

export function AgentResultsRegion({ children }: { children: ReactNode }) {
  return (
    <div {...getLoadingStructureSlotAttributes(AGENT_LIST_RESULTS_PRIMARY_REGION_SLOT)} className="space-y-4">
      {children}
    </div>
  )
}
