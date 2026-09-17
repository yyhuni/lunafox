"use client"

import * as React from "react"
import { usePathname } from "next/navigation"

import { cn } from "@/lib/utils"

interface RouteContentTransitionContextValue {
  navigationId: number
}

const RouteContentTransitionContext =
  React.createContext<RouteContentTransitionContextValue>({ navigationId: 0 })

const useIsomorphicLayoutEffect =
  typeof window === "undefined" ? React.useEffect : React.useLayoutEffect

export function RouteContentTransitionProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname()
  const previousPathnameRef = React.useRef(pathname)
  const [navigationId, setNavigationId] = React.useState(0)

  useIsomorphicLayoutEffect(() => {
    if (previousPathnameRef.current === pathname) {
      return
    }

    previousPathnameRef.current = pathname
    setNavigationId((currentNavigationId) => currentNavigationId + 1)
  }, [pathname])

  return (
    <RouteContentTransitionContext.Provider value={{ navigationId }}>
      {children}
    </RouteContentTransitionContext.Provider>
  )
}

export function RouteContentTransition({
  className,
  children,
  ...props
}: React.ComponentProps<"div">) {
  const { navigationId } = React.useContext(RouteContentTransitionContext)

  return (
    <div
      key={navigationId}
      data-slot="route-content-transition"
      className={cn(navigationId > 0 && "route-content-transition", className)}
      {...props}
    >
      {children}
    </div>
  )
}
