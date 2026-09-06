import { cn } from "@/lib/utils"
import { Skeleton } from "@/components/ui/skeleton"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import {
  getLoadingOwnerAttributes,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"

interface CardGridSkeletonProps {
  owner?: string
  layer?: LoadingLayer
  cards?: number
  actionButtonCount?: number
  showToolbar?: boolean
  withPadding?: boolean
  className?: string
}

/**
 * Generic card grid skeleton screen
 * Suitable for tool lists and other card layouts
 */
export function CardGridSkeleton({
  owner,
  layer = "workspace",
  cards = 4,
  actionButtonCount = 2,
  showToolbar = true,
  withPadding = true,
  className,
}: CardGridSkeletonProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("CardGridSkeleton requires a non-empty owner.")
  }

  const containerClass = cn(
    "flex flex-col gap-4",
    withPadding && "px-4 lg:px-6",
    className
  )

  return (
    <div {...(owner ? getLoadingOwnerAttributes({ owner, layer, intent: "data" }) : {})} data-slot="card-grid-skeleton" className={containerClass}>
      {showToolbar && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <SearchToolbarSkeleton
            inputWidthMode="fill"
            toolbarDensity="standard"
            className="sm:max-w-sm"
            groupClassName="sm:flex-1"
          />
          <div className="flex gap-2 items-center">
            {Array.from({ length: actionButtonCount }).map((_, index) => (
              <ActionSkeleton key={index} widthClassName="w-24" />
            ))}
          </div>
        </div>
      )}

      <div className="gap-4 grid grid-cols-1 lg:grid-cols-3 md:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: cards }).map((_, index) => (
          <div key={index} className="bg-card border border-border rounded-xl p-4 shadow-2xs space-y-4">
            <div className="space-y-2">
              <Skeleton className="h-5 w-3/4" />
              <Skeleton className="h-4 w-full rounded-full" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-1/2 rounded-full" />
              <Skeleton className="h-20 w-full rounded-xl" />
            </div>
            <div className="flex flex-wrap gap-2">
              {Array.from({ length: 3 }).map((_, badgeIndex) => (
                <Skeleton key={badgeIndex} className="h-6 w-16 rounded-full" />
              ))}
            </div>
            <div className="flex gap-2">
              <ActionSkeleton widthClassName="w-full" />
              <ActionSkeleton widthClassName="w-full" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
