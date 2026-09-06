"use client"

import * as React from "react"

import {
  getLoadingOwnerAttributes,
  getLoadingStructureSlotAttributes,
  type LoadingIntent,
  type LoadingLayer,
} from "@/components/shared/loading/loading-owner"
import { cn } from "@/lib/utils"

const CONTENT_HANDOFF_DURATION_MS = 180
const CONTENT_HANDOFF_STABLE_FRAMES = 2

interface ContentHandoffProps extends React.ComponentProps<"div"> {
  owner: string
  layer?: LoadingLayer
  intent?: LoadingIntent
  isLoading: boolean
  skeleton: React.ReactNode
  mountContentWhileLoading?: boolean
  /**
   * Delays handoff until a hidden resolved branch has two stable frames. This
   * is a readiness gate only: it never reserves or mutates visible geometry.
   */
  prepareContentBeforeHandoff?: boolean
  transitionMode?: "crossfade" | "replace"
  skeletonClassName?: string
  contentClassName?: string
}

type ContentHandoffPhase = "loading" | "handoff" | "content"

export function ContentHandoff({
  owner,
  layer = "section",
  intent = "data",
  isLoading,
  skeleton,
  mountContentWhileLoading = false,
  prepareContentBeforeHandoff = false,
  transitionMode = "crossfade",
  skeletonClassName,
  contentClassName,
  className,
  children,
  style,
  ...props
}: ContentHandoffProps) {
  const [phase, setPhase] = React.useState<ContentHandoffPhase>(isLoading ? "loading" : "content")
  const [isHandoffContentReady, setIsHandoffContentReady] = React.useState(!isLoading)
  const [isPreparingHandoff, setIsPreparingHandoff] = React.useState(false)
  const wasLoadingRef = React.useRef(isLoading)
  const skeletonRef = React.useRef<HTMLDivElement | null>(null)
  const contentRef = React.useRef<HTMLDivElement | null>(null)

  const measureStateIntrinsicHeight = React.useCallback((stateNode: HTMLDivElement | null) => {
    if (!stateNode) return 0

    const stateRoots = Array.from(stateNode.children)
    if (stateRoots.length > 0) {
      const rootsWithGeometry = stateRoots
        .map((root) => root.getBoundingClientRect())
        .filter((rect) => rect.height > 0)
      if (rootsWithGeometry.length === 0) return 0

      const top = Math.min(...rootsWithGeometry.map((rect) => rect.top))
      const bottom = Math.max(...rootsWithGeometry.map((rect) => rect.bottom))
      return Math.max(0, bottom - top)
    }

    const hasTextContent = Array.from(stateNode.childNodes).some((node) => (
      node.nodeType === Node.TEXT_NODE && Boolean(node.textContent?.trim())
    ))
    return hasTextContent ? stateNode.scrollHeight : 0
  }, [])

  const measureContentIntrinsicHeight = React.useCallback(() => {
    return measureStateIntrinsicHeight(contentRef.current)
  }, [measureStateIntrinsicHeight])

  const measureSkeletonIntrinsicHeight = React.useCallback(() => {
    return measureStateIntrinsicHeight(skeletonRef.current)
  }, [measureStateIntrinsicHeight])

  const hasCommittedContent = React.useCallback(() => {
    const contentNode = contentRef.current
    if (!contentNode) return false

    return Array.from(contentNode.childNodes).some((node) => {
      if (node.nodeType === 1) return true
      if (node.nodeType === 3) return Boolean(node.textContent?.trim())
      return false
    })
  }, [])

  if (!owner.trim()) {
    throw new Error("ContentHandoff requires a non-empty owner.")
  }

  const shouldMountContentWhileLoading = mountContentWhileLoading || prepareContentBeforeHandoff

  const beginHandoff = React.useCallback((contentIsReady = false) => {
    if (transitionMode === "replace") {
      setIsHandoffContentReady(true)
      setPhase("content")
      return
    }

    setIsHandoffContentReady(contentIsReady)
    setPhase("handoff")
  }, [transitionMode])

  React.useEffect(() => {
    if (isLoading) {
      wasLoadingRef.current = true
      setIsPreparingHandoff(false)
      setIsHandoffContentReady(false)
      setPhase("loading")
      return
    }

    if (wasLoadingRef.current) {
      wasLoadingRef.current = false

      if (prepareContentBeforeHandoff) {
        setIsPreparingHandoff(true)
        setIsHandoffContentReady(false)
        setPhase("loading")
        return
      }

      beginHandoff()
      return
    }

    if (phase === "loading" && !isPreparingHandoff) {
      setIsHandoffContentReady(true)
      setPhase("content")
    }
  }, [beginHandoff, isLoading, isPreparingHandoff, phase, prepareContentBeforeHandoff])

  React.useEffect(() => {
    if (!isPreparingHandoff || isLoading || phase !== "loading") {
      return
    }

    const skeletonNode = skeletonRef.current
    const contentNode = contentRef.current
    if (!skeletonNode || !contentNode) {
      return
    }

    let frameId: number | null = null
    let stableFrameCount = 0
    let previousSignature: string | null = null
    let observedContentRoots: Element[] = []
    let observedSkeletonRoots: Element[] = []

    const cancelFrame = () => {
      if (frameId === null) return
      window.cancelAnimationFrame(frameId)
      frameId = null
    }

    const scheduleStabilityCheck = () => {
      if (frameId !== null) return

      frameId = window.requestAnimationFrame(() => {
        frameId = null

        if (!hasCommittedContent()) {
          stableFrameCount = 0
          previousSignature = null
          return
        }

        const skeletonHeight = measureSkeletonIntrinsicHeight()
        const contentHeight = measureContentIntrinsicHeight()
        if (contentHeight <= 0) {
          stableFrameCount = 0
          previousSignature = null
          return
        }

        const signature = `${skeletonHeight.toFixed(2)}:${contentHeight.toFixed(2)}`
        stableFrameCount = signature === previousSignature ? stableFrameCount + 1 : 1
        previousSignature = signature

        if (stableFrameCount >= CONTENT_HANDOFF_STABLE_FRAMES) {
          setIsPreparingHandoff(false)
          beginHandoff(true)
          return
        }

        scheduleStabilityCheck()
      })
    }

    const invalidateStability = () => {
      stableFrameCount = 0
      previousSignature = null
      scheduleStabilityCheck()
    }

    const observer = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(invalidateStability)

    const syncObservedRoots = () => {
      const syncRoots = (observedRoots: Element[], nextRoots: Element[]) => {
        for (const root of observedRoots) {
          if (!nextRoots.includes(root)) observer?.unobserve(root)
        }
        for (const root of nextRoots) {
          if (!observedRoots.includes(root)) observer?.observe(root)
        }
        return nextRoots
      }

      observedSkeletonRoots = syncRoots(observedSkeletonRoots, Array.from(skeletonNode.children))
      observedContentRoots = syncRoots(observedContentRoots, Array.from(contentNode.children))
    }

    observer?.observe(skeletonNode)
    observer?.observe(contentNode)
    syncObservedRoots()

    const mutationObserver = typeof MutationObserver === "undefined"
      ? null
      : new MutationObserver(() => {
        syncObservedRoots()
        invalidateStability()
      })

    mutationObserver?.observe(contentNode, {
      childList: true,
      subtree: true,
    })

    scheduleStabilityCheck()

    return () => {
      cancelFrame()
      mutationObserver?.disconnect()
      for (const root of observedSkeletonRoots) observer?.unobserve(root)
      for (const root of observedContentRoots) observer?.unobserve(root)
      observer?.disconnect()
    }
  }, [
    beginHandoff,
    hasCommittedContent,
    isLoading,
    isPreparingHandoff,
    measureContentIntrinsicHeight,
    measureSkeletonIntrinsicHeight,
    phase,
  ])

  React.useEffect(() => {
    if (phase !== "handoff") {
      return
    }

    const contentNode = contentRef.current
    if (!contentNode) {
      setIsHandoffContentReady(false)
      return
    }

    const syncContentReady = () => {
      setIsHandoffContentReady(hasCommittedContent() && measureContentIntrinsicHeight() > 0)
    }

    syncContentReady()

    let observedContentRoots: Element[] = []
    const resizeObserver = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(syncContentReady)

    const syncObservedRoots = () => {
      const nextRoots = Array.from(contentNode.children)
      for (const root of observedContentRoots) {
        if (!nextRoots.includes(root)) resizeObserver?.unobserve(root)
      }
      for (const root of nextRoots) {
        if (!observedContentRoots.includes(root)) resizeObserver?.observe(root)
      }
      observedContentRoots = nextRoots
    }

    resizeObserver?.observe(contentNode)
    syncObservedRoots()

    const mutationObserver = typeof MutationObserver === "undefined"
      ? null
      : new MutationObserver(() => {
        syncObservedRoots()
        syncContentReady()
      })
    mutationObserver?.observe(contentNode, {
      childList: true,
      subtree: true,
    })

    return () => {
      mutationObserver?.disconnect()
      for (const root of observedContentRoots) resizeObserver?.unobserve(root)
      resizeObserver?.disconnect()
    }
  }, [hasCommittedContent, measureContentIntrinsicHeight, phase])

  React.useEffect(() => {
    if (phase !== "handoff" || !isHandoffContentReady) {
      return
    }

    const timer = window.setTimeout(() => {
      setPhase("content")
    }, CONTENT_HANDOFF_DURATION_MS)

    return () => window.clearTimeout(timer)
  }, [isHandoffContentReady, phase])

  const showSkeleton = phase === "loading" || phase === "handoff"
  const isMountedLoadingContent = shouldMountContentWhileLoading && phase === "loading"
  const showContent = isMountedLoadingContent || phase === "handoff" || phase === "content"
  // A normal handoff can enter its overlap phase before a lazy child has
  // committed a measurable first frame. Keep that branch inert and hidden
  // until readiness is true, otherwise the unresolved branch can paint above
  // the still-visible skeleton despite the gate remaining active.
  const isContentHiddenUntilReady = isMountedLoadingContent || (
    phase === "handoff" && !isHandoffContentReady
  )
  // An absolutely positioned hidden branch still contributes scrollable
  // overflow. Clip only while it is a readiness probe so a late, tall branch
  // cannot create a loading-stage scrollbar or change the visible width.
  const handoffStyle = isMountedLoadingContent
    ? { ...style, overflow: "clip" }
    : style
  // Hidden content mounts only to emit or verify readiness. Keeping it out of
  // the grid prevents late metadata from changing the visible loading surface.
  const hiddenLoadingContentStyle = isMountedLoadingContent ? {
    position: "absolute" as const,
    top: 0,
    right: 0,
    left: 0,
  } : undefined

  return (
    <div
      {...getLoadingOwnerAttributes({ owner, layer, intent })}
      data-slot="content-handoff"
      data-loading-phase={phase}
      data-loading-content-ready={isHandoffContentReady ? "true" : undefined}
      aria-busy={phase !== "content"}
      className={cn("loading-handoff", className)}
      style={handoffStyle}
      {...props}
    >
      {showSkeleton ? (
        <div
          ref={skeletonRef}
          {...getLoadingStructureSlotAttributes("surface")}
          aria-hidden="true"
          data-loading-structure-state="skeleton"
          className={cn("loading-handoff__skeleton", skeletonClassName)}
        >
          {skeleton}
        </div>
      ) : null}
      {showContent ? (
        <div
          ref={contentRef}
          {...getLoadingStructureSlotAttributes("surface")}
          aria-hidden={isContentHiddenUntilReady ? true : undefined}
          className={cn("loading-handoff__content", contentClassName)}
          data-loading-hidden={isContentHiddenUntilReady ? "true" : undefined}
          data-loading-structure-state="content"
          style={hiddenLoadingContentStyle}
        >
          {children}
        </div>
      ) : null}
    </div>
  )
}
