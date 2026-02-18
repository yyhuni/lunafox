"use client"

import { Suspense } from "react"
import dynamic from "next/dynamic"

const RoutePrefetch = dynamic(
  () => import("@/components/route-prefetch").then((mod) => mod.RoutePrefetch),
  { ssr: false }
)

const RouteProgress = dynamic(
  () => import("@/components/route-progress").then((mod) => mod.RouteProgress),
  { ssr: false }
)

/**
 * Client-only route UX enhancements.
 * Keep them out of the server layout to avoid eagerly coupling non-critical code.
 */
export function LayoutClientEnhancements() {
  return (
    <>
      <Suspense fallback={null}>
        <RouteProgress />
      </Suspense>
      <RoutePrefetch />
    </>
  )
}

