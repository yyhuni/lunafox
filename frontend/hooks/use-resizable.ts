import { useState, useCallback, useEffect } from "react"

interface UseResizableProps {
  initialSize: number
  minSize?: number
  maxSize?: number
  direction?: "horizontal" | "vertical"
}

export function useResizable({ initialSize }: UseResizableProps) {
  const [size] = useState(initialSize)
  const [isResizing, setIsResizing] = useState(false)

  const startResizing = useCallback(() => {
    setIsResizing(true)
  }, [])

  const stopResizing = useCallback(() => {
    setIsResizing(false)
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return

      // Prevent text selection during resize
      e.preventDefault()

      // Calculate new size
      // Note: This assumes the resize handle is at the bottom of the top panel (vertical)
      // For a truly generic hook, we'd need more context. 
      // Let's simplify: Return the event handler and let the component manage the calculation logic 
      // or providing the container ref.
    }

    if (isResizing) {
      window.addEventListener("mousemove", handleMouseMove)
      window.addEventListener("mouseup", stopResizing)
    }

    return () => {
      window.removeEventListener("mousemove", handleMouseMove)
      window.removeEventListener("mouseup", stopResizing)
    }
  }, [isResizing, stopResizing])

  return { size, isResizing, startResizing }
}
