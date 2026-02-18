"use client"

import React from "react"
import { useRouter, usePathname } from "next/navigation"
import { useLocale } from "next-intl"
import { useAuth } from "@/hooks/use-auth"
import { LoadingState } from "@/components/loading-spinner"

// Public routes that don't require authentication
const PUBLIC_ROUTES = ["/login"]
// Skip authentication via environment variable (pnpm dev:noauth)
const SKIP_AUTH = process.env.NEXT_PUBLIC_SKIP_AUTH === 'true'

interface AuthGuardProps {
  children: React.ReactNode
}

/**
 * Authentication guard component
 * Protects routes that require login
 */
export function AuthGuard({ children }: AuthGuardProps) {
  const router = useRouter()
  const pathname = usePathname()
  const { data: auth, isLoading } = useAuth()
  const locale = useLocale()

  // Check if it's a public route
  const pathWithoutLocale = pathname.replace(/^\/[a-z]{2}(?=\/|$)/, '')
  const isPublicRoute = PUBLIC_ROUTES.some((route) => 
    pathWithoutLocale === route || pathWithoutLocale.startsWith(`${route}/`)
  )

  React.useEffect(() => {
    // Skip processing in skip auth mode
    if (SKIP_AUTH) return
    // Skip processing during loading or for public routes
    if (isLoading || isPublicRoute) return

    // Redirect to login page if not authenticated
    if (!auth?.authenticated) {
      const normalized = "/login/"
      const loginPath = `/${locale}${normalized}`
      router.push(loginPath)
    }
  }, [auth, isLoading, isPublicRoute, router, locale])

  // Skip auth mode
  if (SKIP_AUTH) {
    return <>{children}</>
  }

  // Show loading during authentication check
  if (isLoading) {
    return <LoadingState message="loading…" />
  }

  // Render public routes directly
  if (isPublicRoute) {
    return <>{children}</>
  }

  // Don't render content if not authenticated (waiting for redirect)
  if (!auth?.authenticated) {
    return <LoadingState message="loading…" />
  }

  // Render content if authenticated
  return <>{children}</>
}
