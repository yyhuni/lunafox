"use client"

import * as React from "react"
import { IconSearch } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/shared/loading/spinner"
import { SearchInput } from "@/components/shared/search-input"
import { cn } from "@/lib/utils"
import type { DataTableToolbarDensity } from "@/types/data-table.types"

type SimpleSearchToolbarProps = {
  value: string
  onChange: (value: string) => void
  onSubmit?: () => void
  loading?: boolean
  placeholder?: string
  after?: React.ReactNode
  className?: string
  groupClassName?: string
  inputClassName?: string
  inputWidthMode?: "fixed" | "fill"
  showButton?: boolean
  showInlineIcon?: boolean
  toolbarDensity?: DataTableToolbarDensity
}

export function SimpleSearchToolbar({
  value,
  onChange,
  onSubmit,
  loading = false,
  placeholder,
  after,
  className,
  groupClassName,
  inputClassName,
  inputWidthMode = "fixed",
  showButton,
  showInlineIcon = true,
  toolbarDensity = "compact",
}: SimpleSearchToolbarProps) {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      onSubmit?.()
    }
  }

  const shouldShowButton = Boolean(onSubmit) && (showButton ?? false)
  const shouldRenderInlineIcon = showInlineIcon && !shouldShowButton
  const defaultInputWidthClassName = "w-full sm:w-72 lg:w-80"
  const fillInputWidthClassName = "w-full"
  const resolvedInputWidthClassName = inputWidthMode === "fill" ? fillInputWidthClassName : defaultInputWidthClassName
  const isFillWidth = inputWidthMode === "fill"
  const isStandardDensity = toolbarDensity === "standard"
  const toolbarClassName = isFillWidth
    ? "flex w-full flex-wrap items-center gap-2"
    : "flex w-full flex-wrap items-center gap-2 sm:w-auto"
  const searchGroupClassName = isFillWidth
    ? "flex min-w-48 flex-1 items-center gap-2"
    : "flex min-w-48 flex-1 items-center gap-2 sm:flex-none"

  return (
    <div className={cn(toolbarClassName, className)}>
      <div className={cn(searchGroupClassName, groupClassName)}>
        <SearchInput
          name="search"
          placeholder={placeholder}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          onKeyDown={handleKeyDown}
          toolbarDensity={toolbarDensity}
          showIcon={shouldRenderInlineIcon}
          loading={loading && !shouldShowButton}
          className={cn(resolvedInputWidthClassName, inputClassName)}
        />
        {shouldShowButton && (
          <Button
            variant="outline"
            size={isStandardDensity ? "icon" : "icon-sm"}
            onClick={onSubmit}
            disabled={loading}
            aria-label={placeholder}
          >
            {loading ? (
              <Spinner />
            ) : (
              <IconSearch data-icon="inline-start" />
            )}
          </Button>
        )}
      </div>
      {after}
    </div>
  )
}
