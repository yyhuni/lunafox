"use client"

import * as React from "react"
import { IconSearch, IconLoader2 } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

type SimpleSearchToolbarProps = {
  value: string
  onChange: (value: string) => void
  onSubmit?: () => void
  loading?: boolean
  placeholder?: string
  after?: React.ReactNode
  className?: string
  inputClassName?: string
  showButton?: boolean
}

export function SimpleSearchToolbar({
  value,
  onChange,
  onSubmit,
  loading = false,
  placeholder,
  after,
  className,
  inputClassName,
  showButton,
}: SimpleSearchToolbarProps) {
  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      onSubmit?.()
    }
  }

  const shouldShowButton = Boolean(onSubmit) && (showButton ?? true)

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Input
        type="search"
        name="search"
        autoComplete="off"
        placeholder={placeholder}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={handleKeyDown}
        className={cn("h-8 max-w-sm", inputClassName)}
      />
      {shouldShowButton && (
        <Button
          variant="outline"
          size="sm"
          onClick={onSubmit}
          disabled={loading}
        >
          {loading ? (
            <IconLoader2 className="h-4 w-4 animate-spin" />
          ) : (
            <IconSearch className="h-4 w-4" />
          )}
        </Button>
      )}
      {after}
    </div>
  )
}
