"use client"

import * as React from "react"

import { X } from "@/components/icons"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { getStatusToneTextClass } from "@/lib/status-config"
import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

export interface SelectedRowActionBarAction {
  key: string
  label: React.ReactNode
  icon?: React.ComponentType<{ className?: string; "aria-hidden"?: "true" }>
  tone?: "default" | "success" | "muted" | "destructive"
  group?: string
  disabled?: boolean
  /** Explain why a visible action is currently unavailable. */
  disabledReason?: React.ReactNode
  onClick: () => void
}

interface SelectedRowActionBarProps {
  selectedCount: number
  ariaLabel: string
  countLabel: React.ReactNode
  actions: SelectedRowActionBarAction[]
  onClearSelection?: () => void
  clearSelectionLabel?: string
  className?: string
}

function getActionClassName(tone: SelectedRowActionBarAction["tone"]) {
  if (tone === "destructive") {
    return "border-0 bg-transparent px-3 text-destructive shadow-none hover:bg-destructive/10 hover:text-destructive active:bg-destructive/15 dark:bg-transparent dark:hover:bg-destructive/15"
  }

  // Keep contextual actions visually quiet until hover, matching sidebar navigation feedback.
  return "border-0 bg-transparent px-3 text-sidebar-foreground/65 shadow-none hover:bg-sidebar-accent hover:text-sidebar-accent-foreground dark:bg-transparent dark:hover:bg-sidebar-accent"
}

function getIconClassName(tone: SelectedRowActionBarAction["tone"]) {
  if (tone === "success" || tone === "muted") {
    return cn("size-3.5", getStatusToneTextClass(tone))
  }

  return "size-3.5"
}

function ActionBarSeparator() {
  return <div className="bg-border h-5 w-px" aria-hidden="true" />
}

function renderCountLabel(countLabel: React.ReactNode, selectedCount: number) {
  if (typeof countLabel !== "string" && typeof countLabel !== "number") {
    return countLabel
  }

  const label = String(countLabel)
  const count = String(selectedCount)
  const countIndex = label.indexOf(count)

  if (countIndex < 0) {
    return label
  }

  const prefix = label.slice(0, countIndex)
  const suffix = label.slice(countIndex + count.length)

  return (
    <>
      {prefix}
      <span className="text-highlight">{count}</span>
      {suffix}
    </>
  )
}

function SelectedActionButton({ action }: { action: SelectedRowActionBarAction }) {
  const Icon = action.icon
  const button = (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={action.onClick}
      disabled={action.disabled}
      className={getActionClassName(action.tone)}
    >
      {Icon && <Icon className={getIconClassName(action.tone)} aria-hidden="true" />}
      {action.label}
    </Button>
  )

  if (!action.disabled || !action.disabledReason) {
    return button
  }

  const nativeReason = typeof action.disabledReason === "string" ? action.disabledReason : undefined
  return (
    <Tooltip>
      <TooltipTrigger
        render={(
          <span
            tabIndex={0}
            title={nativeReason}
            className="inline-flex"
            aria-label={typeof action.label === "string" ? action.label : undefined}
          />
        )}
      >
        {button}
      </TooltipTrigger>
      <TooltipContent>{action.disabledReason}</TooltipContent>
    </Tooltip>
  )
}

export function SelectedRowActionBar({
  selectedCount,
  ariaLabel,
  countLabel,
  actions,
  onClearSelection,
  clearSelectionLabel,
  className,
}: SelectedRowActionBarProps) {
  const visibleActions = actions.filter((action) => Boolean(action.onClick))

  if (selectedCount <= 0 || visibleActions.length === 0) {
    return null
  }

  return (
    <div
      className={cn(
        "-translate-x-1/2 animate-in bottom-6 duration-200 fade-in fixed left-1/2 slide-in-from-bottom-4 z-50",
        className
      )}
    >
      <div
        className="radius-pill border border-border bg-popover/95 text-popover-foreground flex items-center gap-3 px-4 py-2 shadow-lg"
        role="toolbar"
        aria-label={ariaLabel}
      >
        <div className="flex items-center">
          <span className={cn(textRole.bodyStrong, "whitespace-nowrap tabular-nums")}>
            {renderCountLabel(countLabel, selectedCount)}
          </span>
        </div>
        <ActionBarSeparator />
        <TooltipProvider delay={300}>
          {visibleActions.map((action, index) => {
            const previousAction = visibleActions[index - 1]
            const showGroupSeparator = index > 0 && action.group !== previousAction?.group

            return (
              <React.Fragment key={action.key}>
                {showGroupSeparator && <ActionBarSeparator />}
                <SelectedActionButton action={action} />
              </React.Fragment>
            )
          })}
        </TooltipProvider>
        {onClearSelection && clearSelectionLabel && (
          <>
            <ActionBarSeparator />
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={onClearSelection}
              className="text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              aria-label={clearSelectionLabel}
            >
              <X className="size-4" aria-hidden="true" />
            </Button>
          </>
        )}
      </div>
    </div>
  )
}
