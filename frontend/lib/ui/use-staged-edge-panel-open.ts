"use client"

import * as React from "react"

/**
 * Lets a dynamically mounted controlled edge panel commit its closed transform
 * before opening. Without this boundary, a first render with `open=true`
 * skips the shared Sheet or Drawer enter transition.
 */
export function useStagedEdgePanelOpen(open: boolean | undefined) {
  const [readyToOpen, setReadyToOpen] = React.useState(false)

  React.useEffect(() => {
    if (open !== true) {
      setReadyToOpen(false)
      return undefined
    }

    if (window.matchMedia?.("(prefers-reduced-motion: reduce)").matches) {
      setReadyToOpen(true)
      return undefined
    }

    const frame = window.requestAnimationFrame(() => {
      setReadyToOpen(true)
    })

    return () => {
      window.cancelAnimationFrame(frame)
    }
  }, [open])

  return open === undefined ? undefined : open && readyToOpen
}
