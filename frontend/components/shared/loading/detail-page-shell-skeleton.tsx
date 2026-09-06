import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"
import type { ReactNode } from "react"
import {
  getLoadingOwnerAttributes,
  type LoadingIntent,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"

interface DetailPageShellSkeletonProps {
  owner?: string
  layer?: LoadingLayer
  intent?: LoadingIntent
  primaryTabCount?: number
  primaryTabLabels?: readonly string[]
  activePrimaryTabIndex?: number
  secondaryTabCount?: number
  showSecondaryNav?: boolean
  className?: string
  children?: ReactNode
}

const PRIMARY_TAB_WIDTHS = ["w-20", "w-20", "w-24", "w-28", "w-24", "w-20"]
const SECONDARY_TAB_WIDTHS = ["w-20", "w-24", "w-16", "w-20"]

export function DetailPageShellSkeleton({
  owner,
  layer = "route",
  intent = "route",
  primaryTabCount,
  primaryTabLabels,
  activePrimaryTabIndex = 0,
  secondaryTabCount = 4,
  showSecondaryNav = false,
  className,
  children,
}: DetailPageShellSkeletonProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("DetailPageShellSkeleton owner must be non-empty when provided.")
  }

  if (primaryTabLabels?.some((label) => !label.trim())) {
    throw new Error("DetailPageShellSkeleton primaryTabLabels must not contain blank labels.")
  }

  if (primaryTabLabels && primaryTabCount !== undefined && primaryTabLabels.length !== primaryTabCount) {
    throw new Error("DetailPageShellSkeleton primaryTabLabels must match primaryTabCount.")
  }

  const resolvedPrimaryTabCount = primaryTabLabels?.length ?? primaryTabCount ?? 5
  if (!Number.isInteger(activePrimaryTabIndex) || activePrimaryTabIndex < 0 || activePrimaryTabIndex >= resolvedPrimaryTabCount) {
    throw new Error("DetailPageShellSkeleton activePrimaryTabIndex must reference a rendered tab.")
  }

  const ownerAttributes = owner ? getLoadingOwnerAttributes({ owner, layer, intent }) : {}

  return (
    <div
      {...ownerAttributes}
      data-slot="detail-page-shell-skeleton"
      className={cn("flex flex-col gap-4 py-4 md:gap-6 md:py-6", className)}
    >
      <div className="flex items-center gap-2 text-sm px-4 lg:px-6">
        <Skeleton className="h-4 w-16 rounded-full" />
        <span className="text-muted-foreground">/</span>
        <Skeleton className="h-4 w-32 rounded-full" />
      </div>

      <div className="px-4 lg:px-6">
        <Tabs value={`tab-${activePrimaryTabIndex}`}>
          <TabsList>
            {Array.from({ length: resolvedPrimaryTabCount }).map((_, index) => (
              <TabsTrigger key={index} value={`tab-${index}`} disabled>
                {primaryTabLabels ? (
                  <span aria-hidden="true" className="grid">
                    <span className="invisible col-start-1 row-start-1 inline-flex items-center gap-1.5 whitespace-nowrap">
                      <span className="h-4 w-4 shrink-0" />
                      <span>{primaryTabLabels[index]}</span>
                    </span>
                    <Skeleton className="col-start-1 row-start-1 h-4 w-full self-center rounded-full" />
                  </span>
                ) : (
                  <Skeleton
                    className={cn(
                      "h-4 rounded-full",
                      PRIMARY_TAB_WIDTHS[index] ?? PRIMARY_TAB_WIDTHS[PRIMARY_TAB_WIDTHS.length - 1]
                    )}
                  />
                )}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {showSecondaryNav ? (
        <div className="px-4 lg:px-6">
          <Tabs value="secondary-tab-0">
            <TabsList variant="content">
              {Array.from({ length: secondaryTabCount }).map((_, index) => (
                <TabsTrigger key={index} value={`secondary-tab-${index}`} variant="content" disabled>
                  <Skeleton
                    className={cn(
                      "h-4 rounded-full",
                      SECONDARY_TAB_WIDTHS[index] ?? SECONDARY_TAB_WIDTHS[SECONDARY_TAB_WIDTHS.length - 1]
                    )}
                  />
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
      ) : null}

      {children}
    </div>
  )
}
