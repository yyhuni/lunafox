"use client"

import React from "react"
import dynamic from "next/dynamic"
import { usePathname } from "next/navigation"
import { useLocale, useTranslations } from "next-intl"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { LoadingState } from "@/components/loading-spinner"
import { Suspense } from "react"
import { useAuth } from "@/hooks/use-auth"
import { useRouter } from "next/navigation"

const Toaster = dynamic(
  () => import("@/components/ui/sonner").then((mod) => mod.Toaster),
  { ssr: false }
)

const AppSidebar = dynamic(
  () => import("@/components/app-sidebar").then((mod) => mod.AppSidebar),
  { loading: () => null }
)

const UnifiedHeader = dynamic(
  () => import("@/components/unified-header").then((mod) => mod.UnifiedHeader),
  { loading: () => null }
)

// Public routes that don't require authentication (without locale prefix)
const PUBLIC_ROUTES = ["/login"]

interface AuthLayoutProps {
  children: React.ReactNode
}

/**
 * Check if the current path is a public route
 * Handles internationalized paths like /en/login, /zh/login
 */
function isPublicPath(pathname: string): boolean {
  // Remove locale prefix (e.g., /en/login -> /login, /zh/login -> /login)
  const pathWithoutLocale = pathname.replace(/^\/[a-z]{2}(?=\/|$)/, '')
  return PUBLIC_ROUTES.some((route) => 
    pathWithoutLocale === route || pathWithoutLocale.startsWith(`${route}/`)
  )
}

/**
 * Authentication layout component
 * Decides whether to show sidebar based on login status and route
 * 
 * New layout structure:
 * ┌─────────────────────────────────────────────────────────┐
 * │ Logo area (fixed width) │ Top bar content (search/notification/language, etc.) │
 * ├──────────────────────┼──────────────────────────────────┤
 * │Sidebar menu │Main content area │
 * │ (No Logo) │ │
 * └──────────────────────┴──────────────────────────────────┘
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  const pathname = usePathname()
  const router = useRouter()
  const { data: auth, isLoading } = useAuth()
  const tCommon = useTranslations("common")
  const locale = useLocale()
  const [hydrated, setHydrated] = React.useState(false)

  // Check if it's a public route (login page)
  const isPublicRoute = isPublicPath(pathname)

  React.useEffect(() => {
    setHydrated(true)
  }, [])

  // Redirect to login page if not authenticated (useEffect must be before all conditional returns)
  React.useEffect(() => {
    if (!hydrated) return
    if (!isLoading && !auth?.authenticated && !isPublicRoute) {
      const normalized = "/login/"
      const loginPath = `/${locale}${normalized}`
      router.replace(loginPath)
    }
  }, [auth, hydrated, isLoading, isPublicRoute, router, locale])

  // If it's login page, render content directly (without sidebar)
  if (isPublicRoute) {
    return (
      <>
        {children}
        <Toaster />
      </>
    )
  }

  // Keep first SSR/CSR frame stable on protected routes to avoid hydration mismatches.
  if (!hydrated) {
    return (
      <>
        <LoadingState active message="loading…" />
        <Toaster />
      </>
    )
  }

  if (!isLoading && !auth?.authenticated) {
    return <Toaster />
  }

  const showLoading = isLoading
  const shouldRenderApp = isLoading || !!auth?.authenticated

  // Authenticated - show full layout with unified header
  // Layout structure:
  // ┌─────────────────────────────────────────────────────────┐
  // │ Logo area (fixed width) │ Top bar content (search/notification/language, etc.) │
  // ├──────────────────────┼──────────────────────────────────┤
  // │Sidebar menu │Main content area │
  // │ (No Logo) │ │
  // └──────────────────────┴──────────────────────────────────┘
  return (
    <>
      <LoadingState active={showLoading} message="loading…" />
      {shouldRenderApp ? (
        <SidebarProvider
          className="animate-app-fade-in !min-h-0 flex flex-col h-svh"
          style={
            {
              "--sidebar-width": "calc(var(--spacing) * 62)",
              "--header-height": "calc(var(--spacing) * 12)",
            } as React.CSSProperties
          }
        >
          {/* Unified top bar - spans the entire page, including logo */}
          <UnifiedHeader />
          
          {/* Lower content area: sidebar + main content */}
          <div className="flex flex-1 min-h-0">
            <AppSidebar />
            <SidebarInset className="flex min-h-0 flex-col flex-1">
              <div className="flex flex-col flex-1 min-h-0 overflow-y-auto">
                <div className="@container/main flex-1 min-h-0 flex flex-col gap-2">
                  <Suspense fallback={<LoadingState message={tCommon("status.pageLoading")} />}>
                    {children}
                  </Suspense>
                </div>
              </div>
            </SidebarInset>
          </div>
        </SidebarProvider>
      ) : null}
      <Toaster />
    </>
  )
}
