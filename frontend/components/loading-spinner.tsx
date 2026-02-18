"use client"

import React from "react"
import { cn } from "@/lib/utils"
import { Spinner } from "@/components/ui/spinner"
import { ShieldLoader } from "@/components/ui/shield-loader"

interface LoadingSpinnerProps {
  size?: "sm" | "md" | "lg"
  className?: string
}

/**
 * Unified loading animation component
 * 
 * Features:
 * - Three sizes: sm(16px), md(24px), lg(32px)
 * - Supports custom styles
 * - Uses Tailwind CSS animations
 */
export function LoadingSpinner({ size = "sm", className }: LoadingSpinnerProps) {
  const sizeMap = {
    sm: "size-4",
    md: "size-6", 
    lg: "size-8"
  }

  return <Spinner className={cn(sizeMap[size], className)} />
}

interface LoadingStateProps {
  message?: string
  size?: "sm" | "md" | "lg"
  className?: string
  active?: boolean
  fadeMs?: number
}

/**
 * Loading state component with text
 * 
 * Used for page-level loading state display
 */
export function LoadingState({ 
  message, 
  size = "md", 
  className,
  active = true,
  fadeMs = 250,
}: LoadingStateProps) {
  const [shouldRender, setShouldRender] = React.useState(active)
  const [visible, setVisible] = React.useState(active)

  React.useEffect(() => {
    if (active) {
      setShouldRender(true)
      const raf = requestAnimationFrame(() => setVisible(true))
      return () => cancelAnimationFrame(raf)
    }

    setVisible(false)
    const timer = window.setTimeout(() => {
      setShouldRender(false)
    }, fadeMs)

    return () => window.clearTimeout(timer)
  }, [active, fadeMs])

  if (!shouldRender) return null

  return (
    <div
      className={cn(
        "flex items-center justify-center min-h-[200px] h-screen w-full transition-opacity",
        visible ? "opacity-100 pointer-events-auto" : "opacity-0 pointer-events-none",
        className
      )}
      style={{ transitionDuration: `${fadeMs}ms` }}
    >
      <div className="flex flex-col items-center space-y-4">
        <ShieldLoader size={size} />
        {message ? <p className="text-sm text-muted-foreground">{message}</p> : null}
      </div>
    </div>
  )
}


interface LoadingOverlayProps {
  isLoading: boolean
  message?: string
  children: React.ReactNode
}

/**
 * Loading overlay component
 * 
 * Displays loading overlay on existing content
 */
export function LoadingOverlay({ 
  isLoading, 
  message, 
  children 
}: LoadingOverlayProps) {
  return (
    <div className="relative">
      {children}
      {isLoading && (
        <div className="absolute inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="flex flex-col items-center space-y-2">
            <LoadingSpinner size="lg" />
            <p className="text-sm text-muted-foreground">{message}</p>
          </div>
        </div>
      )}
    </div>
  )
}
