import type { ReactNode } from "react"

import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import type { DataTableToolbarDensity } from "@/types/data-table.types"

interface SearchToolbarSkeletonProps {
  after?: ReactNode
  className?: string
  groupClassName?: string
  inputClassName?: string
  inputWidthMode?: "fixed" | "fill"
  leadingSpaceClassName?: string
  showButton?: boolean
  placeholderWidthClassName?: string
  toolbarDensity?: DataTableToolbarDensity
}

export function SearchToolbarSkeleton({
  after,
  className,
  groupClassName,
  inputClassName,
  inputWidthMode = "fixed",
  leadingSpaceClassName,
  showButton = false,
  placeholderWidthClassName = "w-24",
  toolbarDensity = "compact",
}: SearchToolbarSkeletonProps) {
  const isStandardDensity = toolbarDensity === "standard"
  const defaultInputWidthClassName = "w-full sm:w-72 lg:w-80"
  const fillInputWidthClassName = "w-full"
  const resolvedInputWidthClassName = inputWidthMode === "fill" ? fillInputWidthClassName : defaultInputWidthClassName
  const resolvedLeadingSpaceClassName = leadingSpaceClassName ?? (!showButton ? "size-4" : undefined)

  return (
    <div
      data-slot="search-toolbar-skeleton"
      aria-hidden="true"
      className={cn("flex w-full flex-wrap items-center gap-2 sm:w-auto", className)}
    >
      <div className={cn("flex min-w-48 flex-1 items-center gap-2 sm:flex-none", groupClassName)}>
        <div className="relative flex-1">
          <Input
            type="search"
            size={isStandardDensity ? "default" : "sm"}
            disabled
            className={cn(
              resolvedInputWidthClassName,
              "disabled:cursor-default disabled:opacity-100",
              inputClassName
            )}
          />
          <div className="pointer-events-none absolute inset-y-0 left-3 right-3 flex items-center gap-2">
            {resolvedLeadingSpaceClassName ? (
              <span aria-hidden="true" className={cn("shrink-0 opacity-0", resolvedLeadingSpaceClassName)} />
            ) : null}
            <Skeleton className={cn("h-4 rounded-full", placeholderWidthClassName)} />
          </div>
        </div>
        {showButton ? (
          <ActionSkeleton size={isStandardDensity ? "icon" : "icon-sm"} />
        ) : null}
      </div>
      {after}
    </div>
  )
}
