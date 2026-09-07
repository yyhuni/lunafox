"use client"

import type * as React from "react"

import {
  getLoadingOwnerAttributes,
  type LoadingIntent,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"
import { cn } from "@/lib/utils"

interface ContentRevealProps extends React.ComponentProps<"div"> {
  owner: string
  layer?: LoadingLayer
  intent?: LoadingIntent
}

export function ContentReveal({
  owner,
  layer = "route",
  intent = "route",
  className,
  children,
  ...props
}: ContentRevealProps) {
  if (!owner.trim()) {
    throw new Error("ContentReveal requires a non-empty owner.")
  }

  return (
    <div
      {...getLoadingOwnerAttributes({ owner, layer, intent })}
      data-slot="content-reveal"
      data-loading-phase="content"
      className={cn("loading-content-reveal", className)}
      {...props}
    >
      {children}
    </div>
  )
}
