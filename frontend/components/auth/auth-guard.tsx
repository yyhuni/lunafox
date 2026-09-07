"use client"

import React from "react"
import { usePathname, useSearchParams } from "next/navigation"
import { useAuth } from "@/hooks/use-auth"
import { AppShellWarmup } from "@/components/shared/loading/app-shell-warmup"
import {
  buildLoginRedirectPath,
  buildReturnTo,
  isPublicPathname,
} from "@/lib/auth-runtime.mjs"

// Skip authentication via environment variable (pnpm dev:noauth)
const SKIP_AUTH = process.env.NEXT_PUBLIC_SKIP_AUTH === 'true'
const AUTH_GUARD_WARMUP_DELAY_MS = 180

interface AuthGuardProps {
  children: React.ReactNode
}

/**
 * Authentication guard component
 * Protects routes that require login
 */
export function AuthGuard({ children }: AuthGuardProps) {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const { data: auth, isLoading } = useAuth()
  const redirectPath = React.useMemo(
    () => buildLoginRedirectPath(buildReturnTo(pathname, searchParams.toString())),
    [pathname, searchParams]
  )

  // Check if it's a public route
  const isPublicRoute = isPublicPathname(pathname)

  React.useEffect(() => {
    // Skip processing in skip auth mode
    if (SKIP_AUTH) return
    // Skip processing during loading or for public routes
    if (isLoading || isPublicRoute) return

    // Redirect to login page if not authenticated
    if (!auth?.authenticated) {
      window.location.replace(redirectPath)
    }
  }, [auth, isLoading, isPublicRoute, redirectPath])

  // Skip auth mode
  if (SKIP_AUTH) {
    return <>{children}</>
  }

  // Show loading during authentication check
  if (isLoading) {
    return <AppShellWarmup owner="auth-guard-check" delayMs={AUTH_GUARD_WARMUP_DELAY_MS} />
  }

  // Render public routes directly
  if (isPublicRoute) {
    return <>{children}</>
  }

  // Don't render content if not authenticated (waiting for redirect)
  if (!auth?.authenticated) {
    return <AppShellWarmup owner="auth-guard-redirect" delayMs={AUTH_GUARD_WARMUP_DELAY_MS} />
  }

  // Render content if authenticated
  return <>{children}</>
}
