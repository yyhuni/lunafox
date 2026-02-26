import React from "react"
import { cn } from "@/lib/utils"

interface ResizeHandleProps {
  isResizing: boolean
  onMouseDown: (e: React.MouseEvent) => void
}

export function VerticalResizeHandle({ isResizing, onMouseDown }: ResizeHandleProps) {
  return (
    <button
      type="button"
      className={cn(
        "absolute bottom-0 left-0 right-0 h-2 cursor-row-resize z-50 transition-colors flex items-center justify-center group/handle hover:bg-primary/5",
        isResizing ? "bg-primary/10" : "bg-transparent"
      )}
      onMouseDown={onMouseDown}
      aria-label="Resize vulnerability list and details panels"
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
