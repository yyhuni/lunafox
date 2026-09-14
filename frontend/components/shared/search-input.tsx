"use client"

import * as React from "react"

import { IconSearch } from "@/components/icons"
import { Spinner } from "@/components/shared/loading/spinner"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import type { DataTableToolbarDensity } from "@/types/data-table.types"

type SearchInputProps = Omit<React.ComponentProps<typeof Input>, "size" | "type"> & {
  loading?: boolean
  showIcon?: boolean
  toolbarDensity?: DataTableToolbarDensity
}

export function SearchInput({
  className,
  loading = false,
  showIcon = true,
  toolbarDensity = "compact",
  autoComplete,
  ...props
}: SearchInputProps) {
  const isStandardDensity = toolbarDensity === "standard"

  return (
    <div className="relative flex-1">
      {showIcon ? (
        <IconSearch
          aria-hidden="true"
          className={cn(
            "pointer-events-none absolute top-1/2 -translate-y-1/2 text-muted-foreground",
            isStandardDensity ? "left-3 size-4" : "left-2.5 size-3.5"
          )}
        />
      ) : null}
      <Input
        {...props}
        type="search"
        size={isStandardDensity ? "default" : "sm"}
        autoComplete={autoComplete ?? "off"}
        className={cn(
          showIcon && (isStandardDensity ? "pl-9" : "pl-7"),
          loading && (isStandardDensity ? "pr-9" : "pr-8"),
          className
        )}
      />
      {loading ? (
        <Spinner className={cn(
          "pointer-events-none absolute top-1/2 -translate-y-1/2 text-muted-foreground",
          isStandardDensity ? "right-3" : "right-2.5"
        )} />
      ) : null}
    </div>
  )
}
