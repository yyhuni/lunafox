"use client"

import * as React from "react"
import { Popover as PopoverPrimitive } from "@base-ui/react/popover"

import {
  floatingContentMotionClassName,
  floatingSurfaceClassName,
} from "@/lib/ui/overlay-styles"
import { cn } from "@/lib/utils"

type PopoverAnchorProps = Omit<React.HTMLAttributes<HTMLDivElement>, "ref"> & {
  render?: React.ReactElement
  ref?: React.Ref<HTMLElement>
}

type PopoverContentProps = React.ComponentProps<typeof PopoverPrimitive.Popup> &
  Pick<React.ComponentProps<typeof PopoverPrimitive.Positioner>, "align" | "side" | "sideOffset" | "alignOffset" | "collisionPadding" | "collisionBoundary" | "collisionAvoidance" | "sticky" | "positionMethod"> & {
    container?: HTMLElement | null
    minWidth?: "anchor" | "content"
  }

const PopoverAnchorContext = React.createContext<HTMLElement | null>(null)
const PopoverSetAnchorContext = React.createContext<React.Dispatch<React.SetStateAction<HTMLElement | null>> | null>(null)

function Popover({
  children,
  ...props
}: React.ComponentProps<typeof PopoverPrimitive.Root>) {
  const [anchor, setAnchor] = React.useState<HTMLElement | null>(null)

  return (
    <PopoverAnchorContext.Provider value={anchor}>
      <PopoverSetAnchorContext.Provider value={setAnchor}>
        <PopoverPrimitive.Root data-slot="popover" {...props}>
          {children}
        </PopoverPrimitive.Root>
      </PopoverSetAnchorContext.Provider>
    </PopoverAnchorContext.Provider>
  )
}

function PopoverTrigger({
  ...props
}: React.ComponentProps<typeof PopoverPrimitive.Trigger>) {
  return (
    <PopoverPrimitive.Trigger
      data-slot="popover-trigger"
      {...props}
    />
  )
}

function PopoverAnchor({
  render,
  ref,
  ...props
}: PopoverAnchorProps) {
  const setAnchor = React.useContext(PopoverSetAnchorContext)

  const anchorRef = React.useCallback((node: HTMLElement | null) => {
    setAnchor?.(node)

    if (typeof ref === "function") {
      ref(node)
    } else if (ref) {
      ref.current = node
    }
  }, [ref, setAnchor])

  if (render) {
    return React.cloneElement(render, {
      ref: anchorRef,
    } as React.HTMLAttributes<HTMLElement> & { ref: React.Ref<HTMLElement> })
  }

  return (
    <div data-slot="popover-anchor" ref={anchorRef} {...props}>
      {props.children}
    </div>
  )
}

const PopoverContent = React.forwardRef<
  React.ElementRef<typeof PopoverPrimitive.Popup>,
  PopoverContentProps
>(({
  className,
  align = "center",
  side = "bottom",
  sideOffset = 4,
  alignOffset,
  collisionPadding,
  collisionBoundary,
  collisionAvoidance,
  sticky,
  positionMethod,
  container,
  minWidth = "anchor",
  initialFocus,
  finalFocus,
  ...props
}, ref) => {
  const anchor = React.useContext(PopoverAnchorContext)

  return (
    <PopoverPrimitive.Portal container={container ?? undefined}>
      <PopoverPrimitive.Positioner
        data-slot="popover-positioner"
        // Base UI positions this wrapper, so the stacking level must live here.
        className="z-50"
        align={align}
        side={side}
        sideOffset={sideOffset}
        alignOffset={alignOffset}
        collisionPadding={collisionPadding}
        collisionBoundary={collisionBoundary}
        collisionAvoidance={collisionAvoidance}
        sticky={sticky}
        positionMethod={positionMethod}
        anchor={anchor ?? undefined}
      >
        <PopoverPrimitive.Popup
          ref={ref}
          data-slot="popover-content"
          initialFocus={initialFocus}
          finalFocus={finalFocus}
          className={cn(
            floatingSurfaceClassName,
            floatingContentMotionClassName,
            "w-72",
            minWidth === "anchor" && "min-w-[var(--anchor-width)]",
            className
          )}
          {...props}
        />
      </PopoverPrimitive.Positioner>
    </PopoverPrimitive.Portal>
  )
})
PopoverContent.displayName = "PopoverContent"

export { Popover, PopoverTrigger, PopoverContent, PopoverAnchor }
