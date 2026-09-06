"use client"

import type { RefObject, UIEventHandler } from "react"

import { ChevronDownIcon } from "@/components/icons"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { cn } from "@/lib/utils"

export const TERMINAL_LOG_BODY_CLASS =
  "relative min-h-0 flex-1 bg-[var(--terminal-log-background)] text-[var(--terminal-log-foreground)]"

export const TERMINAL_LOG_VIEWPORT_CLASS =
  "h-full overflow-auto p-3 font-mono text-xs leading-normal"

export const TERMINAL_LOG_FOOTER_CLASS =
  "relative flex min-h-10 flex-wrap items-center justify-between gap-x-3 gap-y-1 border-t bg-muted/50 px-3 py-1.5 text-xs text-muted-foreground sm:h-10 sm:flex-nowrap sm:px-4 sm:py-0"

interface LiveLogSurfaceProps {
  children?: React.ReactNode
  footer: React.ReactNode
  topRightAction?: React.ReactNode
  viewportRef?: RefObject<HTMLDivElement | null>
  onScroll?: UIEventHandler<HTMLDivElement>
  focusable?: boolean
  showJumpToLatest?: boolean
  onJumpToLatest?: () => void
  jumpToLatestLabel?: string
  viewportClassName?: string
}

export function LiveLogSurface({
  children,
  footer,
  topRightAction,
  viewportRef,
  onScroll,
  focusable = false,
  showJumpToLatest = false,
  onJumpToLatest,
  jumpToLatestLabel,
  viewportClassName,
}: LiveLogSurfaceProps) {
  if (showJumpToLatest && (!onJumpToLatest || !jumpToLatestLabel)) {
    throw new Error("LiveLogSurface requires a command and label when jump-to-latest is visible.")
  }

  return (
    <>
      <div data-slot="live-log-terminal-body" className={TERMINAL_LOG_BODY_CLASS}>
        <div
          ref={viewportRef}
          tabIndex={focusable ? -1 : undefined}
          onScroll={onScroll}
          className={cn(
            TERMINAL_LOG_VIEWPORT_CLASS,
            focusable && "focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring/50",
            viewportClassName
          )}
        >
          {children}
        </div>

        {topRightAction && (
          <div data-slot="live-log-top-right-action" className="absolute right-3 top-3 z-10">
            {topRightAction}
          </div>
        )}

        {showJumpToLatest && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger
                render={(
                  <Button
                    type="button"
                    size="icon"
                    variant="surface"
                    onClick={onJumpToLatest}
                    aria-label={jumpToLatestLabel}
                    className="absolute bottom-4 right-4 radius-round"
                  />
                )}
              >
                <ChevronDownIcon className="h-4 w-4" />
              </TooltipTrigger>
              <TooltipContent side="left" sideOffset={6}>
                {jumpToLatestLabel}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
      </div>

      <div data-slot="live-log-footer" className={TERMINAL_LOG_FOOTER_CLASS}>
        {footer}
      </div>
    </>
  )
}
