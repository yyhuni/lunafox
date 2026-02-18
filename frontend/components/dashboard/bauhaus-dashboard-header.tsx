"use client"

import { useEffect, useState } from "react"
import { useLocale, useTranslations } from "next-intl"

/**
 * Bauhaus-style dashboard header.
 * Shown only in the Bauhaus theme, matching the dashboard-demo prototype style.
 */
export function BauhausDashboardHeader() {
  const locale = useLocale()
  const t = useTranslations("dashboard.bauhausHeader")
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
    <div className="hidden [[data-theme=bauhaus]_&]:block px-4 lg:px-6">
      <div className="bg-card border border-border border-t-2 border-t-primary p-5">
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-4">
          {/* Left title area */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="px-1.5 py-0.5 text-[10px] font-mono bg-primary text-primary-foreground tracking-wider">
                DASH-01
              </span>
              <p className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                {t("operationsCommand")}
              </p>
            </div>
            <h1 className="text-2xl font-bold tracking-tight uppercase">
              {t("systemOverview")}
            </h1>
          </div>

          {/* Right status bar */}
          <div className="flex gap-2">
            <div className="px-3 py-1.5 flex items-center gap-2 text-xs font-mono bg-secondary border border-border">
              <span className="w-1.5 h-1.5 bg-[var(--success)] rounded-sm" />
              {t("networkOnline")}
            </div>
            <div className="px-3 py-1.5 flex items-center gap-2 text-xs font-mono bg-secondary border border-border">
              <span className="text-muted-foreground">{t("cycleLabel")}</span>
              {currentTime}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
