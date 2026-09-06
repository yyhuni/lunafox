import React from "react"
import { cn } from "@/lib/utils"

interface ResizeHandleProps {
  isResizing: boolean
  onPointerDown: (e: React.PointerEvent) => void
  ariaLabel: string
}

export function VerticalResizeHandle({
  isResizing,
  onPointerDown,
  ariaLabel,
}: ResizeHandleProps) {
  return (
    <button
      type="button"
      className={cn(
        "flex h-2 shrink-0 cursor-row-resize touch-none items-center justify-center rounded-sm transition-colors hover:bg-primary/5",
        isResizing ? "bg-primary/10" : "bg-transparent"
      )}
      onPointerDown={onPointerDown}
      aria-label={ariaLabel}
    >
      <div
        className={cn(
          "h-1 w-10 rounded-full transition-colors",
          isResizing ? "bg-primary" : "bg-border"
        )}
      />
    </button>
  )
}
