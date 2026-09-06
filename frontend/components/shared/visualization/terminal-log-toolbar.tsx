"use client"

import { IconSearch } from "@/components/icons"
import { ActionSkeleton } from "@/components/shared/loading/action-skeleton"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { StructuredLogLevelFilter } from "@/components/shared/visualization/structured-log-viewer"
import { cn } from "@/lib/utils"

export const TERMINAL_LOG_LEVEL_FILTERS = ["all", "error", "warn", "info", "debug"] as const

type TerminalLogLevelFilter = (typeof TERMINAL_LOG_LEVEL_FILTERS)[number]

export interface TerminalLogToolbarProps {
  searchTerm: string
  onSearchTermChange?: (searchTerm: string) => void
  searchPlaceholder?: string
  levelFilter: StructuredLogLevelFilter
  onLevelFilterChange?: (levelFilter: TerminalLogLevelFilter) => void
  levelLabels: Record<TerminalLogLevelFilter, string>
  lineWindow?: {
    value: number
    options: readonly number[]
    onValueChange: (value: number) => void
    label: string
  }
  loading?: boolean
}

export function TerminalLogToolbar({
  searchTerm,
  onSearchTermChange,
  searchPlaceholder,
  levelFilter,
  onLevelFilterChange,
  levelLabels,
  lineWindow,
  loading = false,
}: TerminalLogToolbarProps) {
  return (
    <div data-slot="terminal-log-toolbar" className="flex min-h-10 flex-wrap items-center gap-x-2 gap-y-1 border-b border-border/50 bg-muted/50 px-3 py-1.5 sm:h-10 sm:flex-nowrap sm:gap-2 sm:px-4 sm:py-0">
      <div className="relative min-w-28 flex-1 sm:min-w-0">
        <IconSearch
          aria-hidden="true"
          className="pointer-events-none absolute left-0 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
        />
        <Input
          type="search"
          size="sm"
          value={searchTerm}
          onChange={(event) => onSearchTermChange?.(event.target.value)}
          placeholder={searchPlaceholder}
          disabled={loading}
          aria-hidden={loading || undefined}
          tabIndex={loading ? -1 : undefined}
          className="!rounded-none !border-0 !bg-transparent pl-6 pr-2 text-xs !shadow-none focus-visible:!border-0 focus-visible:!ring-0 focus-visible:!ring-offset-0"
        />
      </div>
      {lineWindow ? (
        <>
          <Separator orientation="vertical" className="hidden h-4 sm:block" />
          <Select
            value={String(lineWindow.value)}
            disabled={loading}
            onValueChange={(value) => {
              const parsed = Number(value)
              if (Number.isInteger(parsed) && lineWindow.options.includes(parsed)) {
                lineWindow.onValueChange(parsed)
              }
            }}
          >
            <SelectTrigger size="sm" aria-label={lineWindow.label} disabled={loading} className="w-20 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {lineWindow.options.map((option) => (
                <SelectItem key={option} value={String(option)}>
                  {option}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </>
      ) : null}
      <Separator orientation="vertical" className="hidden h-4 sm:block" />
      <div data-slot="terminal-log-level-filters" className="flex w-full justify-between gap-1 sm:w-auto sm:justify-start">
        {TERMINAL_LOG_LEVEL_FILTERS.map((value) =>
          loading ? (
            <ActionSkeleton key={value} size="sm" widthClassName="w-12" />
          ) : (
            <Button
              key={value}
              type="button"
              size="sm"
              variant={levelFilter === value ? "default" : "outline"}
              className={cn("w-11 !text-xs", levelFilter !== value && "!text-muted-foreground")}
              onClick={() => onLevelFilterChange?.(value)}
            >
              {levelLabels[value]}
            </Button>
          ),
        )}
      </div>
    </div>
  )
}
