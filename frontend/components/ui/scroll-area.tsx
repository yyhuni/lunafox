"use client"

import * as React from "react"
import { ScrollArea as ScrollAreaPrimitive } from "@base-ui/react/scroll-area"

import { cn } from "@/lib/utils"

type ScrollAreaType = "auto" | "always" | "scroll" | "hover"

type ScrollAreaStyle = React.CSSProperties & {
  "--scroll-hide-delay"?: string
}

type ScrollAreaProps = ScrollAreaPrimitive.Root.Props & {
  type?: ScrollAreaType
  scrollHideDelay?: number
  viewportClassName?: string
  contentClassName?: string
  contentStyle?: React.CSSProperties
}

const ScrollArea = React.forwardRef<
  React.ElementRef<typeof ScrollAreaPrimitive.Root>,
  ScrollAreaProps
>(
  (
    {
      className,
      children,
      viewportClassName,
      contentClassName,
      contentStyle,
      type: scrollAreaType = "scroll",
      scrollHideDelay = 600,
      style,
      ...props
    },
    ref
  ) => (
    <ScrollAreaPrimitive.Root
      ref={ref}
      data-scroll-type={scrollAreaType}
      className={cn("relative overflow-hidden", className)}
      style={
        {
          "--scroll-hide-delay": `${scrollHideDelay}ms`,
          ...style,
        } as ScrollAreaStyle
      }
      {...props}
    >
      <ScrollAreaPrimitive.Viewport className={cn("h-full w-full rounded-[inherit]", viewportClassName)}>
        <ScrollAreaPrimitive.Content className={contentClassName} style={contentStyle}>{children}</ScrollAreaPrimitive.Content>
      </ScrollAreaPrimitive.Viewport>
      <ScrollBar scrollAreaType={scrollAreaType} />
      <ScrollAreaPrimitive.Corner />
    </ScrollAreaPrimitive.Root>
  )
)
ScrollArea.displayName = "ScrollArea"

type ScrollBarProps = ScrollAreaPrimitive.Scrollbar.Props & {
  scrollAreaType?: ScrollAreaType
}

const ScrollBar = React.forwardRef<
  React.ElementRef<typeof ScrollAreaPrimitive.Scrollbar>,
  ScrollBarProps
>(({ className, orientation = "vertical", scrollAreaType = "scroll", keepMounted, ...props }, ref) => (
  <ScrollAreaPrimitive.Scrollbar
    ref={ref}
    orientation={orientation}
    keepMounted={keepMounted ?? scrollAreaType === "always"}
    className={cn(
      "group/scrollbar flex touch-none select-none opacity-0 transition-[opacity,background-color] delay-(--scroll-hide-delay) duration-200 data-[has-overflow-y]:[--scrollbar-axis:vertical] data-[has-overflow-x]:[--scrollbar-axis:horizontal] data-[hovering]:delay-0 data-[scrolling]:delay-0 data-[scrolling]:opacity-100",
      scrollAreaType === "always" &&
        "data-[has-overflow-y]:opacity-100 data-[has-overflow-x]:opacity-100",
      scrollAreaType === "hover" && "data-[hovering]:opacity-100",
      orientation === "vertical" &&
        "h-full w-2 border-l border-l-transparent p-px",
      orientation === "horizontal" &&
        "h-2 flex-col border-t border-t-transparent p-px",
      className
    )}
    {...props}
  >
    <ScrollAreaPrimitive.Thumb className="relative flex-1 rounded-full bg-border/70 transition-colors duration-200 group-hover/scrollbar:bg-border active:bg-foreground/30" />
  </ScrollAreaPrimitive.Scrollbar>
))
ScrollBar.displayName = "ScrollBar"

export { ScrollArea, ScrollBar }
