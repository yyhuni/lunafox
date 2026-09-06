import * as React from "react"

interface DeferredInteractionMountOptions {
  preload?: () => void
  preloadWhen?: boolean
  unmountDelayMs?: number
}

export const deferredInteractionUnmountDelayMs = 240

export function useDeferredInteractionMount(
  active: boolean,
  options: DeferredInteractionMountOptions = {}
) {
  const { preload, preloadWhen = false, unmountDelayMs } = options
  const [shouldMount, setShouldMount] = React.useState(active || preloadWhen)
  const hasPreloadedRef = React.useRef(false)

  React.useEffect(() => {
    if (active || preloadWhen) {
      setShouldMount(true)
    }

    if ((active || preloadWhen) && preload && !hasPreloadedRef.current) {
      hasPreloadedRef.current = true
      preload()
    }
  }, [active, preload, preloadWhen])

  React.useEffect(() => {
    if (active || preloadWhen || unmountDelayMs === undefined) {
      return undefined
    }

    const timeout = window.setTimeout(() => {
      setShouldMount(false)
    }, unmountDelayMs)

    return () => {
      window.clearTimeout(timeout)
    }
  }, [active, preloadWhen, unmountDelayMs])

  return shouldMount
}
