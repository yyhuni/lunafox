"use client"

import * as React from "react"

const MIN_REFRESH_FEEDBACK_MS = 300

export function useRefreshFeedback(isRefreshing: boolean) {
  const [manualRefreshPending, setManualRefreshPending] = React.useState(false)
  const startedAtRef = React.useRef<number | null>(null)

  React.useEffect(() => {
    if (!manualRefreshPending) return

    const startedAt = startedAtRef.current ?? Date.now()
    const remainingMs = Math.max(MIN_REFRESH_FEEDBACK_MS - (Date.now() - startedAt), 0)
    const timeoutId = window.setTimeout(() => {
      startedAtRef.current = null
      setManualRefreshPending(false)
    }, remainingMs)

    return () => window.clearTimeout(timeoutId)
  }, [manualRefreshPending])

  const beginManualRefresh = React.useCallback(() => {
    startedAtRef.current = Date.now()
    setManualRefreshPending(true)
  }, [])

  return {
    isVisible: isRefreshing || manualRefreshPending,
    beginManualRefresh,
  }
}
