"use client"

import * as React from "react"
import { Progress as ProgressPrimitive } from "@base-ui/react/progress"

import { cn } from "@/lib/utils"

type ProgressProps = Omit<React.ComponentProps<typeof ProgressPrimitive.Root>, "value"> & {
  indicatorClassName?: string
  value?: number | null
}

function Progress({
  className,
  value,
  indicatorClassName,
  ...props
}: ProgressProps) {
  const progressValue = value ?? 0

  return (
    <ProgressPrimitive.Root
      data-slot="progress"
      value={progressValue}
      className={cn(
        "radius-pill relative h-2 w-full overflow-hidden bg-primary/20",
        className
      )}
      {...props}
    >
      <ProgressPrimitive.Indicator
        data-slot="progress-indicator"
        className={cn("h-full w-full flex-1 origin-left bg-primary transition-[transform,background-color]", indicatorClassName)}
        // Base UI already derives a width from `value`; keep the fill at full width
        // so this transform remains the single percentage calculation and animation.
        style={{ width: "100%", transform: `scaleX(${progressValue / 100})` }}
      />
    </ProgressPrimitive.Root>
  )
}

export { Progress }
