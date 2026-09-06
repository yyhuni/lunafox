"use client"

import * as React from "react"

export function usePageRefreshTimestamp() {
  const [lastRefreshedAt, setLastRefreshedAt] = React.useState<Date | null>(null)
  const isMountedRef = React.useRef(false)

  React.useEffect(() => {
    isMountedRef.current = true
    // Read the browser clock after mount so SSR and hydration do not render different timestamps.
    setLastRefreshedAt(new Date())

    return () => {
      isMountedRef.current = false
    }
  }, [])

  const markRefreshCompleted = React.useCallback(() => {
    if (isMountedRef.current) setLastRefreshedAt(new Date())
  }, [])

  return {
    isMountedRef,
    lastRefreshedAt,
    markRefreshCompleted,
  }
}
