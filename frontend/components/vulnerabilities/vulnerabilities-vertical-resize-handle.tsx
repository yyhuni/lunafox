import React from "react"
import { cn } from "@/lib/utils"

interface ResizeHandleProps {
  isResizing: boolean
  onPointerDown: (e: React.PointerEvent) => void
  ariaLabel: string
}

export function VerticalResizeHandle({ isResizing, onPointerDown, ariaLabel }: ResizeHandleProps) {
  return (
    <button
      type="button"
      className={cn(
        "absolute bottom-0 left-0 right-0 h-3 md:h-2 cursor-row-resize z-50 transition-colors flex items-center justify-center group/handle hover:bg-primary/5 touch-none",
        isResizing ? "bg-primary/10" : "bg-transparent"
      )}
      onPointerDown={onPointerDown}
      aria-label={ariaLabel}
    >
      <div
        className={cn(
          "w-10 h-1 rounded-full transition-colors opacity-0 group-hover/handle:opacity-100",
          isResizing ? "bg-primary opacity-100" : "bg-border"
        )}
      />
    </button>
  )
}
