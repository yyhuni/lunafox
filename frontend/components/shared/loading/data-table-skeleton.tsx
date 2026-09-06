import { cn } from "@/lib/utils"
import { Skeleton } from "@/components/ui/skeleton"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { CompactPaginationSkeleton } from "@/components/shared/loading/compact-pagination-skeleton"
import {
  getLoadingOwnerAttributes,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"
import { SearchToolbarSkeleton } from "@/components/shared/loading/search-toolbar-skeleton"
import {
  TABLE_DENSE_CELL_RHYTHM_CLASS,
  TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX,
  TABLE_DENSE_ROW_CLASS,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

interface DataTableSkeletonProps {
  owner?: string
  nested?: boolean
  layer?: LoadingLayer
  statsCount?: number
  toolbarButtonCount?: number
  rows?: number
  mobileRows?: number
  columns?: number
  withSearch?: boolean
  searchInputWidthMode?: "fixed" | "fill"
  toolbarHeight?: "compact" | "tall"
  paginationButtonCount?: number
  withPagination?: boolean
  withPadding?: boolean
  className?: string
}

export const DATA_TABLE_SKELETON_MAX_ROWS = 10

export function getDataTableSkeletonRowCount(
  pageSize: number,
  maxRows = DATA_TABLE_SKELETON_MAX_ROWS
) {
  if (!Number.isFinite(pageSize) || pageSize <= 0) {
    throw new Error("getDataTableSkeletonRowCount requires a positive pageSize.")
  }

  if (!Number.isFinite(maxRows) || maxRows <= 0) {
    throw new Error("getDataTableSkeletonRowCount requires a positive maxRows.")
  }

  return Math.min(Math.floor(pageSize), Math.floor(maxRows))
}

/**
 * Generic data table skeleton screen
 * Configurable stats cards, toolbar buttons, table columns, etc.
 */
export function DataTableSkeleton({
  owner,
  nested = false,
  layer = "workspace",
  statsCount = 0,
  toolbarButtonCount = 2,
  rows = 5,
  mobileRows,
  columns = 4,
  withSearch = true,
  searchInputWidthMode = "fixed",
  toolbarHeight = "compact",
  paginationButtonCount = 4,
  withPagination = true,
  withPadding = true,
  className,
}: DataTableSkeletonProps) {
  if (owner !== undefined && !owner.trim()) {
    throw new Error("DataTableSkeleton owner must be non-empty when provided.")
  }
  if (!owner && !nested) {
    throw new Error("DataTableSkeleton requires an owner unless nested.")
  }

  const ownerAttributes = owner ? getLoadingOwnerAttributes({ owner, layer, intent: "data" }) : {}

  const columnWidthClasses = ["w-10", "w-48", "w-32", "w-40", "w-20", "w-24", "w-32", "w-36"]
  const narrowRows = mobileRows ?? rows
  const renderedRows = Math.max(rows, narrowRows)
  const containerClass = cn(
    "space-y-4",
    withPadding && "px-4 lg:px-6",
    className
  )

  const toolbarNeeded = withSearch || toolbarButtonCount > 0

  return (
    <div {...ownerAttributes} data-slot="data-table-skeleton" className={containerClass}>
      {statsCount > 0 && (
        <div className="gap-4 grid lg:grid-cols-4 sm:grid-cols-2">
          {Array.from({ length: statsCount }).map((_, index) => (
            <div key={index} className="bg-card border border-border rounded-xl p-4 shadow-2xs space-y-3">
              <Skeleton className="h-3.5 w-20 rounded-full" />
              <Skeleton className="h-7 w-24" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
      )}

      {toolbarNeeded && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          {withSearch ? (
            <div className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:max-w-xl sm:flex-1">
              <SearchToolbarSkeleton
                toolbarDensity={toolbarHeight === "tall" ? "standard" : "compact"}
                inputWidthMode={searchInputWidthMode}
                placeholderWidthClassName={toolbarHeight === "tall" ? "w-32" : "w-24"}
              />
            </div>
          ) : null}
          {toolbarButtonCount > 0 && (
            <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
              <div className="flex flex-wrap gap-2 items-center">
                {Array.from({ length: toolbarButtonCount }).map((_, index) => (
                  <ActionSkeleton
                    key={index}
                    size={toolbarHeight === "tall" ? "default" : "sm"}
                    widthClassName="w-24"
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      <div className="overflow-x-auto rounded-md border border-border bg-card">
        <table className="caption-bottom text-sm w-full table-fixed min-w-max">
          <TableHeader>
            <TableRow>
              {Array.from({ length: columns }).map((_, index) => (
                <TableHead key={index}>
                  <Skeleton
                    className={cn(
                      "h-3.5 rounded-full",
                      columnWidthClasses[index % columnWidthClasses.length]
                    )}
                  />
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>

          <TableBody>
            {Array.from({ length: renderedRows }).map((_, rowIndex) => (
              <TableRow
                key={rowIndex}
                className={cn(
                  TABLE_DENSE_ROW_CLASS,
                  rowIndex >= rows && "md:hidden",
                  rowIndex >= narrowRows && "hidden md:table-row",
                  rowIndex === narrowRows - 1 && "border-b-0",
                  rowIndex === rows - 1 && "md:border-b-0"
                )}
                style={{ height: TABLE_DENSE_ROW_ESTIMATED_HEIGHT_PX }}
              >
                {Array.from({ length: columns }).map((_, colIndex) => (
                  <TableCell key={colIndex} className={TABLE_DENSE_CELL_RHYTHM_CLASS}>
                    <Skeleton
                      className={cn(
                        "h-4 rounded-full",
                        columnWidthClasses[colIndex % columnWidthClasses.length]
                      )}
                    />
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </table>
      </div>

      {withPagination ? (
        <CompactPaginationSkeleton
          summaryWidthClassName="w-48"
          rowsPerPageLabelWidthClassName="w-24"
          pageValueWidthClassName="w-20"
          buttonCount={paginationButtonCount >= 4 ? 4 : 2}
        />
      ) : null}
    </div>
  )
}
