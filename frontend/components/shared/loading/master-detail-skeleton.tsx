import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"
import {
  getLoadingOwnerAttributes,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"

interface MasterDetailSkeletonProps {
  owner?: string
  layer?: LoadingLayer
  /** Number of items in the left list */
  listItemCount?: number
  /** Whether to show search box */
  withSearch?: boolean
  /** Page title */
  title?: string
}

/**
 * Master-detail layout skeleton screen
 * Suitable for scan workflow templates, dictionary management, Nuclei templates and other pages
 */
export function MasterDetailSkeleton({
  owner,
  layer = "workspace",
  listItemCount = 5,
  withSearch = true,
  title,
}: MasterDetailSkeletonProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("MasterDetailSkeleton requires a non-empty owner.")
  }

  return (
    <div {...(owner ? getLoadingOwnerAttributes({ owner, layer, intent: "data" }) : {})} data-slot="master-detail-skeleton" className="flex flex-col h-full">
      {/* Header */}
      <div className="flex gap-4 items-start justify-between lg:px-6 px-4 py-4">
        {title ? (
          <h1 className={cn("shrink-0", textRole.pageTitle)}>{title}</h1>
        ) : (
          <div aria-hidden="true" className={cn("relative w-32", textRole.pageTitle)}>
            <span className="invisible">{"\u00a0"}</span>
            <Skeleton className="absolute inset-0" />
          </div>
        )}
        {withSearch && (
          <SearchToolbarSkeleton
            inputWidthMode="fill"
            toolbarDensity="standard"
            className="max-w-md"
            groupClassName="sm:flex-1"
          />
        )}
        <ActionSkeleton widthClassName="w-24" />
      </div>

      <Separator />

      {/* Main content */}
      <div className="flex flex-1 min-h-0">
        {/* Left list */}
        <div className="border-r flex flex-col lg:w-80 w-72">
          <div className="border-b px-4 py-3">
            <Skeleton className="h-4 w-24" />
          </div>
          <div className="space-y-2 p-2">
            {Array.from({ length: listItemCount }).map((_, index) => (
              <div key={index} className="space-y-1.5 rounded-lg px-3 py-2.5">
                <Skeleton className="h-4 w-3/4 rounded-full" />
                <Skeleton className="h-3 w-1/2 rounded-full" />
              </div>
            ))}
          </div>
        </div>

        {/* Right details */}
        <div className="flex flex-1 flex-col min-w-0">
          <div className="border-b px-6 py-4">
            <div className="flex gap-3 items-start">
              <Skeleton className="h-10 rounded-lg w-10" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-5 w-48" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
          </div>
          <div className="flex-1 p-6 space-y-6">
            <div className="space-y-3 rounded-xl border border-border bg-card/70 p-4 shadow-2xs">
              <div className="gap-4 grid grid-cols-2">
                <div className="space-y-2">
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-6 w-24" />
                </div>
                <div className="space-y-2">
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className="h-6 w-24" />
                </div>
              </div>
              <Separator />
              <div className="space-y-3">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            </div>
          </div>
          <div className="border-t flex gap-2 items-center px-6 py-4">
            <ActionSkeleton size="sm" widthClassName="w-24" />
            <div className="flex-1" />
            <ActionSkeleton size="sm" widthClassName="w-20" />
          </div>
        </div>
      </div>
    </div>
  )
}
