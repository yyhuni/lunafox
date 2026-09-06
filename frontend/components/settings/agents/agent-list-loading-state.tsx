import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { getLoadingStructureSlotAttributes } from "@/components/shared/loading/loading-owner"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import { cn } from "@/lib/utils"

import { AgentCardCompactLoadingState } from "./agent-card-compact-loading-state"
import {
  AGENT_CARD_GRID_CLASS,
  AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS,
  AGENT_LIST_TOOLBAR_REGION_SLOT,
  AGENT_TOOLBAR_ACTIONS_CLASS,
  AGENT_TOOLBAR_CONTROLS_CLASS,
  AGENT_TOOLBAR_FILTERS_CLASS,
  AGENT_TOOLBAR_ROOT_CLASS,
  AGENT_TOOLBAR_SEARCH_MAX_WIDTH_CLASS,
} from "./agent-layout-contract"
import { AgentResultsRegion } from "./agent-results-region"

export function AgentToolbarLoadingState() {
  return (
    <div
      {...getLoadingStructureSlotAttributes(AGENT_LIST_TOOLBAR_REGION_SLOT)}
      data-testid="agent-toolbar-loading-state"
      className={AGENT_TOOLBAR_ROOT_CLASS}
    >
      <div className={AGENT_TOOLBAR_CONTROLS_CLASS}>
        <SearchToolbarSkeleton
          className={cn("w-full sm:w-full", AGENT_TOOLBAR_SEARCH_MAX_WIDTH_CLASS)}
          groupClassName="w-full"
          inputWidthMode="fill"
          leadingSpaceClassName="size-4"
          placeholderWidthClassName="w-28"
        />
        <div className={AGENT_TOOLBAR_FILTERS_CLASS}>
          <ActionSkeleton size="sm" widthClassName="w-24" />
        </div>
      </div>
      <div className={AGENT_TOOLBAR_ACTIONS_CLASS}>
        <ActionSkeleton size="sm" widthClassName="w-24" />
        <ActionSkeleton size="sm" widthClassName="w-28" />
      </div>
    </div>
  )
}

export function AgentCardsLoadingState() {
  return (
    <AgentResultsRegion>
      <div data-testid="agent-cards-loading-state" className={AGENT_CARD_GRID_CLASS}>
        {Array.from({ length: 10 }).map((_, index) => (
          <AgentCardCompactLoadingState key={index} />
        ))}
        <ActionSkeleton
          size="action-card"
          widthClassName={AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS}
          className="rounded-lg border-dashed shadow-none"
        />
      </div>
    </AgentResultsRegion>
  )
}
