import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export const AGENT_OVERVIEW_HEADER_CLASS =
  "flex min-w-0 flex-col items-start gap-2 @4xl/main:flex-row @4xl/main:items-center @4xl/main:justify-between"

export const AGENT_OVERVIEW_TITLE_GROUP_CLASS =
  "flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1"

export const AGENT_CLUSTER_SUMMARY_ROOT_CLASS =
  "flex min-w-0 flex-wrap gap-y-4 border-y border-border py-4"

export const AGENT_CLUSTER_SUMMARY_STATE_CLASS =
  "flex min-w-0 w-full flex-col items-start justify-center @4xl/main:w-1/2 @5xl/main:w-1/4 @5xl/main:items-center @5xl/main:px-5"

export const AGENT_CLUSTER_CAPACITY_SUMMARY_CLASS =
  "flex min-w-0 w-full flex-col items-start justify-center @4xl/main:w-1/2 @5xl/main:w-1/4 @5xl/main:items-center @5xl/main:border-l @5xl/main:border-border @5xl/main:px-5"

export const AGENT_CLUSTER_METRICS_CLASS =
  "flex flex-wrap items-center gap-x-4 gap-y-2"

export const AGENT_CLUSTER_SUMMARY_CELL_DIVIDER_CLASS =
  "flex min-w-0 w-full flex-col items-start justify-center @4xl/main:w-1/2 @5xl/main:w-1/4 @5xl/main:items-center @5xl/main:border-l @5xl/main:border-border @5xl/main:px-5"

export const AGENT_TOOLBAR_SEARCH_MAX_WIDTH_CLASS = "@4xl/main:max-w-[360px]"

export const AGENT_TOOLBAR_ROOT_CLASS =
  "flex flex-col gap-3 @5xl/main:flex-row @5xl/main:items-start @5xl/main:justify-between"

export const AGENT_TOOLBAR_CONTROLS_CLASS =
  "flex min-w-0 flex-1 flex-col gap-3 @4xl/main:flex-row @4xl/main:items-center"

export const AGENT_TOOLBAR_FILTERS_CLASS =
  "flex min-w-0 flex-wrap items-center gap-3"

export const AGENT_TOOLBAR_ACTIONS_CLASS =
  "flex flex-wrap items-center gap-2 @5xl/main:justify-end"

export const AGENT_LIST_OVERVIEW_REGION_SLOT = "agent-list-overview-region"

export const AGENT_LIST_TOOLBAR_REGION_SLOT = "agent-list-toolbar-region"

export const AGENT_LIST_RESULTS_PRIMARY_REGION_SLOT = "agent-list-results-primary-region"

export const AGENT_CARD_GRID_CLASS =
  "grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-3 @7xl/main:grid-cols-4"

// Keeps the single-node installation affordance aligned with a regular card cell.
export const AGENT_EXPANSION_SLOT_MIN_HEIGHT_CLASS = "min-h-[252px]"

export const AGENT_CARD_SHELL_CLASS =
  "bg-card border border-border duration-200 flex flex-col gap-0 group overflow-hidden py-0 rounded-lg shadow-sm text-card-foreground transition-[background-color,border-color,box-shadow,opacity]"

export const AGENT_CARD_HEADER_CLASS =
  "bg-muted/20 border-b border-border/50 flex items-center justify-between p-3"

export const AGENT_CARD_HEADER_CONTENT_CLASS =
  "flex flex-1 gap-2.5 items-center min-w-0"

export const AGENT_CARD_TITLE_CLASS = cn(textRole.bodyStrong, "leading-none truncate")

export const AGENT_CARD_IP_CLASS =
  "font-mono mt-1 opacity-80 text-[10px] text-muted-foreground truncate"

export const AGENT_CARD_BODY_CLASS = "flex-1 p-4 space-y-4"

export const AGENT_CARD_INFO_GRID_CLASS = "gap-x-4 gap-y-2 grid grid-cols-2 text-[11px]"

export const AGENT_CARD_METRICS_CLASS = "pt-2 space-y-2.5"

export const AGENT_CARD_FOOTER_CLASS =
  "bg-muted/5 border-border/50 border-t grid gap-2 p-3"
