import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SegmentedMetricProgress } from "@/components/shared/metrics/segmented-metric-progress"
import { Card } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { textRole } from "@/lib/typography"

import {
  AGENT_CARD_BODY_CLASS,
  AGENT_CARD_FOOTER_CLASS,
  AGENT_CARD_HEADER_CLASS,
  AGENT_CARD_HEADER_CONTENT_CLASS,
  AGENT_CARD_IP_CLASS,
  AGENT_CARD_INFO_GRID_CLASS,
  AGENT_CARD_METRICS_CLASS,
  AGENT_CARD_SHELL_CLASS,
  AGENT_CARD_TITLE_CLASS,
} from "./agent-layout-contract"

export function AgentCardCompactLoadingState() {
  return (
    <Card data-testid="agent-skeleton-card" className={AGENT_CARD_SHELL_CLASS}>
      <div className={AGENT_CARD_HEADER_CLASS}>
        <div className={AGENT_CARD_HEADER_CONTENT_CLASS}>
          <Skeleton className="h-5 w-16 rounded-full" />
          <div className="min-w-0">
            {/* Keep the resolved line box: twMerge treats text-sm as conflicting with leading-none. */}
            <div className={`${AGENT_CARD_TITLE_CLASS} relative`}>
              <span aria-hidden="true" className="invisible">&nbsp;</span>
              <Skeleton className="!absolute left-0 top-1/2 h-3 w-24 -translate-y-1/2 rounded-full" />
            </div>
            <div className={`${AGENT_CARD_IP_CLASS} relative`}>
              <span aria-hidden="true" className="invisible">&nbsp;</span>
              <Skeleton className="!absolute left-0 top-1/2 h-2.5 w-20 -translate-y-1/2 rounded-full" />
            </div>
          </div>
        </div>
        <ActionSkeleton size="icon-sm" />
      </div>

      <div className={AGENT_CARD_BODY_CLASS}>
        <div className={AGENT_CARD_INFO_GRID_CLASS}>
          {Array.from({ length: 2 }).map((_, itemIndex) => (
            <div key={itemIndex} className={itemIndex === 1 ? "min-w-0 space-y-0.5 text-right" : "min-w-0 space-y-0.5"}>
              <span className={textRole.helperText}>
                <Skeleton className="inline-block h-3 w-12 align-middle rounded-full" />
              </span>
              <div className="font-mono text-xs truncate">
                <Skeleton className="inline-block h-3 w-full align-middle rounded-full" />
              </div>
            </div>
          ))}
        </div>

        <div className={AGENT_CARD_METRICS_CLASS}>
          {Array.from({ length: 3 }).map((_, metricIndex) => (
            <SegmentedMetricProgress key={metricIndex} loading />
          ))}
        </div>
      </div>

      <div className={AGENT_CARD_FOOTER_CLASS}>
        <div className="flex items-center justify-between gap-3">
          <Skeleton className="h-5 w-16 rounded-md" />
          <Skeleton className="h-5 w-16 rounded-md" />
        </div>
        <div className="flex min-w-0 items-center justify-between gap-3">
          <Skeleton className="h-4 w-16 rounded-full" />
          <Skeleton className="h-4 w-14 rounded-full" />
        </div>
      </div>
    </Card>
  )
}
