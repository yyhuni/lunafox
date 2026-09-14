import * as React from "react"

import { COMPACT_CONTENT_GUTTER_CLASS } from "@/components/shared/layout/page-shell-density"
import { cn } from "@/lib/utils"

export type DetailAssetContentFrameProps = React.ComponentProps<"div">

export function DetailAssetContentFrame({
  className,
  ...props
}: DetailAssetContentFrameProps) {
  return (
    <div
      {...props}
      data-slot="detail-asset-content-frame"
      className={cn(COMPACT_CONTENT_GUTTER_CLASS, "pb-3", className)}
    />
  )
}
