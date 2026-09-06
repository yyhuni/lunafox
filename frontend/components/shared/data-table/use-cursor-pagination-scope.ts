"use client"

import * as React from "react"

/**
 * Opaque cursor tokens are bound to their parent collection. Returning this
 * signal lets a state owner derive a first-page request during the scope-change
 * render, before its effect commits the new token cache.
 */
export function useCursorPaginationScopeChange(scopeKey: string) {
  const previousScopeKeyRef = React.useRef(scopeKey)
  const hasScopeChanged = previousScopeKeyRef.current !== scopeKey

  React.useEffect(() => {
    previousScopeKeyRef.current = scopeKey
  }, [scopeKey])

  return hasScopeChanged
}
