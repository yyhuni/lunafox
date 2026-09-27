"use client"

import * as React from "react"

import { textRole } from "@/lib/typography"
import { cn } from "@/lib/utils"

function Label({
  className,
  ...props
}: React.ComponentProps<"label">) {
  return (
    <label
      data-slot="label"
      className={cn(
        "flex items-center gap-2 leading-4 select-none group-data-[disabled=true]:pointer-events-none group-data-[disabled=true]:opacity-50 peer-disabled:cursor-not-allowed peer-disabled:opacity-50",
        textRole.compactSectionTitle,
        className
      )}
      {...props}
    />
  )
}

export { Label }
