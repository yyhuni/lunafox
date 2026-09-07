import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import {
  getLoadingOwnerAttributes,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"

interface PageSectionSkeletonProps {
  owner?: string
  layer?: LoadingLayer
  className?: string
  sectionCount?: number
  showToolbar?: boolean
}

export function PageSectionSkeleton({
  owner,
  layer = "route",
  className,
  sectionCount = 2,
  showToolbar = true,
}: PageSectionSkeletonProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("PageSectionSkeleton requires a non-empty owner.")
  }

  return (
    <div
      {...(owner ? getLoadingOwnerAttributes({ owner, layer, intent: "route" }) : {})}
      data-slot="page-section-skeleton"
      className={cn("space-y-6 px-4 py-4 md:py-6 lg:px-6", className)}
    >
      <div className="space-y-3">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-72 max-w-full" />
      </div>

      {showToolbar ? (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <SearchToolbarSkeleton
            inputWidthMode="fill"
            toolbarDensity="standard"
            className="max-w-sm"
            groupClassName="sm:flex-1"
          />
          <div className="flex gap-2">
            <ActionSkeleton widthClassName="w-24" />
            <ActionSkeleton size="icon" />
          </div>
        </div>
      ) : null}

      <div className="space-y-4">
        {Array.from({ length: sectionCount }).map((_, index) => (
          <div key={index} className="rounded-xl border border-border bg-card/70 p-5 shadow-2xs">
            <div className="space-y-3">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-28 w-full rounded-xl" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
