"use client"

import React, { Suspense } from "react"

import { AppSidebar } from "@/components/app-sidebar"
import { UnifiedHeader } from "@/components/unified-header"
import { useSessionRenewal } from "@/hooks/use-session-renewal"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { ContentReveal } from "@/components/shared/loading/content-reveal"
import {
  protectedAppShellContentFrameClassName,
  protectedAppShellScrollAreaClassName,
  protectedAppShellStyle,
} from "@/components/auth/protected-app-shell"

interface ProtectedAuthLayoutProps {
  children: React.ReactNode
}

function RouteBootHandoffBlocker() {
  return <div data-boot-handoff-pending="true" hidden aria-hidden="true" />
}

export function ProtectedAuthLayout({
  children,
}: ProtectedAuthLayoutProps) {
  useSessionRenewal()

  // Auth gate is handled by AuthLayout — this component only renders
  // after authentication is confirmed. The global boot layer remains visible
  // until the route's own visible owner has committed.
  //
  // Layout structure:
  // ┌──────────────┬──────────────────────────────────────────┐
  // │ Sidebar      │ Top bar actions                          │
  // │ with logo    ├──────────────────────────────────────────┤
  // │              │ Main content area                        │
  // └──────────────┴──────────────────────────────────────────┘
  // Route ContentReveal and workspace ContentHandoff own entry motion; the
  // shell stays static so hard reloads do not replay two page-level reveals.
  return (
    <SidebarProvider
      className="flex h-svh min-h-0 w-full"
      style={protectedAppShellStyle}
    >
      <AppSidebar />
      <div className="bg-background flex flex-1 min-h-0">
        <div className="flex flex-1 flex-col min-h-0">
          <UnifiedHeader />
          <SidebarInset className="flex flex-1 flex-col min-h-0">
            <div className={protectedAppShellScrollAreaClassName}>
              <div className={protectedAppShellContentFrameClassName}>
                <Suspense fallback={<RouteBootHandoffBlocker />}>
                  <ContentReveal owner="auth-layout-route-content" className="flex min-h-0 flex-1 flex-col">
                    {children}
                  </ContentReveal>
                </Suspense>
              </div>
            </div>
          </SidebarInset>
        </div>
      </div>
    </SidebarProvider>
  )
}
