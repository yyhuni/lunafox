"use client"

import * as React from "react"
import { Checkbox as CheckboxPrimitive } from "@base-ui/react/checkbox"
import { Check, Minus } from "@/components/icons"

import { cn } from "@/lib/utils"

const Checkbox = React.forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, indeterminate, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    indeterminate={indeterminate}
    className={cn(
      "radius-control-subtle peer inline-flex h-4 w-4 shrink-0 items-center justify-center border border-primary/40 align-middle leading-none ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[checked]:border-interaction-accent data-[checked]:bg-interaction-accent data-[checked]:text-interaction-accent-foreground data-[indeterminate]:border-interaction-accent data-[indeterminate]:bg-interaction-accent data-[indeterminate]:text-interaction-accent-foreground",
      className
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator
      keepMounted
      className={cn("pointer-events-none flex items-center justify-center text-current data-[unchecked]:hidden")}
    >
      {indeterminate ? (
        <Minus className="h-3 w-3" />
      ) : (
        <Check className="h-3 w-3" />
      )}
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
))
Checkbox.displayName = CheckboxPrimitive.Root.displayName

export { Checkbox }
