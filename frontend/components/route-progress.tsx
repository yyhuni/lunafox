"use client"

import { useEffect, useState, useCallback, useRef } from "react"
import { usePathname, useSearchParams } from "next/navigation"
import { cn } from "@/lib/utils"

/**
 * Route loading progress bar component
 * 
 * Monitors Next.js App Router route changes and displays top progress bar animation
 */
export function RouteProgress() {
  const pathname = usePathname()
  const searchParams = useSearchParams()
  const [progress, setProgress] = useState(0)
  const [isVisible, setIsVisible] = useState(false)
  const isFirstRender = useRef(true)

  const intervalRef = useRef<NodeJS.Timeout | null>(null)
  const manualTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const manualStartRef = useRef(false)

  const clearManualTimeout = useCallback(() => {
    if (manualTimeoutRef.current) {
      clearTimeout(manualTimeoutRef.current)
      manualTimeoutRef.current = null
    }
  }, [])

  const startProgress = useCallback(() => {
    clearManualTimeout()
    setIsVisible(true)
    setProgress(0)
    
    // Use interval for smooth increment
    let currentProgress = 0
    intervalRef.current = setInterval(() => {
      currentProgress += Math.random() * 10 + 5 // Increase by 5-15% each time
      if (currentProgress >= 90) {
        currentProgress = 90 // Max 90%, wait for completion
        if (intervalRef.current) {
          clearInterval(intervalRef.current)
          intervalRef.current = null
        }
      }
      setProgress(currentProgress)
    }, 100)
  }, [clearManualTimeout])

  const completeProgress = useCallback(() => {
    clearManualTimeout()
    // Clear ongoing interval
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
    
    setProgress(100)
    // Show 100% briefly after completion, then hide
    setTimeout(() => {
      setIsVisible(false)
      setProgress(0)
    }, 300)
  }, [clearManualTimeout])

  useEffect(() => {
    // Skip first render
    if (isFirstRender.current) {
      isFirstRender.current = false
      return
    }

    let timer: ReturnType<typeof setTimeout> | null = null

    if (manualStartRef.current) {
      manualStartRef.current = false
      clearManualTimeout()
      timer = setTimeout(() => completeProgress(), 300)
    } else {
      // Trigger progress bar on route change
      startProgress()
      // End progress bar after page load completes
      timer = setTimeout(() => completeProgress(), 300)
    }
    
    return () => {
      if (timer) {
        clearTimeout(timer)
      }
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [pathname, searchParams, startProgress, completeProgress, clearManualTimeout])

  useEffect(() => {
    const handleStart = () => {
      manualStartRef.current = true
      startProgress()
      manualTimeoutRef.current = setTimeout(() => {
        manualStartRef.current = false
        completeProgress()
      }, 1200)
    }

    window.addEventListener("lunafox:route-progress-start", handleStart)
    return () => {
      window.removeEventListener("lunafox:route-progress-start", handleStart)
      clearManualTimeout()
    }
  }, [startProgress, completeProgress, clearManualTimeout])

  if (!isVisible) return null

  return (
    <div
      className={cn(
        "fixed top-0 left-0 right-0 z-[99999] h-[3px]",
        "pointer-events-none"
      )}
    >
      {/* Progress bar background */}
      <div className="absolute inset-0 bg-transparent" />
      
      {/* Progress bar */}
      <div
        className={cn(
          "h-full bg-[var(--highlight)] transition-[width] duration-200 ease-out",
          "shadow-[0_0_10px_var(--highlight)]"
        )}
        style={{ width: `${progress}%` }}
      />
      
      {/* Glow effect */}
      <div
        className={cn(
          "absolute top-0 right-0 h-full w-24",
          "bg-gradient-to-r from-transparent to-[var(--highlight)]",
          "opacity-50 blur-sm",
          "transition-[left,transform] duration-200"
        )}
        style={{ 
          transform: `translateX(${progress < 100 ? '0' : '100%'})`,
          left: `${Math.max(0, progress - 10)}%`
        }}
      />
    </div>
  )
}
