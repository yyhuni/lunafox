import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"

interface MasterDetailSkeletonProps {
  /** Number of items in the left list */
  listItemCount?: number
  /** Whether to show search box */
  withSearch?: boolean
  /** Page title */
  title?: string
}

/**
 * Master-detail layout skeleton screen
 * Suitable for scan engines, dictionary management, Nuclei templates and other pages
 */
export function MasterDetailSkeleton({
  listItemCount = 5,
  withSearch = true,
  title,
}: MasterDetailSkeletonProps) {
  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between gap-4 px-4 py-4 lg:px-6">
        {title ? (
          <h1 className="text-2xl font-bold shrink-0">{title}</h1>
        ) : (
          <Skeleton className="h-8 w-32" />
        )}
        {withSearch && (
          <div className="flex items-center gap-2 flex-1 max-w-md">
            <Skeleton className="h-9 w-full" />
          </div>
        )}
        <Skeleton className="h-9 w-24" />
      </div>

      <Separator />

      {/* Main content */}
      <div className="flex flex-1 min-h-0">
        {/* Left list */}
        <div className="w-72 lg:w-80 border-r flex flex-col">
          <div className="px-4 py-3 border-b">
            <Skeleton className="h-4 w-24" />
          </div>
          <div className="p-2 space-y-2">
            {Array.from({ length: listItemCount }).map((_, index) => (
              <div key={index} className="rounded-lg px-3 py-2.5 space-y-1.5">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-3 w-1/2" />
              </div>
            ))}
          </div>
        </div>

        {/* Right details */}
        <div className="flex-1 flex flex-col min-w-0">
          <div className="px-6 py-4 border-b">
            <div className="flex items-start gap-3">
              <Skeleton className="h-10 w-10 rounded-lg" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-5 w-48" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
          </div>
          <div className="flex-1 p-6 space-y-6">
            <div className="rounded-lg border p-4 space-y-3">
              <div className="grid grid-cols-2 gap-4">
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
          <div className="px-6 py-4 border-t flex items-center gap-2">
            <Skeleton className="h-8 w-24" />
            <div className="flex-1" />
            <Skeleton className="h-8 w-20" />
          </div>
        </div>
      </div>
    </div>
  )
}
