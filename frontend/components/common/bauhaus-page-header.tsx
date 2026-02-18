"use client"

import { useEffect, useState } from "react"
import { useLocale, useTranslations } from "next-intl"
import type { LucideIcon } from "@/components/icons"
import { cn } from "@/lib/utils"

interface BauhausPageHeaderProps {
  /** Page code, such as "TGT-01" */
  code: string
  /** Subtitle or category, such as "Asset Management" */
  subtitle: string
  /** Main title */
  title: string
  /** Description text (optional) */
  description?: string
  /** Whether to show the description (default: false) */
  showDescription?: boolean
  /** Icon component */
  icon?: LucideIcon
  /** Whether to show the icon (default: false) */
  showIcon?: boolean
  /** Status text, default "ACTIVE" */
  statusText?: string
  /** Whether online, default true */
  isOnline?: boolean
  /** Custom class for outer container */
  className?: string
}

/**
 * Bauhaus-style page header component.
 * Shown only in the Bauhaus theme to provide a consistent industrial look.
 */
export function BauhausPageHeader({
  code,
  subtitle,
  title,
  description,
  showDescription = false,
  icon: Icon,
  showIcon = false,
  statusText,
  isOnline = true,
  className,
}: BauhausPageHeaderProps) {
  const locale = useLocale()
  const tUi = useTranslations("common.ui")
  const resolvedStatusText = statusText ?? tUi("activeStatus")
  const [currentTime, setCurrentTime] = useState<string>("")

  useEffect(() => {
    const updateTime = () => {
      const now = new Date()
      setCurrentTime(
        now.toLocaleTimeString(locale, {
          hour12: false,
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })
      )
    }

    let intervalId: number | null = null

    const shouldRun = () =>
      !document.hidden && document.documentElement.getAttribute("data-theme") === "bauhaus"

    const start = () => {
      updateTime()
      intervalId = window.setInterval(updateTime, 1000)
    }

    const stop = () => {
      if (intervalId) {
        clearInterval(intervalId)
        intervalId = null
      }
    }

    const sync = () => {
      if (shouldRun()) {
        if (!intervalId) start()
      } else {
        stop()
      }
    }

    sync()

    const observer = new MutationObserver(sync)
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] })
    document.addEventListener("visibilitychange", sync)

    return () => {
      document.removeEventListener("visibilitychange", sync)
      observer.disconnect()
      stop()
    }
  }, [locale])

  return (
    <div className={cn("hidden [[data-theme=bauhaus]_&]:block px-4 lg:px-6", className)}>
      <div className="bg-card border border-border border-t-2 border-t-primary p-5">
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
          {/* Left title area */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="px-1.5 py-0.5 text-[10px] font-mono bg-primary text-primary-foreground tracking-wider">
                {code}
              </span>
              <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                {subtitle}
              </p>
            </div>
            <h1 className="text-2xl font-bold tracking-tight uppercase flex items-center gap-3">
              {showIcon && Icon ? <Icon className="h-6 w-6" /> : null}
              {title}
            </h1>
            {showDescription && description ? (
              <p className="text-sm text-muted-foreground">{description}</p>
            ) : null}
          </div>

          {/* Right status bar */}
          <div className="flex gap-2">
            <div className="px-3 py-1.5 flex items-center gap-2 text-xs font-mono bg-secondary border border-border">
              <span
                className={`w-1.5 h-1.5 rounded-sm ${
                  isOnline ? "bg-[var(--success)]" : "bg-[var(--error)]"
                }`}
              />
              {tUi("statusLabel")}: {resolvedStatusText}
            </div>
            <div className="px-3 py-1.5 flex items-center gap-2 text-xs font-mono bg-secondary border border-border">
              <span className="text-muted-foreground">{tUi("cycleLabel")}:</span>
              {currentTime}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
