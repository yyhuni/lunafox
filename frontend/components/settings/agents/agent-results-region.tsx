import type { ReactNode } from "react"

import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { COMPACT_SECTION_STACK_CLASS } from "@/components/shared/layout/page-shell-density"

import { AGENT_LIST_RESULTS_PRIMARY_REGION_SLOT } from "./agent-layout-contract"

export function AgentResultsRegion({ children }: { children: ReactNode }) {
  return (
    <div {...getLoadingStructureSlotAttributes(AGENT_LIST_RESULTS_PRIMARY_REGION_SLOT)} className={COMPACT_SECTION_STACK_CLASS}>
      {children}
    </div>
  )
}
