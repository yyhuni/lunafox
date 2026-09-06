"use client"

import * as React from "react"
import { useTranslations } from "next-intl"

import { AppSidebar } from "@/components/app-sidebar"
import {
  protectedAppShellContentFrameClassName,
  protectedAppShellScrollAreaClassName,
  protectedAppShellStyle,
} from "@/components/auth/protected-app-shell"
import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { UnifiedHeader } from "@/components/unified-header"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { cn } from "@/lib/utils"

interface AppShellWarmupProps {
  owner: string
  delayMs?: number
  className?: string
}

export function AppShellWarmup({
  owner,
  delayMs,
  className,
}: AppShellWarmupProps) {
  const t = useTranslations("common.ui")
  const resolvedDelayMs = delayMs ?? 120
  const [isVisible, setIsVisible] = React.useState(resolvedDelayMs <= 0)

  if (!owner.trim()) {
    throw new Error("AppShellWarmup requires a non-empty owner.")
  }

  React.useEffect(() => {
    if (resolvedDelayMs <= 0) {
      setIsVisible(true)
      return
    }

    setIsVisible(false)
    const timer = window.setTimeout(() => {
      setIsVisible(true)
    }, resolvedDelayMs)
    return () => window.clearTimeout(timer)
  }, [resolvedDelayMs])

  if (!isVisible) {
    return null
  }

  return (
    <SidebarProvider
      {...getLoadingOwnerAttributes({ owner, layer: "app-shell", intent: "app-shell" })}
      data-slot="app-shell-warmup"
      role="status"
      aria-live="polite"
      aria-label={t("loading")}
      style={protectedAppShellStyle}
      className={cn("bg-background h-svh min-h-0 text-foreground pointer-events-none", className)}
    >
      <AppSidebar warmup />
      <div className="bg-background flex flex-1 min-h-0">
        <div className="flex flex-1 flex-col min-h-0">
          <UnifiedHeader warmup />
          <SidebarInset className="flex flex-1 flex-col min-h-0">
            <div className={protectedAppShellScrollAreaClassName}>
              <div className={protectedAppShellContentFrameClassName}>
                <div aria-hidden="true" className="flex flex-1" />
              </div>
            </div>
          </SidebarInset>
        </div>
      </div>
    </SidebarProvider>
  )
}
