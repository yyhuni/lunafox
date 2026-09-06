"use client"

import { useEffect, useRef, useState, type PointerEvent, type PropsWithChildren, type ReactNode } from "react"

export const CANVAS_WIDTH = 920
export const CANVAS_HEIGHT = 600
export const GROUP_X = 252
export const GROUP_WIDTH = 652
export const ENGINE_WIDTH = 230
export const ENGINE_HEIGHT = 48

export type FlowFrame = {
  x: number
  y: number
  width: number
  height: number
}

export type FlowPoint = {
  x: number
  y: number
}

export const serverFrame: FlowFrame = { x: 64, y: 278, width: 132, height: 124 }
export const groupFrames: FlowFrame[] = [
  { x: GROUP_X, y: 28, width: GROUP_WIDTH, height: 208 },
  { x: GROUP_X, y: 240, width: GROUP_WIDTH, height: 164 },
  { x: GROUP_X, y: 412, width: GROUP_WIDTH, height: 172 },
]
export const agentFrames: FlowFrame[] = [
  { x: 302, y: 82, width: 236, height: 72 },
  { x: 302, y: 272, width: 236, height: 72 },
  { x: 302, y: 462, width: 236, height: 72 },
]
export const engineFrames: FlowFrame[][] = [
  [
    { x: 640, y: 62, width: ENGINE_WIDTH, height: ENGINE_HEIGHT },
    { x: 640, y: 116, width: ENGINE_WIDTH, height: ENGINE_HEIGHT },
    { x: 640, y: 170, width: ENGINE_WIDTH, height: ENGINE_HEIGHT },
  ],
  [
    { x: 640, y: 253, width: ENGINE_WIDTH, height: ENGINE_HEIGHT },
    { x: 640, y: 307, width: ENGINE_WIDTH, height: ENGINE_HEIGHT },
  ],
  [
    { x: 640, y: 446, width: ENGINE_WIDTH, height: ENGINE_HEIGHT },
  ],
]

export function centerY(frame: FlowFrame) {
  return frame.y + frame.height / 2
}

export function leftCenter(frame: FlowFrame): FlowPoint {
  return { x: frame.x, y: centerY(frame) }
}

export function rightCenter(frame: FlowFrame): FlowPoint {
  return { x: frame.x + frame.width, y: centerY(frame) }
}

export function leftAnchor(frame: FlowFrame, yOffset = 0): FlowPoint {
  return { x: frame.x, y: centerY(frame) + yOffset }
}

export function rightAnchor(frame: FlowFrame, yOffset = 0): FlowPoint {
  return { x: frame.x + frame.width, y: centerY(frame) + yOffset }
}

export function ArchitectureFlowShell({
  children,
  fillAvailableSpace = false,
}: PropsWithChildren<{ fillAvailableSpace?: boolean }>) {
  return (
    <div className={fillAvailableSpace ? "h-full w-full" : "w-full"}>{children}</div>
  )
}

export function ArchitectureFlowViewport({
  children,
  fillAvailableSpace = false,
  overlay,
}: PropsWithChildren<{ fillAvailableSpace?: boolean; overlay?: ReactNode }>) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [scale, setScale] = useState(1)
  const [viewportSize, setViewportSize] = useState({ height: CANVAS_HEIGHT, width: CANVAS_WIDTH })
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [drag, setDrag] = useState<{
    pointerId: number
    startX: number
    startY: number
    panX: number
    panY: number
  } | null>(null)

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const updateScale = () => {
      const widthScale = container.clientWidth / CANVAS_WIDTH
      const heightScale = container.clientHeight / CANVAS_HEIGHT
      const nextScale = fillAvailableSpace
        ? Math.min(widthScale, heightScale)
        : Math.min(1, widthScale)

      setViewportSize({ height: container.clientHeight, width: container.clientWidth })
      setScale((currentScale) => (
        Math.abs(currentScale - nextScale) > 0.005 ? nextScale : currentScale
      ))
    }

    updateScale()

    if (typeof ResizeObserver === "undefined") {
      window.addEventListener("resize", updateScale)
      return () => window.removeEventListener("resize", updateScale)
    }

    const observer = new ResizeObserver(updateScale)
    observer.observe(container)

    return () => observer.disconnect()
  }, [fillAvailableSpace])

  const contentOffset = fillAvailableSpace
    ? {
      x: Math.max(0, (viewportSize.width - CANVAS_WIDTH * scale) / 2),
      y: Math.max(0, (viewportSize.height - CANVAS_HEIGHT * scale) / 2),
    }
    : { x: 0, y: 0 }

  const beginPan = (event: PointerEvent<HTMLDivElement>) => {
    // Node drags would desynchronize the fixed connector and group geometry.
    if ((event.target as HTMLElement).closest("[data-architecture-flow-node]")) return

    event.currentTarget.setPointerCapture(event.pointerId)
    setDrag({
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      panX: pan.x,
      panY: pan.y,
    })
  }

  const movePan = (event: PointerEvent<HTMLDivElement>) => {
    if (!drag || event.pointerId !== drag.pointerId) return

    setPan({
      x: drag.panX + event.clientX - drag.startX,
      y: drag.panY + event.clientY - drag.startY,
    })
  }

  const endPan = (event: PointerEvent<HTMLDivElement>) => {
    if (!drag || event.pointerId !== drag.pointerId) return

    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    setDrag(null)
  }

  return (
    <div
      ref={containerRef}
      className={drag
        ? `relative w-full cursor-grabbing overflow-hidden select-none${fillAvailableSpace ? " h-full" : ""}`
        : `relative w-full cursor-grab overflow-hidden${fillAvailableSpace ? " h-full" : ""}`}
      data-architecture-flow-viewport
      onPointerCancel={endPan}
      onPointerDown={beginPan}
      onPointerMove={movePan}
      onPointerUp={endPan}
      style={{ height: fillAvailableSpace ? "100%" : CANVAS_HEIGHT * scale }}
    >
      <div
        className="relative"
        style={{
          width: CANVAS_WIDTH,
          height: CANVAS_HEIGHT,
          transform: `translate(${contentOffset.x + pan.x}px, ${contentOffset.y + pan.y}px) scale(${scale})`,
          transformOrigin: "top left",
        }}
      >
        {children}
      </div>
      {overlay ? (
        <div className="pointer-events-none absolute inset-x-0 bottom-3">
          {overlay}
        </div>
      ) : null}
    </div>
  )
}
