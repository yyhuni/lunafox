import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { SelectShellSkeleton } from "@/components/shared/loading/select-shell-skeleton"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"

interface CompactPaginationSkeletonProps {
  mode?: "numbered" | "cursor"
  showSummary?: boolean
  summaryWidthClassName?: string
  rowsPerPageLabelWidthClassName?: string
  pageValueWidthClassName?: string
  pageSizeWidthClassName?: string
  buttonCount?: 2 | 3 | 4
  className?: string
}

export function CompactPaginationSkeleton({
  mode = "numbered",
  showSummary = mode !== "cursor",
  summaryWidthClassName = "w-28",
  rowsPerPageLabelWidthClassName = "w-20",
  pageValueWidthClassName = "w-14",
  pageSizeWidthClassName = "w-24",
  buttonCount = 4,
  className,
}: CompactPaginationSkeletonProps) {
  const showEdgeButtons = buttonCount === 4
  const showFirstButton = buttonCount === 3 || showEdgeButtons
  const isCursorPagination = mode === "cursor"

  return (
    <div
      data-slot="compact-pagination-skeleton"
      aria-hidden="true"
      className={cn("flex flex-col gap-3 px-2 sm:flex-row sm:items-center sm:justify-between", className)}
    >
      {showSummary ? <Skeleton className={cn("h-5 rounded-full", summaryWidthClassName)} /> : null}

      <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-center sm:gap-4">
        <div className="flex items-center gap-2">
          <Skeleton className={cn("h-5 rounded-full", rowsPerPageLabelWidthClassName)} />
          <SelectShellSkeleton size="sm" widthClassName={pageSizeWidthClassName} valueWidthClassName="w-8" />
        </div>

        {!isCursorPagination ? <Skeleton className={cn("h-5 rounded-full", pageValueWidthClassName)} /> : null}

        <div className="flex items-center gap-2">
          {showFirstButton ? <ActionSkeleton size="icon-sm" className={isCursorPagination ? undefined : "hidden lg:block"} /> : null}
          <ActionSkeleton size="icon-sm" />
          <ActionSkeleton size="icon-sm" />
          {showEdgeButtons ? <ActionSkeleton size="icon-sm" className="hidden lg:block" /> : null}
        </div>
      </div>
    </div>
  )
}
