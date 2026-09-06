"use client"

import * as React from "react"
import { PreviewCard as HoverCardPrimitive } from "@base-ui/react/preview-card"

import {
  floatingContentMotionClassName,
  floatingSurfaceClassName,
} from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"

type HoverCardContentProps = React.ComponentProps<typeof HoverCardPrimitive.Popup> &
  Pick<React.ComponentProps<typeof HoverCardPrimitive.Positioner>, "align" | "side" | "sideOffset" | "alignOffset" | "collisionPadding" | "collisionBoundary" | "collisionAvoidance" | "sticky" | "positionMethod"> & {
    container?: React.ComponentProps<typeof HoverCardPrimitive.Portal>["container"]
  }

function HoverCard({
  ...props
}: React.ComponentProps<typeof HoverCardPrimitive.Root>) {
  return <HoverCardPrimitive.Root data-slot="hover-card" {...props} />
}

function HoverCardTrigger({
  ...props
}: React.ComponentProps<typeof HoverCardPrimitive.Trigger>) {
  return (
    <HoverCardPrimitive.Trigger
      data-slot="hover-card-trigger"
      {...props}
    />
  )
}

const HoverCardContent = React.forwardRef<
  React.ElementRef<typeof HoverCardPrimitive.Popup>,
  HoverCardContentProps
>(({
  className,
  align = "center",
  sideOffset = 4,
  side,
  alignOffset,
  collisionPadding,
  collisionBoundary,
  collisionAvoidance,
  sticky,
  positionMethod,
  container,
  ...props
}, ref) => (
  <HoverCardPrimitive.Portal container={container}>
    <HoverCardPrimitive.Positioner
      data-slot="hover-card-positioner"
      align={align}
      side={side}
      sideOffset={sideOffset}
      alignOffset={alignOffset}
      collisionPadding={collisionPadding}
      collisionBoundary={collisionBoundary}
      collisionAvoidance={collisionAvoidance}
      sticky={sticky}
      positionMethod={positionMethod}
    >
      <HoverCardPrimitive.Popup
        ref={ref}
        data-slot="hover-card-content"
        className={cn(
          floatingSurfaceClassName,
          floatingContentMotionClassName,
          "w-64",
          className
        )}
        {...props}
      />
    </HoverCardPrimitive.Positioner>
  </HoverCardPrimitive.Portal>
))
HoverCardContent.displayName = "HoverCardContent"

export { HoverCard, HoverCardTrigger, HoverCardContent }
