"use client"

import * as React from "react"

import { ContentHandoff } from "@/components/shared/loading/content-handoff"
import type { LoadingIntent, LoadingLayer } from "@/components/shared/loading/loading-owner"

export interface HiddenReadinessRouteBoundaryRenderProps {
  onReady: () => void
  deferInitialSkeleton: boolean
}

export interface HiddenReadinessRouteBoundaryProps extends Omit<React.ComponentProps<"div">, "children"> {
  owner: string
  skeleton: React.ReactNode
  children: (props: HiddenReadinessRouteBoundaryRenderProps) => React.ReactNode
  layer?: LoadingLayer
  intent?: LoadingIntent
  skeletonClassName?: string
  contentClassName?: string
  initiallyDeferSkeleton?: boolean
  transitionMode?: "crossfade" | "replace"
}

export function HiddenReadinessRouteBoundary({
  owner,
  skeleton,
  layer = "route",
  intent = "route",
  className,
  skeletonClassName,
  contentClassName,
  initiallyDeferSkeleton = true,
  transitionMode = "crossfade",
  children,
  ...props
}: HiddenReadinessRouteBoundaryProps) {
  // Child-tab navigation changes route intent, not the first-entry shell owner.
  const [shouldDeferInitialSkeleton] = React.useState(() => initiallyDeferSkeleton)
  const [isReady, setIsReady] = React.useState(() => !shouldDeferInitialSkeleton)
  const deferInitialSkeleton = shouldDeferInitialSkeleton && !isReady
  const handleReady = React.useCallback(() => {
    setIsReady(true)
  }, [])

  return (
    <ContentHandoff
      owner={owner}
      layer={layer}
      intent={intent}
      isLoading={deferInitialSkeleton}
      skeleton={skeleton}
      mountContentWhileLoading
      prepareContentBeforeHandoff
      transitionMode={transitionMode}
      className={className}
      skeletonClassName={skeletonClassName}
      contentClassName={contentClassName}
      {...props}
    >
      {children({ onReady: handleReady, deferInitialSkeleton })}
    </ContentHandoff>
  )
}
