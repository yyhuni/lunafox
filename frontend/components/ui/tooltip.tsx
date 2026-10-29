"use client"

import * as React from "react"
import { Tooltip as TooltipPrimitive } from "@base-ui/react/tooltip"

import {
  floatingContentMotionClassName,
  sidebarHoverTiming,
  tooltipArrowClassName,
  tooltipSurfaceClassName,
} from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"

type TooltipContentProps = React.ComponentProps<typeof TooltipPrimitive.Popup> &
  Pick<React.ComponentProps<typeof TooltipPrimitive.Positioner>, "align" | "side" | "sideOffset" | "alignOffset" | "collisionPadding" | "collisionBoundary" | "collisionAvoidance" | "sticky" | "positionMethod"> & {
    container?: React.ComponentProps<typeof TooltipPrimitive.Portal>["container"]
    showArrow?: boolean
  }

const TOOLTIP_OPEN_DELAY_MS = 400
const TOOLTIP_CLOSE_DELAY_MS = 0
const TOOLTIP_ADJACENT_SWITCH_TIMEOUT_MS = 400

type TooltipProviderProps = Omit<
  React.ComponentProps<typeof TooltipPrimitive.Provider>,
  "delay" | "closeDelay" | "timeout"
>

type TooltipTriggerProps = Omit<
  React.ComponentProps<typeof TooltipPrimitive.Trigger>,
  "delay" | "closeDelay" | "timeout"
>

function TooltipProvider(props: TooltipProviderProps) {
  return (
    <TooltipPrimitive.Provider
      {...props}
      data-slot="tooltip-provider"
      delay={TOOLTIP_OPEN_DELAY_MS}
      closeDelay={TOOLTIP_CLOSE_DELAY_MS}
      timeout={TOOLTIP_ADJACENT_SWITCH_TIMEOUT_MS}
    />
  )
}

function SidebarTooltipProvider(props: TooltipProviderProps) {
  return (
    <TooltipPrimitive.Provider
      {...props}
      data-slot="sidebar-tooltip-provider"
      delay={sidebarHoverTiming.openDelay}
      closeDelay={sidebarHoverTiming.closeDelay}
      timeout={sidebarHoverTiming.adjacentSwitchTimeout}
    />
  )
}

function Tooltip({
  ...props
}: React.ComponentProps<typeof TooltipPrimitive.Root>) {
  return <TooltipPrimitive.Root data-slot="tooltip" {...props} />
}

function TooltipTrigger(props: TooltipTriggerProps) {
  return (
    <TooltipPrimitive.Trigger
      data-slot="tooltip-trigger"
      {...props}
    />
  )
}

function TooltipContent({
  className,
  side = "top",
  sideOffset = 4,
  align = "center",
  alignOffset = 0,
  collisionPadding,
  collisionBoundary,
  collisionAvoidance,
  sticky,
  positionMethod,
  container,
  showArrow = true,
  children,
  ...props
}: TooltipContentProps) {
  return (
    <TooltipPrimitive.Portal container={container}>
      <TooltipPrimitive.Positioner
        data-slot="tooltip-positioner"
        align={align}
        side={side}
        sideOffset={sideOffset}
        alignOffset={alignOffset}
        collisionPadding={collisionPadding}
        collisionBoundary={collisionBoundary}
        collisionAvoidance={collisionAvoidance}
        sticky={sticky}
        positionMethod={positionMethod}
        className="isolate z-50"
      >
        <TooltipPrimitive.Popup
          data-slot="tooltip-content"
          className={cn(
            tooltipSurfaceClassName,
            floatingContentMotionClassName,
            className
          )}
          {...props}
        >
          {children}
          {showArrow ? <TooltipPrimitive.Arrow className={tooltipArrowClassName} /> : null}
        </TooltipPrimitive.Popup>
      </TooltipPrimitive.Positioner>
    </TooltipPrimitive.Portal>
  )
}

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider, SidebarTooltipProvider }
