"use client"

import * as React from "react"

import { Info } from "@/components/icons"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface InlineHelpTooltipProps {
  ariaLabel: string
  children: React.ReactNode
  side?: "top" | "right" | "bottom" | "left"
  sideOffset?: number
}

/**
 * Renders a compact, keyboard-focusable explanation affordance next to a label or heading.
 * The trigger intentionally stays a sibling of buttons and links so it does not create nested interactive controls.
 */
export function InlineHelpTooltip({
  ariaLabel,
  children,
  side = "bottom",
  sideOffset = 8,
}: InlineHelpTooltipProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          render={(
            <span
              aria-label={ariaLabel}
              className="inline-flex shrink-0 cursor-help text-muted-foreground"
              role="img"
              tabIndex={0}
            />
          )}
        >
          <Info className="size-4" aria-hidden="true" />
        </TooltipTrigger>
        <TooltipContent side={side} sideOffset={sideOffset} align="center" className="max-w-sm whitespace-normal text-left">
          {children}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
