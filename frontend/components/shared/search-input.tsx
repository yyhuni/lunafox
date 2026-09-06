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
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        />
      ) : null}
      <Input
        {...props}
        type="search"
        size={isStandardDensity ? "default" : "sm"}
        autoComplete={autoComplete ?? "off"}
        className={cn(showIcon && "pl-9", loading && "pr-9", className)}
      />
      {loading ? (
        <Spinner className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
      ) : null}
    </div>
  )
}
