"use client"

import * as React from "react"

export interface DetailShellReadyContextValue {
  onReady: () => void
  deferInitialSkeleton: boolean
}

const DetailShellReadyContext = React.createContext<DetailShellReadyContextValue | null>(null)

export function DetailShellReadyProvider({
  value,
  children,
}: {
  value: DetailShellReadyContextValue
  children: React.ReactNode
}) {
  return (
    <DetailShellReadyContext.Provider value={value}>
      {children}
    </DetailShellReadyContext.Provider>
  )
}

export function useDetailShellReadyContext() {
  return React.useContext(DetailShellReadyContext)
}

export function useDetailShellReadySignal(shouldSignalReady: boolean) {
  const handoff = useDetailShellReadyContext()

  React.useEffect(() => {
    if (!handoff?.onReady || !shouldSignalReady) {
      return
    }

    let firstFrame = 0
    let secondFrame = 0

    firstFrame = window.requestAnimationFrame(() => {
      secondFrame = window.requestAnimationFrame(() => {
        handoff.onReady()
      })
    })

    return () => {
      if (firstFrame) {
        window.cancelAnimationFrame(firstFrame)
      }
      if (secondFrame) {
        window.cancelAnimationFrame(secondFrame)
      }
    }
  }, [handoff, shouldSignalReady])

  return handoff
}
