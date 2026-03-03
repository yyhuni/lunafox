import { useCallback, useEffect, useRef, useState } from "react"

interface UseHorizontalResizeOptions {
  min?: number
  max?: number
}

export function useHorizontalResize(
  initialWidthPercentage = 30,
  options: UseHorizontalResizeOptions = {}
) {
  const { min = 20, max = 70 } = options
  const [width, setWidth] = useState(initialWidthPercentage)
  const [isResizing, setIsResizing] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const startResizing = useCallback(() => setIsResizing(true), [])
  const stopResizing = useCallback(() => setIsResizing(false), [])

  useEffect(() => {
    const handlePointerMove = (e: PointerEvent) => {
      if (!isResizing || !containerRef.current) return

      const containerRect = containerRef.current.getBoundingClientRect()
      if (containerRect.width <= 0) return

      const newWidth = ((e.clientX - containerRect.left) / containerRect.width) * 100
      setWidth(Math.min(Math.max(newWidth, min), max))
    }

    if (isResizing) {
      window.addEventListener("pointermove", handlePointerMove)
      window.addEventListener("pointerup", stopResizing)
      window.addEventListener("pointercancel", stopResizing)
      document.body.style.cursor = "col-resize"
      document.body.style.userSelect = "none"
    } else {
      document.body.style.cursor = ""
      document.body.style.userSelect = ""
    }

    return () => {
      window.removeEventListener("pointermove", handlePointerMove)
      window.removeEventListener("pointerup", stopResizing)
      window.removeEventListener("pointercancel", stopResizing)
      document.body.style.cursor = ""
      document.body.style.userSelect = ""
    }
  }, [isResizing, max, min, stopResizing])

  return { width, isResizing, startResizing, containerRef }
}

