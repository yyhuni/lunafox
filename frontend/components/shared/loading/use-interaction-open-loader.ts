"use client"

import * as React from "react"

export function useInteractionOpenLoader<TKey extends string>() {
  const [pendingInteraction, setPendingInteraction] = React.useState<TKey | null>(null)
  const pendingInteractionRef = React.useRef<TKey | null>(null)

  const cancelPending = React.useCallback((key?: TKey) => {
    if (key === undefined || pendingInteractionRef.current === key) {
      pendingInteractionRef.current = null
    }

    setPendingInteraction((current) => {
      if (key === undefined || current === key) {
        return null
      }
      return current
    })
  }, [])

  const openAfterLoad = React.useCallback(async (
    key: TKey,
    load: () => Promise<unknown>,
    onLoaded: () => void,
    onError?: (error: unknown) => void,
  ) => {
    pendingInteractionRef.current = key
    setPendingInteraction(key)

    try {
      await load()

      const shouldOpen = pendingInteractionRef.current === key

      if (shouldOpen) {
        pendingInteractionRef.current = null
        setPendingInteraction(null)
        onLoaded()
      }
    } catch (error) {
      if (pendingInteractionRef.current === key) {
        pendingInteractionRef.current = null
        setPendingInteraction(null)
      }
      onError?.(error)
    }
  }, [])

  return {
    pendingInteraction,
    cancelPending,
    openAfterLoad,
  }
}
