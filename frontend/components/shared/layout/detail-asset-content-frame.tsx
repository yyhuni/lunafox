import * as React from "react"

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
      className={cn("px-4 pb-4 md:pb-6 lg:px-6", className)}
    />
  )
}
