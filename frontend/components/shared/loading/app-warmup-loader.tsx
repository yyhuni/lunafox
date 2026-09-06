"use client"

import { useTranslations } from "next-intl"

import { getLoadingOwnerAttributes } from "@/components/shared/loading/loading-owner"
import { cn } from "@/lib/utils"

type AppWarmupIntent = "app-shell" | "auth"

interface AppWarmupLoaderProps {
  owner: string
  intent: AppWarmupIntent
  label?: string
  className?: string
}

export function AppWarmupLoader({
  owner,
  intent,
  label,
  className,
}: AppWarmupLoaderProps) {
  const t = useTranslations("common.ui")
  const resolvedLabel = label ?? t("loading")
  const eyebrow = intent === "auth" ? "AUTH" : "APP"
  const layer = intent === "auth" ? "auth-shell" : "app-shell"

  if (!owner.trim()) {
    throw new Error("AppWarmupLoader requires a non-empty owner.")
  }

  return (
    <div
      {...getLoadingOwnerAttributes({ owner, layer, intent })}
      data-slot="app-warmup-loader"
      role="status"
      aria-live="polite"
      aria-label={resolvedLabel}
      className={cn(
        "app-warmup-loader bg-background text-foreground",
        className
      )}
    >
      <div className="app-warmup-loader__panel border border-border bg-card/90 shadow-sm">
        <div className="app-warmup-loader__eyebrow text-[10px] font-medium tracking-[0.32em] text-muted-foreground">
          {eyebrow}
        </div>
        <div className="mt-4 space-y-3">
          <div className="app-warmup-loader__line h-2.5 w-14 rounded-full" />
          <div className="app-warmup-loader__line h-6 w-40 rounded-full" />
          <div className="app-warmup-loader__line h-3 w-32 rounded-full" />
        </div>
        <div className="mt-6 flex items-center gap-3">
          <span
            aria-hidden="true"
            className="app-warmup-loader__pulse size-2 rounded-full bg-primary/70"
          />
          <span className="text-sm text-muted-foreground">{resolvedLabel}</span>
        </div>
      </div>
    </div>
  )
}
