"use client"

import type { Header } from "@tanstack/react-table"
import { cn } from "@/lib/utils"

interface ColumnResizerProps<TData> {
  header: Header<TData, unknown>
  className?: string
}

/**
 * Unified column width adjustment handle component
 * 
 * Design specifications:
 * - Clickable area width: 8px (w-2)
 * - Positioned inside the right edge of the column
 * - TableHead needs to add pr-2 to reserve space for resizer
 * - Visual indicator line width: 2px (w-0.5)
 * - Height: 100% fills the table header
 * - Supports mouse and touch events
 * - Double-click to reset column width
 * - Show highlight indicator line on hover
 */
export function ColumnResizer<TData>({ header, className }: ColumnResizerProps<TData>) {
  if (!header.column.getCanResize()) {
    return null
  }

  const resizeHandler = header.getResizeHandler()

  return (
    <div
      onMouseDown={(e) => {
        e.preventDefault()
        e.stopPropagation()
        resizeHandler(e)
      }}
      onTouchStart={(e) => {
        e.stopPropagation()
        resizeHandler(e)
      }}
      onDoubleClick={(e) => {
        e.preventDefault()
        e.stopPropagation()
        header.column.resetSize()
      }}
      className={cn(
        // Clickable area: 8px wide, positioned at column right edge
        "group absolute right-0 top-0 h-full w-2 cursor-col-resize select-none touch-none z-10",
        "flex items-center justify-center",
        className
      )}
    >
      {/* Visual indicator line: 2px wide, 80% height, only show on hover or drag */}
      <div 
        className={cn(
          "w-0.5 h-4/5 rounded-full transition-[opacity,background-color]",
          header.column.getIsResizing() 
            ? "bg-primary opacity-100" 
            : "bg-primary/50 opacity-0 group-hover:opacity-100"
        )} 
      />
    </div>
  )
}
