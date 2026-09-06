"use client"

import React from "react"
import { usePathname, useRouter, useSearchParams } from "next/navigation"

import { useAuth } from "@/hooks/use-auth"
import { replaceWithRouteProgress } from "@/components/route-progress"
import {
  buildLoginRedirectPath,
  buildReturnTo,
  isPublicPathname,
} from "@/lib/auth-runtime.mjs"

interface AuthLayoutProps {
  children: React.ReactNode
}

const ProtectedAuthLayout = React.lazy(() => import("@/components/auth/protected-auth-layout").then((mod) => ({
  default: mod.ProtectedAuthLayout,
})))

function AuthBootHandoffBlocker() {
  return (
    <div
      data-boot-handoff-pending="true"
      data-testid="auth-boot-handoff-blocker"
      hidden
      aria-hidden="true"
    />
  )
}

/**
 * Authentication layout component
 *
 * State machine:
 *   public route           → render children directly (no shell)
 *   authenticated === true → render ProtectedAuthLayout (app shell + route suspense)
 *   auth unresolved        → keep the global boot layer as owner, never protected shell
 *   authenticated === false → trigger login redirect, keep the global boot layer as owner
 *
 * Layout structure (authenticated):
 * ┌──────────────┬──────────────────────────────────────────┐
 * │ Sidebar      │ Top bar actions                          │
 * │ with logo    ├──────────────────────────────────────────┤
 * │              │ Main content area                        │
 * └──────────────┴──────────────────────────────────────────┘
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const router = useRouter()
  const { data: auth, isLoading } = useAuth()
  const [hydrated, setHydrated] = React.useState(false)
  const redirectPath = React.useMemo(
    () => buildLoginRedirectPath(buildReturnTo(pathname, searchParams.toString())),
    [pathname, searchParams]
  )

  const isPublicRoute = isPublicPathname(pathname)

  React.useEffect(() => {
    setHydrated(true)
  }, [])

  // Redirect to login page if not authenticated.
  // Must fire before any conditional return so the effect dependency order stays stable.
  React.useEffect(() => {
    if (!hydrated) return
    if (!isLoading && !auth?.authenticated && !isPublicRoute) {
      replaceWithRouteProgress(router, redirectPath)
    }
  }, [auth, hydrated, isLoading, isPublicRoute, redirectPath, router])

  // Public route — no auth gate, no shell
  if (isPublicRoute) {
    return children
  }

  // Auth unresolved or redirect pending: keep the server boot layer visible until
  // login or the authenticated app shell can provide a real first-frame owner.
  if (!hydrated || isLoading || !auth?.authenticated) {
    return <AuthBootHandoffBlocker />
  }

  // Keep boot visible until the protected shell provides the route's real owner.
  return (
    <React.Suspense fallback={<AuthBootHandoffBlocker />}>
      <ProtectedAuthLayout>
        {children}
      </ProtectedAuthLayout>
    </React.Suspense>
  )
}
